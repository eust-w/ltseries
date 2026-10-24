package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"ltseries/backend/internal/models"
)

type LLMAnalyzer struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func NewLLMAnalyzer(baseURL string, apiKey string, model string) *LLMAnalyzer {
	return &LLMAnalyzer{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (a *LLMAnalyzer) Analyze(ctx context.Context, project *models.Project) (*models.ScriptAnalysis, error) {
	apiKey := envString("LLM_API_KEY", envString("DASHSCOPE_API_KEY", a.apiKey))
	baseURL := strings.TrimRight(envString("LLM_BASE_URL", a.baseURL), "/")
	model := envString("LLM_MODEL", a.model)
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	if model == "" {
		model = "qwen-plus"
	}
	if apiKey == "" {
		return nil, errors.New("LLM_API_KEY is not configured")
	}
	if baseURL == "" {
		return nil, errors.New("LLM_BASE_URL is not configured")
	}
	if model == "" {
		return nil, errors.New("LLM_MODEL is not configured")
	}

	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": analysisSystemPrompt(),
			},
			{
				"role":    "user",
				"content": fmt.Sprintf("Title: %s\nScript:\n%s", project.Title, project.Script),
			},
		},
		"temperature": 0.2,
		"response_format": map[string]string{
			"type": "json_object",
		},
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("LLM request failed: status=%d body=%s", resp.StatusCode, string(responseBody))
	}

	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return nil, fmt.Errorf("decode LLM response: %w", err)
	}
	if len(completion.Choices) == 0 || strings.TrimSpace(completion.Choices[0].Message.Content) == "" {
		return nil, errors.New("LLM returned no analysis content")
	}

	var analysis models.ScriptAnalysis
	content := stripJSONFence(completion.Choices[0].Message.Content)
	if err := json.Unmarshal([]byte(content), &analysis); err != nil {
		return nil, fmt.Errorf("decode analysis JSON: %w; raw=%s", err, content)
	}
	normalizeAnalysis(&analysis)
	return &analysis, nil
}

func analysisSystemPrompt() string {
	return strings.Join([]string{
		"你是专业短剧 showrunner、美术指导和分镜导演。",
		"只返回严格 JSON，不要使用 markdown 代码块，不要输出解释。",
		"把剧本解析成可生产的短剧项目数据：故事梗概、美术方向、角色、场景、道具和可人工调整的分镜。",
		"所有面向用户显示的字段必须使用中文，包括 synopsis、角色 name/description、场景 name/description、道具 name/description、分镜 action/camera/dialogue/prompt、美术推荐 name/description/positive_prompt/negative_prompt。",
		"不要把角色描述、镜头动作、镜头语言写成英文；即使使用电影术语，也要用中文表达。",
		"分镜 prompt 也优先使用中文，可少量保留通用模型关键词，但不得整段英文。",
		"角色、场景、道具描述要稳定，能直接作为图片生成提示词使用；分镜要覆盖完整剧情，而不是只生成一个镜头；短剧至少拆成 4 个镜头，剧情较长时拆成 6-12 个镜头。",
		"JSON schema: {\"synopsis\":\"...\",\"art_direction\":{\"selected_style_id\":\"cinematic\",\"style_config\":{\"positive_prompt\":\"...\",\"negative_prompt\":\"...\",\"palette\":[\"#000000\"],\"camera_language\":\"...\"},\"custom_styles\":[],\"ai_recommendations\":[{\"id\":\"...\",\"name\":\"...\",\"description\":\"...\",\"positive_prompt\":\"...\",\"negative_prompt\":\"...\"}]},\"characters\":[{\"name\":\"...\",\"description\":\"外貌、服装、年龄、气质、连续性细节\"}],\"scenes\":[{\"name\":\"...\",\"description\":\"地点、时间、光线、置景\"}],\"props\":[{\"name\":\"...\",\"description\":\"视觉细节和叙事功能\"}],\"shots\":[{\"index\":1,\"scene_id\":\"\",\"character_ids\":[],\"action\":\"...\",\"camera\":\"...\",\"dialogue\":\"...\",\"prompt\":\"包含风格和连续性要求的中文图像/视频提示词\"}]}",
	}, " ")
}

func (a *LLMAnalyzer) CompleteText(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	apiKey := envString("LLM_API_KEY", envString("DASHSCOPE_API_KEY", a.apiKey))
	baseURL := strings.TrimRight(envString("LLM_BASE_URL", a.baseURL), "/")
	model := envString("LLM_MODEL", a.model)
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	if model == "" {
		model = "qwen-plus"
	}
	if apiKey == "" {
		return "", errors.New("LLM_API_KEY is not configured")
	}
	if strings.TrimSpace(userPrompt) == "" {
		return "", errors.New("prompt is required")
	}
	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.35,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(encoded))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("LLM request failed: status=%d body=%s", resp.StatusCode, string(responseBody))
	}
	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return "", fmt.Errorf("decode LLM response: %w", err)
	}
	if len(completion.Choices) == 0 || strings.TrimSpace(completion.Choices[0].Message.Content) == "" {
		return "", errors.New("LLM returned no content")
	}
	return strings.TrimSpace(completion.Choices[0].Message.Content), nil
}

func stripJSONFence(content string) string {
	trimmed := strings.TrimSpace(content)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	return strings.TrimSpace(trimmed)
}

func normalizeAnalysis(analysis *models.ScriptAnalysis) {
	for i := range analysis.Characters {
		if analysis.Characters[i].ID == "" {
			analysis.Characters[i].ID = "char_" + uuid.NewString()
		}
		normalizeEntity(&analysis.Characters[i])
	}
	for i := range analysis.Scenes {
		if analysis.Scenes[i].ID == "" {
			analysis.Scenes[i].ID = "scene_" + uuid.NewString()
		}
		normalizeEntity(&analysis.Scenes[i])
	}
	for i := range analysis.Props {
		if analysis.Props[i].ID == "" {
			analysis.Props[i].ID = "prop_" + uuid.NewString()
		}
		normalizeEntity(&analysis.Props[i])
	}
	characterAliases := entityAliasMap(analysis.Characters)
	sceneAliases := entityAliasMap(analysis.Scenes)
	for i := range analysis.Shots {
		if analysis.Shots[i].ID == "" {
			analysis.Shots[i].ID = "shot_" + uuid.NewString()
		}
		if analysis.Shots[i].Index == 0 {
			analysis.Shots[i].Index = i + 1
		}
		if analysis.Shots[i].Status == "" {
			analysis.Shots[i].Status = models.GenerationStatusPending
		}
		if analysis.Shots[i].ImageAsset.Variants == nil {
			analysis.Shots[i].ImageAsset.Variants = []models.ImageVariant{}
		}
		if analysis.Shots[i].RenderedAsset.Variants == nil {
			analysis.Shots[i].RenderedAsset.Variants = []models.ImageVariant{}
		}
		analysis.Shots[i].SceneID = resolveEntityReference(analysis.Shots[i].SceneID, sceneAliases)
		analysis.Shots[i].CharacterIDs = resolveEntityReferences(analysis.Shots[i].CharacterIDs, characterAliases)
	}
	if analysis.ArtDirection == nil {
		analysis.ArtDirection = defaultArtDirection(analysis)
	} else {
		if analysis.ArtDirection.CustomStyles == nil {
			analysis.ArtDirection.CustomStyles = []map[string]any{}
		}
		if analysis.ArtDirection.AIRecommendations == nil {
			analysis.ArtDirection.AIRecommendations = []map[string]any{}
		}
	}
}

func entityAliasMap(entities []models.Entity) map[string]string {
	aliases := map[string]string{}
	for _, entity := range entities {
		if entity.ID == "" {
			continue
		}
		for _, value := range []string{entity.ID, entity.Name} {
			key := normalizeReferenceKey(value)
			if key != "" {
				aliases[key] = entity.ID
			}
		}
	}
	return aliases
}

func resolveEntityReference(value string, aliases map[string]string) string {
	key := normalizeReferenceKey(value)
	if key == "" {
		return ""
	}
	if resolved := aliases[key]; resolved != "" {
		return resolved
	}
	return value
}

func resolveEntityReferences(values []string, aliases map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	resolved := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		next := resolveEntityReference(value, aliases)
		key := normalizeReferenceKey(next)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		resolved = append(resolved, next)
	}
	return resolved
}

func normalizeReferenceKey(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), ""))
}

func normalizeEntity(entity *models.Entity) {
	if entity.Status == "" {
		entity.Status = models.GenerationStatusPending
	}
	if entity.Image.Variants == nil {
		entity.Image.Variants = []models.ImageVariant{}
	}
	if entity.FullBody.ImageVariants == nil {
		entity.FullBody.ImageVariants = []models.ImageVariant{}
	}
	if entity.FullBody.VideoVariants == nil {
		entity.FullBody.VideoVariants = []models.VideoVariant{}
	}
	if entity.ThreeViews.ImageVariants == nil {
		entity.ThreeViews.ImageVariants = []models.ImageVariant{}
	}
	if entity.ThreeViews.VideoVariants == nil {
		entity.ThreeViews.VideoVariants = []models.VideoVariant{}
	}
	if entity.HeadShot.ImageVariants == nil {
		entity.HeadShot.ImageVariants = []models.ImageVariant{}
	}
	if entity.HeadShot.VideoVariants == nil {
		entity.HeadShot.VideoVariants = []models.VideoVariant{}
	}
	if entity.VideoAssets == nil {
		entity.VideoAssets = []models.VideoTask{}
	}
}
