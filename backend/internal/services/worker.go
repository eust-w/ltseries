package services

import (
	"context"
	"encoding/json"
	"log"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"ltseries/backend/internal/models"
	"ltseries/backend/internal/queue"
)

type Analyzer interface {
	Analyze(ctx context.Context, project *models.Project) (*models.ScriptAnalysis, error)
}

type MediaProvider interface {
	GenerateAsset(ctx context.Context, payload map[string]any) (map[string]any, error)
	GenerateVideo(ctx context.Context, payload map[string]any) (map[string]any, error)
	GenerateAudio(ctx context.Context, payload map[string]any) (map[string]any, error)
}

type Worker struct {
	queue    *queue.MongoQueue
	projects *ProjectService
	analyzer Analyzer
	provider MediaProvider
	media    *MediaService
	poll     time.Duration
}

func NewWorker(q *queue.MongoQueue, projects *ProjectService, analyzer Analyzer, provider MediaProvider, media *MediaService, poll time.Duration) *Worker {
	if poll <= 0 {
		poll = time.Second
	}
	return &Worker{
		queue:    q,
		projects: projects,
		analyzer: analyzer,
		provider: provider,
		media:    media,
		poll:     poll,
	}
}

func (w *Worker) Start(ctx context.Context, workerCount int) {
	if workerCount <= 0 {
		return
	}
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		workerID := "ltseries-worker-" + time.Now().Format("20060102150405") + "-" + string(rune('a'+i))
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			w.loop(ctx, id)
		}(workerID)
	}
	go func() {
		<-ctx.Done()
		wg.Wait()
	}()
}

func (w *Worker) loop(ctx context.Context, workerID string) {
	ticker := time.NewTicker(w.poll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			processed, err := w.ProcessOne(ctx, workerID)
			if err != nil {
				log.Printf("worker %s process error: %v", workerID, err)
			}
			if processed {
				continue
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) ProcessOne(ctx context.Context, workerID string) (bool, error) {
	task, err := w.queue.ClaimNext(ctx, workerID)
	if err != nil || task == nil {
		return false, err
	}

	switch task.Type {
	case models.TaskTypeAnalysis:
		w.processAnalysis(ctx, task)
	case models.TaskTypeStoryboardAnalysis:
		w.processStoryboardAnalysis(ctx, task)
	case models.TaskTypeArtDirection:
		w.processArtDirection(ctx, task)
	case models.TaskTypeAssetGeneration:
		w.processAsset(ctx, task)
	case models.TaskTypeStoryboardRender:
		w.processStoryboardRender(ctx, task)
	case models.TaskTypeVideoGeneration:
		w.processVideo(ctx, task)
	case models.TaskTypeAssetVideo:
		w.processVideo(ctx, task)
	case models.TaskTypeAudioGeneration, models.TaskTypeSFXGeneration, models.TaskTypeBGMGeneration:
		w.processAudio(ctx, task)
	case models.TaskTypeMerge:
		w.processMerge(ctx, task)
	case models.TaskTypeStoryboardPipeline:
		w.processStoryboardPipeline(ctx, task)
	default:
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, "unsupported task type: "+string(task.Type), nil)
	}
	return true, nil
}

func (w *Worker) processStoryboardAnalysis(ctx context.Context, task *models.Task) {
	project, err := w.projects.Get(ctx, task.ProjectID)
	if err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	if text, ok := task.Payload["text"].(string); ok && text != "" {
		project.Script = text
	}
	analysis, err := w.analyzer.Analyze(ctx, project)
	if err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	if err := w.projects.SaveAnalysis(ctx, project.ID, analysis); err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, 100, "", map[string]any{"frames": asMap(analysis)["shots"]})
}

func (w *Worker) processArtDirection(ctx context.Context, task *models.Task) {
	project, err := w.projects.Get(ctx, task.ProjectID)
	if err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	analysis, err := w.analyzer.Analyze(ctx, project)
	if err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	artDirection := analysis.ArtDirection
	if artDirection == nil {
		artDirection = defaultArtDirection(analysis)
	}
	if _, err := w.projects.SaveArtDirection(ctx, project.ID, *artDirection); err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	result := map[string]any{"art_direction": asMap(artDirection)}
	_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, 100, "", result)
}

func (w *Worker) processAnalysis(ctx context.Context, task *models.Task) {
	project, err := w.projects.Get(ctx, task.ProjectID)
	if err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	analysis, err := w.analyzer.Analyze(ctx, project)
	if err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	if err := w.projects.SaveAnalysis(ctx, project.ID, analysis); err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, 100, "", map[string]any{
		"analysis": asMap(analysis),
	})
}

func (w *Worker) processAsset(ctx context.Context, task *models.Task) {
	result, err := w.provider.GenerateAsset(ctx, task.Payload)
	if err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	if err := w.projects.ApplyAssetGenerationResult(ctx, task, result); err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, 100, "", result)
}

func (w *Worker) processStoryboardRender(ctx context.Context, task *models.Task) {
	result, err := w.provider.GenerateAsset(ctx, task.Payload)
	if err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	if err := w.projects.ApplyStoryboardRenderResult(ctx, task, result); err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, 100, "", result)
}

func (w *Worker) processVideo(ctx context.Context, task *models.Task) {
	result, err := w.provider.GenerateVideo(ctx, task.Payload)
	if err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	if err := w.projects.ApplyVideoResult(ctx, task, result); err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, 100, "", result)
}

func (w *Worker) processAudio(ctx context.Context, task *models.Task) {
	result, err := w.provider.GenerateAudio(ctx, task.Payload)
	if err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	if err := w.projects.ApplyAudioResult(ctx, task, result); err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, 100, "", result)
}

func (w *Worker) processMerge(ctx context.Context, task *models.Task) {
	project, err := w.projects.Get(ctx, task.ProjectID)
	if err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	url, err := w.media.MergeProject(ctx, project)
	if err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	if err := w.projects.SaveMergedVideo(ctx, project.ID, url); err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, 100, "", map[string]any{"merged_video_url": url})
}

func (w *Worker) processStoryboardPipeline(ctx context.Context, task *models.Task) {
	project, err := w.projects.Get(ctx, task.ProjectID)
	if err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, err.Error(), nil)
		return
	}
	frames := append([]models.StoryboardShot{}, project.Frames...)
	sort.Slice(frames, func(i, j int) bool { return frames[i].Index < frames[j].Index })
	if len(frames) == 0 {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 0, "no storyboard frames to process", nil)
		return
	}

	model := stringFromAny(task.Payload["model"])
	if model == "" {
		model = project.ModelSettings.I2VModel
	}
	if model == "" {
		model = envString("DEFAULT_VIDEO_MODEL", "wan2.7-i2v")
	}
	duration := intFromAny(task.Payload["duration"], 5)
	parameters := mapFromAny(task.Payload["parameters"])
	if parameters == nil {
		parameters = map[string]any{}
	}
	if _, ok := parameters["aspect_ratio"]; !ok && project.ModelSettings.StoryboardAspectRatio != "" {
		parameters["aspect_ratio"] = project.ModelSettings.StoryboardAspectRatio
	}
	if _, ok := parameters["ratio"]; !ok && project.ModelSettings.StoryboardAspectRatio != "" {
		parameters["ratio"] = project.ModelSettings.StoryboardAspectRatio
	}
	if _, ok := parameters["audio"]; !ok {
		parameters["audio"] = false
	}
	if _, ok := parameters["watermark"]; !ok {
		parameters["watermark"] = false
	}
	regenerateImages := boolParam(task.Payload, "regenerate_images", false)
	regenerateVideos := boolParam(task.Payload, "regenerate_videos", false)

	processed := 0
	previousFrameImageURL := ""
	for index, frame := range frames {
		imageURL := selectedFrameImageURL(frame)
		references := frameReferenceURLs(project, frame, previousFrameImageURL)
		if frame.VideoURL != "" && !regenerateVideos {
			if imageURL != "" {
				previousFrameImageURL = firstNonEmpty(frame.LastFrameURL, imageURL)
			}
			processed++
			_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusProcessing, pipelineProgress(index+1, len(frames), 90), "", nil)
			continue
		}
		if regenerateImages {
			imageURL = ""
		}
		if imageURL == "" {
			imagePayload := map[string]any{
				"frame_id":             frame.ID,
				"prompt":               storyboardImagePrompt(project, frame, len(references) > 0, previousFrameImageURL != ""),
				"model":                firstNonEmpty(project.ModelSettings.ImageModel, project.ModelSettings.T2IModel, envString("DEFAULT_IMAGE_MODEL", "wan2.7-image-pro")),
				"references":           references,
				"reference_image_urls": references,
				"parameters": map[string]any{
					"aspect_ratio": project.ModelSettings.StoryboardAspectRatio,
					"watermark":    false,
					"n":            1,
				},
			}
			imageResult, err := w.provider.GenerateAsset(ctx, imagePayload)
			if err != nil {
				if shouldRetryWithoutReferences(err) && len(references) > 0 {
					log.Printf("retry storyboard image without expired/inaccessible references project=%s frame=%s: %v", task.ProjectID, frame.ID, err)
					imagePayload["references"] = []string{}
					imagePayload["reference_image_urls"] = []string{}
					imageResult, err = w.provider.GenerateAsset(ctx, imagePayload)
				}
				if err != nil {
					_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, pipelineProgress(index, len(frames), 90), err.Error(), nil)
					return
				}
			}
			imageTask := &models.Task{ProjectID: task.ProjectID, Payload: imagePayload}
			if err := w.projects.ApplyStoryboardRenderResult(ctx, imageTask, imageResult); err != nil {
				_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, pipelineProgress(index, len(frames), 90), err.Error(), nil)
				return
			}
			imageURL = firstString(imageResult, "image_url", "url", "rendered_url")
		}
		if imageURL != "" {
			previousFrameImageURL = imageURL
		}

		videoPayload := map[string]any{
			"frame_id":             frame.ID,
			"prompt":               storyboardVideoPrompt(project, frame),
			"image_url":            imageURL,
			"generation_mode":      firstNonEmpty(project.Workflow, "i2v"),
			"model":                model,
			"duration":             duration,
			"generate_audio":       false,
			"reference_image_urls": references,
			"parameters":           cloneMap(parameters),
		}
		if _, err := w.projects.AddFrameVideoRecord(ctx, task.ProjectID, frame.ID, videoPayload); err != nil {
			_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, pipelineProgress(index, len(frames), 90), err.Error(), nil)
			return
		}
		videoResult, err := w.provider.GenerateVideo(ctx, videoPayload)
		if err != nil {
			_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, pipelineProgress(index, len(frames), 90), err.Error(), nil)
			return
		}
		videoTask := &models.Task{Type: models.TaskTypeVideoGeneration, ProjectID: task.ProjectID, Payload: videoPayload}
		if err := w.projects.ApplyVideoResult(ctx, videoTask, videoResult); err != nil {
			_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, pipelineProgress(index, len(frames), 90), err.Error(), nil)
			return
		}
		if lastFrameURL := w.extractAndSaveLastFrame(ctx, task.ProjectID, frame.ID, firstString(videoResult, "video_url", "url")); lastFrameURL != "" {
			previousFrameImageURL = lastFrameURL
		}
		processed++
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusProcessing, pipelineProgress(index+1, len(frames), 90), "", nil)
	}

	project, err = w.projects.Get(ctx, task.ProjectID)
	if err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 90, err.Error(), nil)
		return
	}
	if w.media == nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, 100, "", map[string]any{"processed_frames": processed})
		return
	}
	url, err := w.media.MergeProject(ctx, project)
	if err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 90, err.Error(), nil)
		return
	}
	if err := w.projects.SaveMergedVideo(ctx, project.ID, url); err != nil {
		_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusFailed, 95, err.Error(), nil)
		return
	}
	_ = w.queue.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, 100, "", map[string]any{"processed_frames": processed, "merged_video_url": url})
}

func (w *Worker) extractAndSaveLastFrame(ctx context.Context, projectID string, frameID string, videoURL string) string {
	if w.media == nil || strings.TrimSpace(videoURL) == "" {
		return ""
	}
	lastFrameURL, err := w.media.ExtractLastFrame(ctx, videoURL, projectID, frameID)
	if err != nil {
		log.Printf("extract last frame failed project=%s frame=%s: %v", projectID, frameID, err)
		return ""
	}
	if _, err := w.projects.UpdateFrame(ctx, projectID, frameID, map[string]any{"last_frame_url": lastFrameURL}); err != nil {
		log.Printf("save last frame failed project=%s frame=%s: %v", projectID, frameID, err)
		return ""
	}
	return lastFrameURL
}

func selectedFrameImageURL(frame models.StoryboardShot) string {
	if frame.RenderedURL != "" {
		return frame.RenderedURL
	}
	if frame.ImageURL != "" {
		return frame.ImageURL
	}
	if url := selectedImageURL(frame.RenderedAsset); url != "" {
		return url
	}
	return selectedImageURL(frame.ImageAsset)
}

func storyboardImagePrompt(project *models.Project, frame models.StoryboardShot, hasReferences bool, hasPreviousFrame bool) string {
	parts := []string{firstNonEmpty(frame.Prompt, frame.Action)}
	if project != nil && project.PromptConfig.StoryboardPolish != "" {
		parts = append(parts, project.PromptConfig.StoryboardPolish)
	}
	if project != nil && project.ArtDirection != nil && project.ArtDirection.StyleConfig != nil {
		if value := strings.TrimSpace(stringFromAny(project.ArtDirection.StyleConfig["positive_prompt"])); value != "" {
			parts = append(parts, value)
		}
	}
	if hasReferences {
		parts = append(parts, "连续性约束：严格参考输入图片中的角色五官、发型、服装、体型、场景空间和道具外观；不要重新设计角色、服装、场景或关键道具。")
	}
	if hasPreviousFrame {
		parts = append(parts, "承接上一镜头画面：保持人物状态、光线、天气、色调和空间关系连贯，同时按本镜头动作与机位重新构图。")
	}
	return joinPromptParts(parts)
}

func storyboardVideoPrompt(project *models.Project, frame models.StoryboardShot) string {
	parts := []string{
		firstNonEmpty(frame.Prompt, frame.Action),
		"保持首帧中角色身份、服装、场景、道具和色调稳定，镜头运动只服务于本镜头动作。",
	}
	if project != nil && project.PromptConfig.VideoPolish != "" {
		parts = append(parts, project.PromptConfig.VideoPolish)
	}
	return joinPromptParts(parts)
}

func joinPromptParts(parts []string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.TrimRight(part, "。.")
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	return strings.Join(cleaned, "。")
}

func frameReferenceURLs(project *models.Project, frame models.StoryboardShot, previousFrameImageURL string) []string {
	if project == nil {
		return nil
	}
	urls := []string{}
	urls = appendReferenceURL(urls, previousFrameImageURL)

	frameText := strings.Join([]string{frame.Action, frame.Camera, frame.Dialogue, frame.Prompt}, " ")
	characters := entitiesForReferences(project.Characters, frame.CharacterIDs, frameText, len(project.Characters) == 1)
	for _, character := range characters {
		urls = appendReferenceURL(urls, selectedUnitImageURL(character.FullBody))
		urls = appendReferenceURL(urls, selectedUnitImageURL(character.ThreeViews))
		urls = appendReferenceURL(urls, selectedUnitImageURL(character.HeadShot))
		urls = appendReferenceURL(urls, selectedImageURL(character.Image))
	}

	if scene, ok := entityForReference(project.Scenes, frame.SceneID, frameText, len(project.Scenes) == 1); ok {
		urls = appendReferenceURL(urls, selectedImageURL(scene.Image))
	}

	includeAllProps := len(project.Props) <= 3
	for _, prop := range entitiesForReferences(project.Props, nil, frameText, includeAllProps) {
		urls = appendReferenceURL(urls, selectedImageURL(prop.Image))
	}
	return limitReferences(urls, 8)
}

func entitiesForReferences(entities []models.Entity, references []string, text string, includeFallback bool) []models.Entity {
	result := []models.Entity{}
	if len(references) > 0 {
		for _, reference := range references {
			if entity, ok := entityForReference(entities, reference, "", false); ok {
				result = append(result, entity)
			}
		}
		return result
	}
	for _, entity := range entities {
		if includeFallback || entityMentioned(text, entity) {
			result = append(result, entity)
		}
	}
	return result
}

func entityForReference(entities []models.Entity, reference string, text string, includeFallback bool) (models.Entity, bool) {
	refKey := normalizeReferenceKey(reference)
	for _, entity := range entities {
		if refKey != "" && (normalizeReferenceKey(entity.ID) == refKey || normalizeReferenceKey(entity.Name) == refKey) {
			return entity, true
		}
	}
	for _, entity := range entities {
		if text != "" && entityMentioned(text, entity) {
			return entity, true
		}
	}
	if includeFallback && len(entities) > 0 {
		return entities[0], true
	}
	return models.Entity{}, false
}

func entityMentioned(text string, entity models.Entity) bool {
	normalizedText := normalizeReferenceKey(text)
	if normalizedText == "" {
		return false
	}
	for _, value := range []string{entity.Name, entity.ID} {
		key := normalizeReferenceKey(value)
		if key != "" && strings.Contains(normalizedText, key) {
			return true
		}
	}
	return false
}

func appendReferenceURL(urls []string, url string) []string {
	url = strings.TrimSpace(url)
	if !isProviderFetchableReferenceURL(url) {
		return urls
	}
	for _, existing := range urls {
		if existing == url {
			return urls
		}
	}
	return append(urls, url)
}

func isProviderFetchableReferenceURL(url string) bool {
	lower := strings.ToLower(strings.TrimSpace(url))
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return false
	}
	if referenceURLExpired(lower, 30*time.Minute) {
		return false
	}
	return !strings.Contains(lower, "localhost") &&
		!strings.Contains(lower, "127.0.0.1") &&
		!strings.Contains(lower, "[::1]")
}

func referenceURLExpired(rawURL string, margin time.Duration) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	expires := parsed.Query().Get("Expires")
	if expires == "" {
		expires = parsed.Query().Get("expires")
	}
	if expires == "" {
		return false
	}
	unix, err := strconv.ParseInt(expires, 10, 64)
	if err != nil {
		return false
	}
	return unix <= time.Now().Add(margin).Unix()
}

func shouldRetryWithoutReferences(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "datainspection") ||
		strings.Contains(message, "authorization to access the media resource") ||
		strings.Contains(message, "access the media resource")
}

func limitReferences(urls []string, limit int) []string {
	if limit <= 0 || len(urls) <= limit {
		return urls
	}
	return urls[:limit]
}

func pipelineProgress(done int, total int, max int) int {
	if total <= 0 {
		return 0
	}
	progress := done * max / total
	if progress < 1 {
		return 1
	}
	if progress > max {
		return max
	}
	return progress
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func asMap(value any) map[string]any {
	encoded, err := json.Marshal(value)
	if err != nil {
		return map[string]any{"value": value}
	}
	var result map[string]any
	if err := json.Unmarshal(encoded, &result); err != nil {
		return map[string]any{"value": value}
	}
	return result
}
