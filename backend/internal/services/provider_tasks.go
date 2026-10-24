package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type ExternalTaskProvider struct {
	assetEndpoint string
	videoEndpoint string
	audioEndpoint string
	apiKey        string
	client        *http.Client
}

func NewExternalTaskProvider(assetEndpoint string, videoEndpoint string, audioEndpoint string, apiKey string, timeout time.Duration) *ExternalTaskProvider {
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	return &ExternalTaskProvider{
		assetEndpoint: assetEndpoint,
		videoEndpoint: videoEndpoint,
		audioEndpoint: audioEndpoint,
		apiKey:        apiKey,
		client:        &http.Client{Timeout: timeout},
	}
}

func (p *ExternalTaskProvider) GenerateAsset(ctx context.Context, payload map[string]any) (map[string]any, error) {
	if p.assetEndpoint == "" {
		if p.canUseDashScope() {
			return p.generateDashScopeImage(ctx, payload)
		}
		return nil, errors.New("ASSET_PROVIDER_ENDPOINT is not configured and DASHSCOPE_API_KEY is not configured")
	}
	return p.postProvider(ctx, p.assetEndpoint, payload)
}

func (p *ExternalTaskProvider) GenerateVideo(ctx context.Context, payload map[string]any) (map[string]any, error) {
	if p.videoEndpoint == "" {
		if p.canUseDashScope() {
			return p.generateDashScopeVideo(ctx, payload)
		}
		return nil, errors.New("VIDEO_PROVIDER_ENDPOINT is not configured and DASHSCOPE_API_KEY is not configured")
	}
	return p.postProvider(ctx, p.videoEndpoint, payload)
}

func (p *ExternalTaskProvider) GenerateAudio(ctx context.Context, payload map[string]any) (map[string]any, error) {
	if p.audioEndpoint == "" {
		return nil, errors.New("AUDIO_PROVIDER_ENDPOINT is not configured")
	}
	return p.postProvider(ctx, p.audioEndpoint, payload)
}

func (p *ExternalTaskProvider) canUseDashScope() bool {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("AI_PROVIDER")))
	return provider == "" || provider == "dashscope" || p.dashScopeAPIKey() != ""
}

func (p *ExternalTaskProvider) generateDashScopeImage(ctx context.Context, payload map[string]any) (map[string]any, error) {
	prompt := strings.TrimSpace(stringFromAny(payload["prompt"]))
	if prompt == "" {
		return nil, errors.New("prompt is required")
	}
	model := strings.TrimSpace(stringFromAny(payload["model"]))
	if model == "" {
		model = envString("DEFAULT_IMAGE_MODEL", "wan2.7-image-pro")
	}
	parameters := mapFromAny(payload["parameters"])
	size := stringParam(parameters, "size", "")
	if size == "" {
		size = sizeForAspectRatio(stringParam(parameters, "aspect_ratio", stringParam(parameters, "ratio", "")), "1280*1280")
	}

	content := []map[string]string{}
	references := collectStrings(payload, "references", "reference_image_urls", "ref_image_urls")
	if inputs := mapFromAny(payload["inputs"]); inputs != nil {
		references = append(references, collectStrings(inputs, "references", "reference_image_urls", "ref_image_urls")...)
	}
	for _, ref := range references {
		if ref != "" {
			content = append(content, map[string]string{"image": ref})
		}
	}
	content = append(content, map[string]string{"text": prompt})

	dashPayload := map[string]any{
		"model": model,
		"input": map[string]any{
			"messages": []map[string]any{
				{
					"role":    "user",
					"content": content,
				},
			},
		},
		"parameters": map[string]any{
			"prompt_extend": boolParam(parameters, "prompt_extend", true),
			"watermark":     boolParam(parameters, "watermark", false),
			"n":             intParam(parameters, "n", 1),
			"size":          size,
		},
	}
	if negative := stringParam(parameters, "negative_prompt", ""); negative != "" {
		dashPayload["parameters"].(map[string]any)["negative_prompt"] = negative
	}
	if seed := intParam(parameters, "seed", 0); seed > 0 {
		dashPayload["parameters"].(map[string]any)["seed"] = seed
	}

	url, taskID, err := p.runDashScopeTask(ctx, "/api/v1/services/aigc/image-generation/generation", dashPayload, "image")
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"image_url":        url,
		"url":              url,
		"provider":         "dashscope",
		"model":            model,
		"provider_task_id": taskID,
	}, nil
}

func (p *ExternalTaskProvider) generateDashScopeVideo(ctx context.Context, payload map[string]any) (map[string]any, error) {
	prompt := strings.TrimSpace(stringFromAny(payload["prompt"]))
	if prompt == "" {
		return nil, errors.New("prompt is required")
	}
	model := strings.TrimSpace(stringFromAny(payload["model"]))
	if model == "" {
		model = envString("DEFAULT_VIDEO_MODEL", "wan2.7-i2v")
	}
	parameters := mapFromAny(payload["parameters"])
	inputs := mapFromAny(payload["inputs"])
	mode := strings.ToLower(strings.TrimSpace(stringFromAny(payload["generation_mode"])))
	if mode == "" && inputs != nil {
		mode = strings.ToLower(strings.TrimSpace(stringFromAny(inputs["generation_mode"])))
	}
	if mode == "" {
		mode = inferVideoMode(model)
	}

	duration := intParam(payload, "duration", intParam(parameters, "duration", 5))
	generateAudio := boolParam(payload, "generate_audio", boolParam(parameters, "audio", true))
	watermark := boolParam(parameters, "watermark", false)
	seed := intParam(payload, "seed", intParam(parameters, "seed", 0))
	ratio := stringParam(parameters, "ratio", stringParam(parameters, "aspect_ratio", "9:16"))
	resolution := stringParam(parameters, "resolution", stringFromAny(payload["resolution"]))
	if resolution == "" {
		resolution = "720P"
	}

	var dashPayload map[string]any
	switch {
	case mode == "r2v" || strings.Contains(model, "-r2v"):
		refs := collectStrings(payload, "reference_image_urls", "ref_image_urls")
		refs = append(refs, collectStrings(payload, "reference_video_urls", "ref_video_urls")...)
		if inputs != nil {
			refs = append(refs, collectStrings(inputs, "reference_image_urls", "ref_image_urls", "reference_video_urls", "ref_video_urls")...)
		}
		if imageURL := firstPayloadString(payload, inputs, "image_url", "img_url"); imageURL != "" {
			refs = append(refs, imageURL)
		}
		if len(refs) == 0 {
			return nil, errors.New("reference_image_urls or reference_video_urls is required for R2V generation")
		}
		inputKey := "reference_image_urls"
		if strings.Contains(model, "wan2.6-r2v") || strings.Contains(model, "wan2.5-r2v") {
			inputKey = "reference_video_urls"
		}
		dashPayload = map[string]any{
			"model": model,
			"input": map[string]any{
				"prompt": prompt,
				inputKey: refs,
			},
			"parameters": map[string]any{
				"duration":  duration,
				"audio":     generateAudio,
				"shot_type": stringParam(parameters, "shot_type", "multi"),
			},
		}
		if strings.HasPrefix(model, "wan2.7-") {
			dashPayload["parameters"].(map[string]any)["ratio"] = ratio
		} else {
			dashPayload["parameters"].(map[string]any)["size"] = sizeForAspectRatio(ratio, "1280*720")
		}
	case mode == "t2v" || strings.Contains(model, "-t2v"):
		dashPayload = map[string]any{
			"model": model,
			"input": map[string]any{
				"prompt": prompt,
			},
			"parameters": map[string]any{
				"resolution": resolution,
				"duration":   duration,
				"watermark":  watermark,
				"ratio":      ratio,
			},
		}
	case strings.HasPrefix(model, "happyhorse-") || strings.Contains(model, "kling-") || strings.Contains(model, "vidu") || strings.Contains(model, "pixverse"):
		imageURL := firstPayloadString(payload, inputs, "image_url", "img_url")
		if imageURL == "" {
			return nil, errors.New("image_url is required for I2V generation")
		}
		dashPayload = map[string]any{
			"model": model,
			"input": map[string]any{
				"prompt": prompt,
				"media":  []map[string]string{{"type": "first_frame", "url": imageURL}},
			},
			"parameters": map[string]any{
				"resolution": resolution,
				"duration":   duration,
				"watermark":  watermark,
			},
		}
	default:
		imageURL := firstPayloadString(payload, inputs, "image_url", "img_url")
		if imageURL == "" {
			return nil, errors.New("image_url is required for I2V generation")
		}
		videoInput := map[string]any{
			"prompt": prompt,
		}
		if strings.HasPrefix(model, "wan2.7-") {
			videoInput["media"] = []map[string]string{{"type": "first_frame", "url": imageURL}}
		} else {
			videoInput["img_url"] = imageURL
		}
		dashPayload = map[string]any{
			"model": model,
			"input": videoInput,
			"parameters": map[string]any{
				"duration":      duration,
				"prompt_extend": boolParam(parameters, "prompt_extend", true),
				"watermark":     watermark,
				"audio":         generateAudio,
				"shot_type":     stringParam(parameters, "shot_type", "single"),
			},
		}
		if strings.HasPrefix(model, "wan2.7-") {
			dashPayload["parameters"].(map[string]any)["ratio"] = ratio
		} else {
			dashPayload["parameters"].(map[string]any)["resolution"] = resolution
		}
		if negative := stringParam(parameters, "negative_prompt", ""); negative != "" {
			dashPayload["input"].(map[string]any)["negative_prompt"] = negative
		}
	}
	if seed > 0 {
		dashPayload["parameters"].(map[string]any)["seed"] = seed
	}

	url, taskID, err := p.runDashScopeTask(ctx, "/api/v1/services/aigc/video-generation/video-synthesis", dashPayload, "video")
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"video_url":        url,
		"url":              url,
		"provider":         "dashscope",
		"model":            model,
		"provider_task_id": taskID,
	}, nil
}

func (p *ExternalTaskProvider) runDashScopeTask(ctx context.Context, path string, payload map[string]any, resultKind string) (string, string, error) {
	apiKey := p.dashScopeAPIKey()
	if apiKey == "" {
		return "", "", errors.New("DASHSCOPE_API_KEY is not configured")
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}
	createURL := p.dashScopeBaseURL() + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, bytes.NewReader(encoded))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("X-DashScope-Async", "enable")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("DashScope task creation failed: status=%d body=%s", resp.StatusCode, string(body))
	}
	var created map[string]any
	if err := json.Unmarshal(body, &created); err != nil {
		return "", "", fmt.Errorf("decode DashScope create response: %w", err)
	}
	taskID := stringParam(mapFromAny(created["output"]), "task_id", "")
	if taskID == "" {
		return "", "", fmt.Errorf("DashScope create response missing task_id: %s", string(body))
	}

	resultURL, err := p.pollDashScopeTask(ctx, taskID, resultKind)
	if err != nil {
		return "", taskID, err
	}
	return resultURL, taskID, nil
}

func (p *ExternalTaskProvider) pollDashScopeTask(ctx context.Context, taskID string, resultKind string) (string, error) {
	interval := time.Duration(envInt("DASHSCOPE_POLL_INTERVAL_MS", 10000)) * time.Millisecond
	if interval <= 0 {
		interval = 10 * time.Second
	}
	maxWait := time.Duration(envInt("DASHSCOPE_MAX_WAIT_SECONDS", 900)) * time.Second
	deadline := time.Now().Add(maxWait)
	for {
		if time.Now().After(deadline) {
			return "", fmt.Errorf("DashScope task %s timed out after %s", taskID, maxWait)
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.dashScopeBaseURL()+"/api/v1/tasks/"+taskID, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+p.dashScopeAPIKey())
		resp, err := p.client.Do(req)
		if err != nil {
			return "", err
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return "", readErr
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("DashScope task poll failed: status=%d body=%s", resp.StatusCode, string(body))
		}
		var polled map[string]any
		if err := json.Unmarshal(body, &polled); err != nil {
			return "", fmt.Errorf("decode DashScope poll response: %w", err)
		}
		output := mapFromAny(polled["output"])
		status := stringParam(output, "task_status", "")
		switch status {
		case "SUCCEEDED":
			return extractDashScopeResultURL(output, resultKind)
		case "FAILED":
			return "", fmt.Errorf("DashScope task failed: %s %s", stringParam(output, "code", ""), stringParam(output, "message", ""))
		case "CANCELED", "UNKNOWN":
			return "", fmt.Errorf("DashScope task %s: %s", status, string(body))
		}
	}
}

func extractDashScopeResultURL(output map[string]any, resultKind string) (string, error) {
	if resultKind == "video" {
		if url := stringParam(output, "video_url", ""); url != "" {
			return url, nil
		}
		return "", errors.New("DashScope video task response missing video_url")
	}
	choices, ok := output["choices"].([]any)
	if !ok || len(choices) == 0 {
		return "", errors.New("DashScope image task response missing choices")
	}
	choice := mapFromAny(choices[0])
	message := mapFromAny(choice["message"])
	content, ok := message["content"].([]any)
	if !ok || len(content) == 0 {
		return "", errors.New("DashScope image task response missing message content")
	}
	item := mapFromAny(content[0])
	if url := stringParam(item, "image", ""); url != "" {
		return url, nil
	}
	return "", errors.New("DashScope image task response missing image URL")
}

func (p *ExternalTaskProvider) dashScopeAPIKey() string {
	if value := strings.TrimSpace(os.Getenv("DASHSCOPE_API_KEY")); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("LLM_API_KEY")); value != "" {
		return value
	}
	return strings.TrimSpace(p.apiKey)
}

func (p *ExternalTaskProvider) dashScopeBaseURL() string {
	base := strings.TrimSpace(os.Getenv("DASHSCOPE_BASE_URL"))
	if base == "" {
		base = "https://dashscope.aliyuncs.com"
	}
	return strings.TrimRight(base, "/")
}

func (p *ExternalTaskProvider) postProvider(ctx context.Context, endpoint string, payload map[string]any) (map[string]any, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("provider request failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode provider response: %w", err)
	}
	return result, nil
}

func mapFromAny(value any) map[string]any {
	if value == nil {
		return nil
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func collectStrings(values map[string]any, keys ...string) []string {
	if values == nil {
		return nil
	}
	var result []string
	for _, key := range keys {
		result = append(result, stringSliceFromAny(values[key])...)
	}
	return compactStrings(result)
}

func stringSliceFromAny(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(stringFromAny(item)); text != "" {
				values = append(values, text)
			}
		}
		return values
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{typed}
	default:
		return nil
	}
}

func compactStrings(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func firstPayloadString(payload map[string]any, inputs map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(stringFromAny(payload[key])); value != "" {
			return value
		}
		if inputs != nil {
			if value := strings.TrimSpace(stringFromAny(inputs[key])); value != "" {
				return value
			}
		}
	}
	return ""
}

func stringParam(values map[string]any, key string, fallback string) string {
	if values == nil {
		return fallback
	}
	if value := strings.TrimSpace(stringFromAny(values[key])); value != "" {
		return value
	}
	return fallback
}

func intParam(values map[string]any, key string, fallback int) int {
	if values == nil {
		return fallback
	}
	return intFromAny(values[key], fallback)
}

func boolParam(values map[string]any, key string, fallback bool) bool {
	if values == nil {
		return fallback
	}
	switch value := values[key].(type) {
	case bool:
		return value
	case string:
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	default:
		return fallback
	}
	return fallback
}

func envString(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func sizeForAspectRatio(ratio string, fallback string) string {
	switch strings.TrimSpace(ratio) {
	case "9:16", "720*1280":
		return "720*1280"
	case "16:9", "1280*720":
		return "1280*720"
	case "4:3":
		return "1024*768"
	case "3:4":
		return "768*1024"
	case "1:1", "":
		return "1280*1280"
	default:
		if strings.Contains(ratio, "*") {
			return ratio
		}
		return fallback
	}
}

func inferVideoMode(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(model, "-r2v"):
		return "r2v"
	case strings.Contains(model, "-t2v"):
		return "t2v"
	default:
		return "i2v"
	}
}
