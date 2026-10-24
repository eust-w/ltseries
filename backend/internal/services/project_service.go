package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"ltseries/backend/internal/models"
	"ltseries/backend/internal/queue"
)

var ErrNotFound = errors.New("not found")

type ProjectService struct {
	projects *mongo.Collection
	queue    *queue.MongoQueue
}

type SeriesService struct {
	series   *mongo.Collection
	projects *ProjectService
	queue    *queue.MongoQueue
}

func NewProjectService(ctx context.Context, projects *mongo.Collection, q *queue.MongoQueue) (*ProjectService, error) {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "title", Value: 1}}},
	}
	if _, err := projects.Indexes().CreateMany(ctx, indexes); err != nil {
		return nil, err
	}
	return &ProjectService{projects: projects, queue: q}, nil
}

func (s *ProjectService) Create(ctx context.Context, title string, script string, workflow string) (*models.Project, error) {
	title = strings.TrimSpace(title)
	script = strings.TrimSpace(script)
	workflow = strings.TrimSpace(workflow)
	if title == "" {
		return nil, errors.New("title is required")
	}
	if script == "" {
		return nil, errors.New("script is required")
	}
	if workflow == "" {
		workflow = "r2v"
	}

	now := time.Now().UTC()
	project := &models.Project{
		ID:            uuid.NewString(),
		Title:         title,
		Script:        script,
		Workflow:      workflow,
		Characters:    []models.Entity{},
		Scenes:        []models.Entity{},
		Props:         []models.Entity{},
		Frames:        []models.StoryboardShot{},
		VideoTasks:    []models.VideoTask{},
		StylePreset:   "realistic",
		ModelSettings: defaultModelSettings(),
		PromptConfig:  models.PromptConfig{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if _, err := s.projects.InsertOne(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *ProjectService) Get(ctx context.Context, id string) (*models.Project, error) {
	var project models.Project
	err := s.projects.FindOne(ctx, bson.M{"_id": id}).Decode(&project)
	if err == mongo.ErrNoDocuments {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (s *ProjectService) ProductionAudit(ctx context.Context, projectID string) (*models.ProductionAuditReport, error) {
	project, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return buildProductionAudit(project), nil
}

func (s *ProjectService) List(ctx context.Context, limit int64) ([]models.Project, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	cursor, err := s.projects.Find(
		ctx,
		bson.M{},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	projects := make([]models.Project, 0)
	if err := cursor.All(ctx, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (s *ProjectService) EnqueueAnalysis(ctx context.Context, projectID string) (*models.Task, error) {
	project, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	task := &models.Task{
		ID:        uuid.NewString(),
		Type:      models.TaskTypeAnalysis,
		Status:    models.TaskStatusPending,
		Progress:  0,
		ProjectID: project.ID,
		Payload: map[string]any{
			"title":  project.Title,
			"script": project.Script,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.queue.Enqueue(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *ProjectService) EnqueueAsset(ctx context.Context, projectID string, payload map[string]any) (*models.Task, error) {
	if _, err := s.Get(ctx, projectID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	task := &models.Task{
		ID:        uuid.NewString(),
		Type:      models.TaskTypeAssetGeneration,
		Status:    models.TaskStatusPending,
		Progress:  0,
		ProjectID: projectID,
		Payload:   payload,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.queue.Enqueue(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *ProjectService) EnqueueVideo(ctx context.Context, projectID string, payload map[string]any) (*models.Task, error) {
	if _, err := s.Get(ctx, projectID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	task := &models.Task{
		ID:        uuid.NewString(),
		Type:      models.TaskTypeVideoGeneration,
		Status:    models.TaskStatusPending,
		Progress:  0,
		ProjectID: projectID,
		Payload:   payload,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.queue.Enqueue(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *ProjectService) SaveAnalysis(ctx context.Context, projectID string, analysis *models.ScriptAnalysis) error {
	if analysis == nil {
		return errors.New("analysis is required")
	}
	if analysis.ArtDirection == nil {
		analysis.ArtDirection = defaultArtDirection(analysis)
	}
	set := bson.M{
		"analysis":   analysis,
		"characters": analysis.Characters,
		"scenes":     analysis.Scenes,
		"props":      analysis.Props,
		"frames":     analysis.Shots,
		"updated_at": time.Now().UTC(),
	}
	if analysis.ArtDirection != nil {
		set["art_direction"] = analysis.ArtDirection
	}
	result, err := s.projects.UpdateOne(
		ctx,
		bson.M{"_id": projectID},
		bson.M{"$set": set},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *ProjectService) Delete(ctx context.Context, id string) error {
	result, err := s.projects.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *ProjectService) Update(ctx context.Context, id string, updates map[string]any) (*models.Project, error) {
	set := bson.M{"updated_at": time.Now().UTC()}
	for key, value := range updates {
		raw, ok := value.(string)
		if !ok {
			continue
		}
		next := strings.TrimSpace(raw)
		switch key {
		case "title":
			if next == "" {
				return nil, errors.New("title is required")
			}
			set["title"] = next
		case "script":
			if next == "" {
				return nil, errors.New("script is required")
			}
			set["script"] = next
		case "workflow":
			if next == "" {
				return nil, errors.New("workflow is required")
			}
			set["workflow"] = next
		}
	}
	result, err := s.projects.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": set})
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, id)
}

func (s *ProjectService) AddEntity(ctx context.Context, projectID string, assetType string, name string, description string) (*models.Project, error) {
	field, err := assetField(assetType)
	if err != nil {
		return nil, err
	}
	entity := models.Entity{
		ID:          assetPrefix(assetType) + "_" + uuid.NewString(),
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
		Status:      models.GenerationStatusPending,
		Image:       models.ImageAsset{Variants: []models.ImageVariant{}},
		FullBody:    models.AssetUnit{ImageVariants: []models.ImageVariant{}, VideoVariants: []models.VideoVariant{}},
		ThreeViews:  models.AssetUnit{ImageVariants: []models.ImageVariant{}, VideoVariants: []models.VideoVariant{}},
		HeadShot:    models.AssetUnit{ImageVariants: []models.ImageVariant{}, VideoVariants: []models.VideoVariant{}},
		VideoAssets: []models.VideoTask{},
		VoiceSpeed:  1,
		VoicePitch:  1,
		VoiceVolume: 50,
	}
	if entity.Name == "" || entity.Description == "" {
		return nil, errors.New("name and description are required")
	}
	if _, err := s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{
		"$push": bson.M{field: entity},
		"$set":  bson.M{"updated_at": time.Now().UTC()},
	}); err != nil {
		return nil, err
	}
	return s.Get(ctx, projectID)
}

func (s *ProjectService) DeleteEntity(ctx context.Context, projectID string, assetType string, assetID string) (*models.Project, error) {
	field, err := assetField(assetType)
	if err != nil {
		return nil, err
	}
	if _, err := s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{
		"$pull": bson.M{field: bson.M{"id": assetID}},
		"$set":  bson.M{"updated_at": time.Now().UTC()},
	}); err != nil {
		return nil, err
	}
	return s.Get(ctx, projectID)
}

func (s *ProjectService) UpdateEntity(ctx context.Context, projectID string, assetType string, assetID string, updates map[string]any) (*models.Project, error) {
	field, err := assetField(assetType)
	if err != nil {
		return nil, err
	}
	set := bson.M{"updated_at": time.Now().UTC()}
	for key, value := range updates {
		if key == "id" {
			continue
		}
		set[field+".$."+key] = value
	}
	result, err := s.projects.UpdateOne(ctx, bson.M{"_id": projectID, field + ".id": assetID}, bson.M{"$set": set})
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, projectID)
}

func (s *ProjectService) ToggleEntityLock(ctx context.Context, projectID string, assetType string, assetID string) (*models.Project, error) {
	project, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	entity, _, err := findEntity(project, assetType, assetID)
	if err != nil {
		return nil, err
	}
	return s.UpdateEntity(ctx, projectID, assetType, assetID, map[string]any{"locked": !entity.Locked})
}

func (s *ProjectService) AddUploadedVariant(ctx context.Context, projectID string, assetType string, assetID string, uploadType string, url string, prompt string) (*models.Project, error) {
	return s.appendImageVariant(ctx, projectID, assetType, assetID, uploadType, models.ImageVariant{
		ID:             "variant_" + uuid.NewString(),
		URL:            url,
		Prompt:         prompt,
		UploadedSource: true,
		UploadType:     uploadType,
		CreatedAt:      time.Now().UTC(),
	})
}

func (s *ProjectService) EnqueueAssetGeneration(ctx context.Context, projectID string, assetID string, assetType string, uploadType string, prompt string, model string, inputs map[string]any, parameters map[string]any) (*models.Task, error) {
	if _, err := s.Get(ctx, projectID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	task := &models.Task{
		ID:        uuid.NewString(),
		Type:      models.TaskTypeAssetGeneration,
		Status:    models.TaskStatusPending,
		ProjectID: projectID,
		Payload: map[string]any{
			"asset_id":    assetID,
			"asset_type":  assetType,
			"upload_type": uploadType,
			"prompt":      prompt,
			"model":       model,
			"inputs":      inputs,
			"parameters":  parameters,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.queue.Enqueue(ctx, task); err != nil {
		return nil, err
	}
	_, _ = s.UpdateEntity(ctx, projectID, assetType, assetID, map[string]any{"status": models.GenerationStatusProcessing})
	return task, nil
}

func (s *ProjectService) ApplyAssetGenerationResult(ctx context.Context, task *models.Task, result map[string]any) error {
	assetID, _ := task.Payload["asset_id"].(string)
	assetType, _ := task.Payload["asset_type"].(string)
	uploadType, _ := task.Payload["upload_type"].(string)
	prompt, _ := task.Payload["prompt"].(string)
	url := firstString(result, "image_url", "url", "asset_url")
	if url == "" {
		return errors.New("asset provider response missing image_url/url")
	}
	if _, err := s.appendImageVariant(ctx, task.ProjectID, assetType, assetID, uploadType, models.ImageVariant{
		ID:        "variant_" + uuid.NewString(),
		URL:       url,
		Prompt:    prompt,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return err
	}
	_, _ = s.UpdateEntity(ctx, task.ProjectID, assetType, assetID, map[string]any{"status": models.GenerationStatusCompleted})
	return nil
}

func (s *ProjectService) appendImageVariant(ctx context.Context, projectID string, assetType string, assetID string, uploadType string, variant models.ImageVariant) (*models.Project, error) {
	project, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	state, err := imageVariantState(project, assetType, assetID, uploadType)
	if err != nil {
		return nil, err
	}
	if _, err := s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{
		"$push": bson.M{state.variantsPath: variant},
		"$set":  bson.M{state.selectedPath: variant.ID, "updated_at": time.Now().UTC()},
	}); err != nil {
		return nil, err
	}
	return s.Get(ctx, projectID)
}

func (s *ProjectService) SelectVariant(ctx context.Context, projectID string, assetType string, assetID string, uploadType string, variantID string) (*models.Project, error) {
	project, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	state, err := imageVariantState(project, assetType, assetID, uploadType)
	if err != nil {
		return nil, err
	}
	if !containsImageVariant(state.variants, variantID) {
		return nil, ErrNotFound
	}
	_, err = s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{"$set": bson.M{state.selectedPath: variantID, "updated_at": time.Now().UTC()}})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, projectID)
}

func (s *ProjectService) FavoriteVariant(ctx context.Context, projectID string, assetType string, assetID string, uploadType string, variantID string, favorited bool) (*models.Project, error) {
	project, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	state, err := imageVariantState(project, assetType, assetID, uploadType)
	if err != nil {
		return nil, err
	}
	for index, variant := range state.variants {
		if variant.ID == variantID {
			path := fmt.Sprintf("%s.%d.favorited", state.variantsPath, index)
			if _, err := s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{"$set": bson.M{path: favorited, "updated_at": time.Now().UTC()}}); err != nil {
				return nil, err
			}
			return s.Get(ctx, projectID)
		}
	}
	return nil, ErrNotFound
}

func (s *ProjectService) DeleteVariant(ctx context.Context, projectID string, assetType string, assetID string, uploadType string, variantID string) (*models.Project, error) {
	project, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	state, err := imageVariantState(project, assetType, assetID, uploadType)
	if err != nil {
		return nil, err
	}
	remaining := make([]models.ImageVariant, 0, len(state.variants))
	found := false
	for _, variant := range state.variants {
		if variant.ID == variantID {
			found = true
			continue
		}
		remaining = append(remaining, variant)
	}
	if !found {
		return nil, ErrNotFound
	}
	set := bson.M{state.variantsPath: remaining, "updated_at": time.Now().UTC()}
	update := bson.M{"$set": set}
	if state.selectedID == variantID {
		if len(remaining) > 0 {
			set[state.selectedPath] = remaining[0].ID
		} else {
			update["$unset"] = bson.M{state.selectedPath: ""}
		}
	}
	if _, err := s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, update); err != nil {
		return nil, err
	}
	return s.Get(ctx, projectID)
}

func (s *ProjectService) EnqueueAllAssetGenerations(ctx context.Context, projectID string, model string, parameters map[string]any, force bool) ([]*models.Task, error) {
	project, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	tasks := []*models.Task{}
	enqueue := func(assetType string, asset models.Entity, uploadType string, prompt string, inputs map[string]any, aspectRatio string) error {
		if asset.Locked {
			return nil
		}
		params := cloneMap(parameters)
		if params == nil {
			params = map[string]any{}
		}
		if _, exists := params["aspect_ratio"]; !exists && aspectRatio != "" {
			params["aspect_ratio"] = aspectRatio
		}
		task, err := s.EnqueueAssetGeneration(ctx, projectID, asset.ID, assetType, uploadType, prompt, model, inputs, params)
		if err != nil {
			return err
		}
		tasks = append(tasks, task)
		return nil
	}
	for _, asset := range project.Characters {
		if asset.Locked {
			continue
		}
		fullBodyURL := selectedUnitImageURL(asset.FullBody)
		if force || fullBodyURL == "" {
			if err := enqueue("character", asset, "full_body", assetGenerationPrompt(asset, "full_body"), nil, project.ModelSettings.CharacterAspectRatio); err != nil {
				return nil, err
			}
			if !force {
				continue
			}
		}
		if isProviderFetchableReferenceURL(fullBodyURL) {
			inputs := map[string]any{"reference_image_urls": []string{fullBodyURL}}
			if force || selectedUnitImageURL(asset.ThreeViews) == "" {
				if err := enqueue("character", asset, "three_views", assetGenerationPrompt(asset, "three_views"), inputs, project.ModelSettings.CharacterAspectRatio); err != nil {
					return nil, err
				}
			}
			if force || selectedUnitImageURL(asset.HeadShot) == "" {
				if err := enqueue("character", asset, "head_shot", assetGenerationPrompt(asset, "head_shot"), inputs, project.ModelSettings.CharacterAspectRatio); err != nil {
					return nil, err
				}
			}
		} else {
			if force || selectedUnitImageURL(asset.ThreeViews) == "" {
				if err := enqueue("character", asset, "three_views", assetGenerationPrompt(asset, "three_views"), nil, project.ModelSettings.CharacterAspectRatio); err != nil {
					return nil, err
				}
			}
			if force || selectedUnitImageURL(asset.HeadShot) == "" {
				if err := enqueue("character", asset, "head_shot", assetGenerationPrompt(asset, "head_shot"), nil, project.ModelSettings.CharacterAspectRatio); err != nil {
					return nil, err
				}
			}
		}
	}
	for _, asset := range project.Scenes {
		if !force && selectedImageURL(asset.Image) != "" {
			continue
		}
		if err := enqueue("scene", asset, "image", assetGenerationPrompt(asset, "image"), nil, project.ModelSettings.SceneAspectRatio); err != nil {
			return nil, err
		}
	}
	for _, asset := range project.Props {
		if !force && selectedImageURL(asset.Image) != "" {
			continue
		}
		if err := enqueue("prop", asset, "image", assetGenerationPrompt(asset, "image"), nil, project.ModelSettings.PropAspectRatio); err != nil {
			return nil, err
		}
	}
	return tasks, nil
}

func (s *ProjectService) EnqueueAssetVideo(ctx context.Context, projectID string, assetType string, assetID string, uploadType string, payload map[string]any) (*models.Task, error) {
	project, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if payload == nil {
		payload = map[string]any{}
	}
	if uploadType == "" {
		uploadType = stringFromAny(payload["upload_type"])
	}
	state, err := imageVariantState(project, assetType, assetID, uploadType)
	if err != nil {
		return nil, err
	}
	imageURL := selectedImageURL(models.ImageAsset{SelectedID: state.selectedID, Variants: state.variants})
	if imageURL == "" {
		return nil, errors.New("selected asset image is required before generating asset video")
	}
	payload["asset_id"] = assetID
	payload["asset_type"] = assetType
	payload["upload_type"] = state.uploadType
	payload["image_url"] = imageURL
	inputs, _ := payload["inputs"].(map[string]any)
	if inputs == nil {
		inputs = map[string]any{}
	}
	inputs["image_url"] = imageURL
	payload["inputs"] = inputs
	if _, ok := payload["duration"]; !ok {
		payload["duration"] = 5
	}
	return s.enqueueGeneric(ctx, projectID, models.TaskTypeAssetVideo, payload)
}

func (s *ProjectService) AddFrame(ctx context.Context, projectID string, action string, camera string, sceneID string) (*models.Project, error) {
	frame := models.StoryboardShot{
		ID:            "frame_" + uuid.NewString(),
		Index:         int(time.Now().UnixNano()),
		SceneID:       sceneID,
		Action:        strings.TrimSpace(action),
		Camera:        camera,
		Status:        models.GenerationStatusPending,
		ImageAsset:    models.ImageAsset{Variants: []models.ImageVariant{}},
		RenderedAsset: models.ImageAsset{Variants: []models.ImageVariant{}},
	}
	if frame.Action == "" {
		return nil, errors.New("action is required")
	}
	if _, err := s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{
		"$push": bson.M{"frames": frame},
		"$set":  bson.M{"updated_at": time.Now().UTC()},
	}); err != nil {
		return nil, err
	}
	return s.reindexFrames(ctx, projectID)
}

func (s *ProjectService) UpdateFrame(ctx context.Context, projectID string, frameID string, updates map[string]any) (*models.Project, error) {
	set := bson.M{"updated_at": time.Now().UTC()}
	for key, value := range updates {
		if key == "id" {
			continue
		}
		set["frames.$."+key] = value
	}
	result, err := s.projects.UpdateOne(ctx, bson.M{"_id": projectID, "frames.id": frameID}, bson.M{"$set": set})
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, projectID)
}

func (s *ProjectService) ToggleFrameLock(ctx context.Context, projectID string, frameID string) (*models.Project, error) {
	project, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, frame := range project.Frames {
		if frame.ID == frameID {
			return s.UpdateFrame(ctx, projectID, frameID, map[string]any{"locked": !frame.Locked})
		}
	}
	return nil, ErrNotFound
}

func (s *ProjectService) AddFrameImageVariant(ctx context.Context, projectID string, frameID string, url string, prompt string, uploaded bool) (*models.Project, error) {
	project, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	frameIndex := -1
	for index, frame := range project.Frames {
		if frame.ID == frameID {
			frameIndex = index
			break
		}
	}
	if frameIndex < 0 {
		return nil, ErrNotFound
	}
	variant := models.ImageVariant{
		ID:             "variant_" + uuid.NewString(),
		URL:            url,
		Prompt:         prompt,
		UploadedSource: uploaded,
		UploadType:     "storyboard",
		CreatedAt:      time.Now().UTC(),
	}
	if _, err := s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{
		"$push": bson.M{fmt.Sprintf("frames.%d.rendered_asset.variants", frameIndex): variant},
		"$set": bson.M{
			fmt.Sprintf("frames.%d.rendered_asset.selected_id", frameIndex): variant.ID,
			fmt.Sprintf("frames.%d.rendered_url", frameIndex):               url,
			"updated_at": time.Now().UTC(),
		},
	}); err != nil {
		return nil, err
	}
	return s.Get(ctx, projectID)
}

func (s *ProjectService) FrameVideoURL(ctx context.Context, projectID string, frameID string, videoTaskID string) (string, error) {
	project, err := s.Get(ctx, projectID)
	if err != nil {
		return "", err
	}
	videoByID := map[string]models.VideoTask{}
	for _, task := range project.VideoTasks {
		videoByID[task.ID] = task
	}
	if videoTaskID != "" {
		if task, ok := videoByID[videoTaskID]; ok && task.VideoURL != "" {
			return task.VideoURL, nil
		}
		return "", ErrNotFound
	}
	for _, frame := range project.Frames {
		if frame.ID != frameID {
			continue
		}
		if frame.SelectedVideoID != "" {
			if task, ok := videoByID[frame.SelectedVideoID]; ok && task.VideoURL != "" {
				return task.VideoURL, nil
			}
		}
		if frame.VideoURL != "" {
			return frame.VideoURL, nil
		}
		for _, task := range project.VideoTasks {
			if task.FrameID == frameID && task.VideoURL != "" {
				return task.VideoURL, nil
			}
		}
		return "", errors.New("frame has no generated video")
	}
	return "", ErrNotFound
}

func (s *ProjectService) DeleteFrame(ctx context.Context, projectID string, frameID string) (*models.Project, error) {
	if _, err := s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{
		"$pull": bson.M{"frames": bson.M{"id": frameID}},
		"$set":  bson.M{"updated_at": time.Now().UTC()},
	}); err != nil {
		return nil, err
	}
	return s.reindexFrames(ctx, projectID)
}

func (s *ProjectService) CopyFrame(ctx context.Context, projectID string, frameID string) (*models.Project, error) {
	project, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, frame := range project.Frames {
		if frame.ID == frameID {
			frame.ID = "frame_" + uuid.NewString()
			frame.SelectedVideoID = ""
			if _, err := s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{
				"$push": bson.M{"frames": frame},
				"$set":  bson.M{"updated_at": time.Now().UTC()},
			}); err != nil {
				return nil, err
			}
			return s.reindexFrames(ctx, projectID)
		}
	}
	return nil, ErrNotFound
}

func (s *ProjectService) ReorderFrames(ctx context.Context, projectID string, frameIDs []string) (*models.Project, error) {
	project, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	byID := map[string]models.StoryboardShot{}
	for _, frame := range project.Frames {
		byID[frame.ID] = frame
	}
	frames := make([]models.StoryboardShot, 0, len(project.Frames))
	for _, id := range frameIDs {
		if frame, ok := byID[id]; ok {
			frames = append(frames, frame)
			delete(byID, id)
		}
	}
	for _, frame := range project.Frames {
		if _, ok := byID[frame.ID]; ok {
			frames = append(frames, frame)
		}
	}
	for i := range frames {
		frames[i].Index = i + 1
	}
	_, err = s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{"$set": bson.M{"frames": frames, "updated_at": time.Now().UTC()}})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, projectID)
}

func (s *ProjectService) reindexFrames(ctx context.Context, projectID string) (*models.Project, error) {
	project, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for i := range project.Frames {
		project.Frames[i].Index = i + 1
	}
	_, err = s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{"$set": bson.M{"frames": project.Frames, "updated_at": time.Now().UTC()}})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, projectID)
}

func (s *ProjectService) EnqueueStoryboardAnalysis(ctx context.Context, projectID string, text string) (*models.Task, error) {
	if strings.TrimSpace(text) == "" {
		project, err := s.Get(ctx, projectID)
		if err != nil {
			return nil, err
		}
		text = project.Script
	}
	return s.enqueueGeneric(ctx, projectID, models.TaskTypeStoryboardAnalysis, map[string]any{"text": text})
}

func (s *ProjectService) EnqueueArtDirection(ctx context.Context, projectID string) (*models.Task, error) {
	return s.enqueueGeneric(ctx, projectID, models.TaskTypeArtDirection, map[string]any{})
}

func (s *ProjectService) EnqueueStoryboardRender(ctx context.Context, projectID string, frameID string, prompt string, references []string) (*models.Task, error) {
	return s.enqueueGeneric(ctx, projectID, models.TaskTypeStoryboardRender, map[string]any{"frame_id": frameID, "prompt": prompt, "references": references})
}

func (s *ProjectService) EnqueueStoryboardPipeline(ctx context.Context, projectID string, payload map[string]any) (*models.Task, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	return s.enqueueGeneric(ctx, projectID, models.TaskTypeStoryboardPipeline, payload)
}

func (s *ProjectService) ApplyStoryboardRenderResult(ctx context.Context, task *models.Task, result map[string]any) error {
	frameID, _ := task.Payload["frame_id"].(string)
	url := firstString(result, "image_url", "url", "rendered_url")
	if url == "" {
		return errors.New("storyboard provider response missing image_url/url")
	}
	_, err := s.UpdateFrame(ctx, task.ProjectID, frameID, map[string]any{
		"rendered_url": url,
		"status":       models.GenerationStatusCompleted,
	})
	return err
}

func (s *ProjectService) EnqueueVideoTask(ctx context.Context, projectID string, payload map[string]any) (*models.Task, error) {
	frameID := stringFromAny(payload["frame_id"])
	if _, err := s.addFrameVideoRecord(ctx, projectID, frameID, payload); err != nil {
		return nil, err
	}
	return s.enqueueGeneric(ctx, projectID, models.TaskTypeVideoGeneration, payload)
}

func (s *ProjectService) AddFrameVideoRecord(ctx context.Context, projectID string, frameID string, payload map[string]any) (string, error) {
	return s.addFrameVideoRecord(ctx, projectID, frameID, payload)
}

func (s *ProjectService) addFrameVideoRecord(ctx context.Context, projectID string, frameID string, payload map[string]any) (string, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	prompt, _ := payload["prompt"].(string)
	video := models.VideoTask{
		ID:             "video_" + uuid.NewString(),
		ProjectID:      projectID,
		FrameID:        frameID,
		ImageURL:       stringFromAny(payload["image_url"]),
		Prompt:         prompt,
		Status:         models.TaskStatusPending,
		Duration:       intFromAny(payload["duration"], 5),
		Resolution:     stringFromAny(payload["resolution"]),
		Model:          stringFromAny(payload["model"]),
		GenerationMode: stringFromAny(payload["generation_mode"]),
		ReferenceImageURLs: compactStrings(append(
			collectStrings(payload, "reference_image_urls", "ref_image_urls"),
			collectStrings(mapFromAny(payload["inputs"]), "reference_image_urls", "ref_image_urls")...,
		)),
		ReferenceVideoURLs: compactStrings(append(
			collectStrings(payload, "reference_video_urls", "ref_video_urls"),
			collectStrings(mapFromAny(payload["inputs"]), "reference_video_urls", "ref_video_urls")...,
		)),
		Parameters: payload,
		CreatedAt:  time.Now().UTC(),
	}
	payload["video_task_id"] = video.ID
	if _, err := s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{
		"$push": bson.M{"video_tasks": video},
		"$set":  bson.M{"updated_at": time.Now().UTC()},
	}); err != nil {
		return "", err
	}
	return video.ID, nil
}

func (s *ProjectService) ApplyVideoResult(ctx context.Context, task *models.Task, result map[string]any) error {
	if task.Type == models.TaskTypeAssetVideo {
		return s.applyAssetVideoResult(ctx, task, result)
	}
	videoTaskID, _ := task.Payload["video_task_id"].(string)
	frameID, _ := task.Payload["frame_id"].(string)
	url := firstString(result, "video_url", "url")
	if url == "" {
		return errors.New("video provider response missing video_url/url")
	}
	_, err := s.projects.UpdateOne(ctx, bson.M{"_id": task.ProjectID, "video_tasks.id": videoTaskID}, bson.M{
		"$set": bson.M{
			"video_tasks.$.status":    models.TaskStatusCompleted,
			"video_tasks.$.video_url": url,
			"video_tasks.$.error":     "",
			"updated_at":              time.Now().UTC(),
		},
	})
	if err != nil {
		return err
	}
	if frameID != "" {
		_, _ = s.UpdateFrame(ctx, task.ProjectID, frameID, map[string]any{"video_url": url, "selected_video_id": videoTaskID})
	}
	return nil
}

func (s *ProjectService) applyAssetVideoResult(ctx context.Context, task *models.Task, result map[string]any) error {
	assetID, _ := task.Payload["asset_id"].(string)
	assetType, _ := task.Payload["asset_type"].(string)
	uploadType, _ := task.Payload["upload_type"].(string)
	prompt, _ := task.Payload["prompt"].(string)
	url := firstString(result, "video_url", "url")
	if url == "" {
		return errors.New("video provider response missing video_url/url")
	}
	project, err := s.Get(ctx, task.ProjectID)
	if err != nil {
		return err
	}
	location, err := assetVideoLocation(project, assetType, assetID, uploadType)
	if err != nil {
		return err
	}
	variant := models.VideoVariant{
		ID:        "video_" + uuid.NewString(),
		URL:       url,
		Prompt:    prompt,
		CreatedAt: time.Now().UTC(),
	}
	if location.unitVideo {
		_, err = s.projects.UpdateOne(ctx, bson.M{"_id": task.ProjectID}, bson.M{
			"$push": bson.M{location.videosPath: variant},
			"$set":  bson.M{location.selectedPath: variant.ID, "updated_at": time.Now().UTC()},
		})
		return err
	}
	video := models.VideoTask{
		ID:             variant.ID,
		ProjectID:      task.ProjectID,
		AssetID:        assetID,
		ImageURL:       stringFromAny(task.Payload["image_url"]),
		Prompt:         prompt,
		Status:         models.TaskStatusCompleted,
		VideoURL:       url,
		Duration:       intFromAny(task.Payload["duration"], 5),
		Model:          stringFromAny(task.Payload["model"]),
		GenerationMode: stringFromAny(task.Payload["generation_mode"]),
		Parameters:     task.Payload,
		CreatedAt:      variant.CreatedAt,
	}
	_, err = s.projects.UpdateOne(ctx, bson.M{"_id": task.ProjectID}, bson.M{
		"$push": bson.M{location.videosPath: video},
		"$set":  bson.M{"updated_at": time.Now().UTC()},
	})
	return err
}

func (s *ProjectService) DeleteAssetVideo(ctx context.Context, projectID string, assetType string, assetID string, uploadType string, videoID string) (*models.Project, error) {
	project, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	location, err := assetVideoLocation(project, assetType, assetID, uploadType)
	if err != nil {
		return nil, err
	}
	update := bson.M{
		"$pull": bson.M{location.videosPath: bson.M{"id": videoID}},
		"$set":  bson.M{"updated_at": time.Now().UTC()},
	}
	if location.unitVideo && location.selectedID == videoID {
		update["$unset"] = bson.M{location.selectedPath: ""}
	}
	if _, err := s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, update); err != nil {
		return nil, err
	}
	return s.Get(ctx, projectID)
}

func (s *ProjectService) SelectFrameVideo(ctx context.Context, projectID string, frameID string, videoID string) (*models.Project, error) {
	return s.UpdateFrame(ctx, projectID, frameID, map[string]any{"selected_video_id": videoID})
}

func (s *ProjectService) EnqueueAudio(ctx context.Context, projectID string, frameID string, audioType string, payload map[string]any) (*models.Task, error) {
	payload["frame_id"] = frameID
	switch audioType {
	case "sfx":
		return s.enqueueGeneric(ctx, projectID, models.TaskTypeSFXGeneration, payload)
	case "bgm":
		return s.enqueueGeneric(ctx, projectID, models.TaskTypeBGMGeneration, payload)
	default:
		return s.enqueueGeneric(ctx, projectID, models.TaskTypeAudioGeneration, payload)
	}
}

func (s *ProjectService) ApplyAudioResult(ctx context.Context, task *models.Task, result map[string]any) error {
	frameID, _ := task.Payload["frame_id"].(string)
	url := firstString(result, "audio_url", "url", "sfx_url", "bgm_url")
	if url == "" {
		return errors.New("audio provider response missing audio_url/url")
	}
	key := "audio_url"
	if task.Type == models.TaskTypeSFXGeneration {
		key = "sfx_url"
	}
	if task.Type == models.TaskTypeBGMGeneration {
		key = "bgm_url"
	}
	_, err := s.UpdateFrame(ctx, task.ProjectID, frameID, map[string]any{key: url})
	return err
}

func (s *ProjectService) EnqueueMerge(ctx context.Context, projectID string) (*models.Task, error) {
	return s.enqueueGeneric(ctx, projectID, models.TaskTypeMerge, map[string]any{})
}

func (s *ProjectService) SaveMergedVideo(ctx context.Context, projectID string, url string) error {
	result, err := s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{"$set": bson.M{"merged_video_url": url, "updated_at": time.Now().UTC()}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *ProjectService) SaveArtDirection(ctx context.Context, projectID string, art models.ArtDirection) (*models.Project, error) {
	if _, err := s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{"$set": bson.M{"art_direction": art, "updated_at": time.Now().UTC()}}); err != nil {
		return nil, err
	}
	return s.Get(ctx, projectID)
}

func (s *ProjectService) UpdateModelSettings(ctx context.Context, projectID string, settings models.ModelSettings) (*models.Project, error) {
	if _, err := s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{"$set": bson.M{"model_settings": settings, "updated_at": time.Now().UTC()}}); err != nil {
		return nil, err
	}
	return s.Get(ctx, projectID)
}

func (s *ProjectService) UpdatePromptConfig(ctx context.Context, projectID string, promptConfig models.PromptConfig) (*models.Project, error) {
	if _, err := s.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{"$set": bson.M{"prompt_config": promptConfig, "updated_at": time.Now().UTC()}}); err != nil {
		return nil, err
	}
	return s.Get(ctx, projectID)
}

func (s *ProjectService) enqueueGeneric(ctx context.Context, projectID string, taskType models.TaskType, payload map[string]any) (*models.Task, error) {
	if _, err := s.Get(ctx, projectID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	task := &models.Task{
		ID:        uuid.NewString(),
		Type:      taskType,
		Status:    models.TaskStatusPending,
		ProjectID: projectID,
		Payload:   payload,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.queue.Enqueue(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func NewSeriesService(ctx context.Context, collection *mongo.Collection, projects *ProjectService, q *queue.MongoQueue) (*SeriesService, error) {
	indexes := []mongo.IndexModel{{Keys: bson.D{{Key: "created_at", Value: -1}}}}
	if _, err := collection.Indexes().CreateMany(ctx, indexes); err != nil {
		return nil, err
	}
	return &SeriesService{series: collection, projects: projects, queue: q}, nil
}

func (s *SeriesService) Create(ctx context.Context, title string, description string, workflow string) (*models.Series, error) {
	now := time.Now().UTC()
	if workflow == "" {
		workflow = "r2v"
	}
	series := &models.Series{
		ID:            uuid.NewString(),
		Title:         strings.TrimSpace(title),
		Description:   description,
		Characters:    []models.Entity{},
		Scenes:        []models.Entity{},
		Props:         []models.Entity{},
		PromptConfig:  models.PromptConfig{},
		ModelSettings: defaultModelSettings(),
		Workflow:      workflow,
		EpisodeIDs:    []string{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if series.Title == "" {
		return nil, errors.New("title is required")
	}
	if _, err := s.series.InsertOne(ctx, series); err != nil {
		return nil, err
	}
	return series, nil
}

func (s *SeriesService) List(ctx context.Context) ([]models.Series, error) {
	cursor, err := s.series.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	series := make([]models.Series, 0)
	if err := cursor.All(ctx, &series); err != nil {
		return nil, err
	}
	return series, nil
}

func (s *SeriesService) Get(ctx context.Context, id string) (*models.Series, error) {
	var series models.Series
	err := s.series.FindOne(ctx, bson.M{"_id": id}).Decode(&series)
	if err == mongo.ErrNoDocuments {
		return nil, ErrNotFound
	}
	return &series, err
}

func (s *SeriesService) Update(ctx context.Context, id string, updates map[string]any) (*models.Series, error) {
	updates["updated_at"] = time.Now().UTC()
	result, err := s.series.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updates})
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, id)
}

func (s *SeriesService) Delete(ctx context.Context, id string) error {
	result, err := s.series.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SeriesService) RemoveProjectReferences(ctx context.Context, projectID string) error {
	cursor, err := s.series.Find(ctx, bson.M{"episode_ids": projectID})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	affected := make([]models.Series, 0)
	if err := cursor.All(ctx, &affected); err != nil {
		return err
	}
	if len(affected) == 0 {
		return nil
	}

	if _, err := s.series.UpdateMany(ctx, bson.M{"episode_ids": projectID}, bson.M{
		"$pull": bson.M{"episode_ids": projectID},
		"$set":  bson.M{"updated_at": time.Now().UTC()},
	}); err != nil {
		return err
	}
	for _, series := range affected {
		if err := s.reindexEpisodes(ctx, series.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *SeriesService) AddEpisode(ctx context.Context, seriesID string, projectID string) (*models.Series, error) {
	project, err := s.projects.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project.SeriesID != "" && project.SeriesID != seriesID {
		if _, err := s.series.UpdateOne(ctx, bson.M{"_id": project.SeriesID}, bson.M{"$pull": bson.M{"episode_ids": projectID}, "$set": bson.M{"updated_at": time.Now().UTC()}}); err != nil {
			return nil, err
		}
		_ = s.reindexEpisodes(ctx, project.SeriesID)
	}
	result, err := s.series.UpdateOne(ctx, bson.M{"_id": seriesID}, bson.M{"$addToSet": bson.M{"episode_ids": projectID}, "$set": bson.M{"updated_at": time.Now().UTC()}})
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, ErrNotFound
	}
	if err := s.reindexEpisodes(ctx, seriesID); err != nil {
		return nil, err
	}
	return s.Get(ctx, seriesID)
}

func (s *SeriesService) RemoveEpisode(ctx context.Context, seriesID string, projectID string) (*models.Series, error) {
	if _, err := s.series.UpdateOne(ctx, bson.M{"_id": seriesID}, bson.M{"$pull": bson.M{"episode_ids": projectID}, "$set": bson.M{"updated_at": time.Now().UTC()}}); err != nil {
		return nil, err
	}
	_, _ = s.projects.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{
		"$set":   bson.M{"series_id": "", "updated_at": time.Now().UTC()},
		"$unset": bson.M{"episode_number": ""},
	})
	if err := s.reindexEpisodes(ctx, seriesID); err != nil {
		return nil, err
	}
	return s.Get(ctx, seriesID)
}

func (s *SeriesService) reindexEpisodes(ctx context.Context, seriesID string) error {
	series, err := s.Get(ctx, seriesID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for index, projectID := range series.EpisodeIDs {
		if _, err := s.projects.projects.UpdateOne(ctx, bson.M{"_id": projectID}, bson.M{"$set": bson.M{
			"series_id":      seriesID,
			"episode_number": index + 1,
			"updated_at":     now,
		}}); err != nil {
			return err
		}
	}
	return nil
}

func defaultModelSettings() models.ModelSettings {
	return models.ModelSettings{
		T2IModel:              envString("DEFAULT_IMAGE_MODEL", "wan2.7-image-pro"),
		I2IModel:              envString("DEFAULT_IMAGE_MODEL", "wan2.7-image-pro"),
		ImageModel:            envString("DEFAULT_IMAGE_MODEL", "wan2.7-image-pro"),
		I2VModel:              envString("DEFAULT_VIDEO_MODEL", "wan2.7-i2v"),
		CharacterAspectRatio:  "9:16",
		SceneAspectRatio:      "16:9",
		PropAspectRatio:       "1:1",
		StoryboardAspectRatio: "9:16",
	}
}

func defaultArtDirection(analysis *models.ScriptAnalysis) *models.ArtDirection {
	description := "cinematic vertical short drama, coherent character design, consistent lighting and color palette"
	if analysis != nil && strings.TrimSpace(analysis.Synopsis) != "" {
		description = "cinematic vertical short drama based on: " + analysis.Synopsis
	}
	return &models.ArtDirection{
		SelectedStyleID: "cinematic",
		StyleConfig: map[string]any{
			"positive_prompt": "cinematic lighting, consistent character identity, production design continuity, detailed scene atmosphere",
			"negative_prompt": "low quality, inconsistent face, inconsistent costume, broken anatomy, unreadable text, watermark",
			"description":     description,
		},
		CustomStyles: []map[string]any{},
		AIRecommendations: []map[string]any{
			{
				"id":              "cinematic_default",
				"name":            "Cinematic Continuity",
				"description":     description,
				"positive_prompt": "film still, coherent lighting, consistent costume and props, detailed environment",
				"negative_prompt": "low quality, style drift, distorted hands, extra limbs, watermark",
			},
		},
	}
}

func assetField(assetType string) (string, error) {
	switch assetType {
	case "character", "characters":
		return "characters", nil
	case "scene", "scenes":
		return "scenes", nil
	case "prop", "props":
		return "props", nil
	default:
		return "", fmt.Errorf("invalid asset_type: %s", assetType)
	}
}

func assetPrefix(assetType string) string {
	switch assetType {
	case "scene", "scenes":
		return "scene"
	case "prop", "props":
		return "prop"
	default:
		return "char"
	}
}

func findEntity(project *models.Project, assetType string, assetID string) (*models.Entity, string, error) {
	field, err := assetField(assetType)
	if err != nil {
		return nil, "", err
	}
	var entities []models.Entity
	switch field {
	case "characters":
		entities = project.Characters
	case "scenes":
		entities = project.Scenes
	case "props":
		entities = project.Props
	}
	for i := range entities {
		if entities[i].ID == assetID {
			return &entities[i], field, nil
		}
	}
	return nil, field, ErrNotFound
}

func entityIndex(project *models.Project, assetType string, assetID string) (string, int, error) {
	field, err := assetField(assetType)
	if err != nil {
		return "", -1, err
	}
	var entities []models.Entity
	switch field {
	case "characters":
		entities = project.Characters
	case "scenes":
		entities = project.Scenes
	case "props":
		entities = project.Props
	}
	for i, entity := range entities {
		if entity.ID == assetID {
			return field, i, nil
		}
	}
	return field, -1, ErrNotFound
}

type imageVariantLocation struct {
	uploadType   string
	variants     []models.ImageVariant
	selectedID   string
	variantsPath string
	selectedPath string
}

func imageVariantState(project *models.Project, assetType string, assetID string, uploadType string) (imageVariantLocation, error) {
	field, index, err := entityIndex(project, assetType, assetID)
	if err != nil {
		return imageVariantLocation{}, err
	}
	entity := entityAt(project, field, index)
	uploadType = normalizeUploadType(uploadType)
	switch {
	case field == "characters" && uploadType == "full_body":
		return imageVariantLocation{
			uploadType:   uploadType,
			variants:     entity.FullBody.ImageVariants,
			selectedID:   entity.FullBody.SelectedImageID,
			variantsPath: fmt.Sprintf("%s.%d.full_body.image_variants", field, index),
			selectedPath: fmt.Sprintf("%s.%d.full_body.selected_image_id", field, index),
		}, nil
	case field == "characters" && uploadType == "three_views":
		return imageVariantLocation{
			uploadType:   uploadType,
			variants:     entity.ThreeViews.ImageVariants,
			selectedID:   entity.ThreeViews.SelectedImageID,
			variantsPath: fmt.Sprintf("%s.%d.three_views.image_variants", field, index),
			selectedPath: fmt.Sprintf("%s.%d.three_views.selected_image_id", field, index),
		}, nil
	case field == "characters" && uploadType == "head_shot":
		return imageVariantLocation{
			uploadType:   uploadType,
			variants:     entity.HeadShot.ImageVariants,
			selectedID:   entity.HeadShot.SelectedImageID,
			variantsPath: fmt.Sprintf("%s.%d.head_shot.image_variants", field, index),
			selectedPath: fmt.Sprintf("%s.%d.head_shot.selected_image_id", field, index),
		}, nil
	default:
		return imageVariantLocation{
			uploadType:   "image",
			variants:     entity.Image.Variants,
			selectedID:   entity.Image.SelectedID,
			variantsPath: fmt.Sprintf("%s.%d.image.variants", field, index),
			selectedPath: fmt.Sprintf("%s.%d.image.selected_id", field, index),
		}, nil
	}
}

type assetVideoVariantLocation struct {
	unitVideo    bool
	videosPath   string
	selectedPath string
	selectedID   string
}

func assetVideoLocation(project *models.Project, assetType string, assetID string, uploadType string) (assetVideoVariantLocation, error) {
	field, index, err := entityIndex(project, assetType, assetID)
	if err != nil {
		return assetVideoVariantLocation{}, err
	}
	entity := entityAt(project, field, index)
	uploadType = normalizeUploadType(uploadType)
	switch {
	case field == "characters" && uploadType == "full_body":
		return assetVideoVariantLocation{
			unitVideo:    true,
			videosPath:   fmt.Sprintf("%s.%d.full_body.video_variants", field, index),
			selectedPath: fmt.Sprintf("%s.%d.full_body.selected_video_id", field, index),
			selectedID:   entity.FullBody.SelectedVideoID,
		}, nil
	case field == "characters" && uploadType == "three_views":
		return assetVideoVariantLocation{
			unitVideo:    true,
			videosPath:   fmt.Sprintf("%s.%d.three_views.video_variants", field, index),
			selectedPath: fmt.Sprintf("%s.%d.three_views.selected_video_id", field, index),
			selectedID:   entity.ThreeViews.SelectedVideoID,
		}, nil
	case field == "characters" && uploadType == "head_shot":
		return assetVideoVariantLocation{
			unitVideo:    true,
			videosPath:   fmt.Sprintf("%s.%d.head_shot.video_variants", field, index),
			selectedPath: fmt.Sprintf("%s.%d.head_shot.selected_video_id", field, index),
			selectedID:   entity.HeadShot.SelectedVideoID,
		}, nil
	default:
		return assetVideoVariantLocation{videosPath: fmt.Sprintf("%s.%d.video_assets", field, index)}, nil
	}
}

func entityAt(project *models.Project, field string, index int) models.Entity {
	switch field {
	case "characters":
		return project.Characters[index]
	case "scenes":
		return project.Scenes[index]
	default:
		return project.Props[index]
	}
}

func normalizeUploadType(uploadType string) string {
	switch strings.TrimSpace(uploadType) {
	case "", "image":
		return "image"
	case "three_view", "three-views":
		return "three_views"
	case "headshot", "head-shot":
		return "head_shot"
	default:
		return strings.TrimSpace(uploadType)
	}
}

func containsImageVariant(variants []models.ImageVariant, variantID string) bool {
	for _, variant := range variants {
		if variant.ID == variantID {
			return true
		}
	}
	return false
}

func selectedUnitImageURL(unit models.AssetUnit) string {
	return selectedImageURL(models.ImageAsset{SelectedID: unit.SelectedImageID, Variants: unit.ImageVariants})
}

func selectedImageURL(asset models.ImageAsset) string {
	if asset.SelectedID != "" {
		for _, variant := range asset.Variants {
			if variant.ID == asset.SelectedID {
				return variant.URL
			}
		}
	}
	if len(asset.Variants) > 0 {
		return asset.Variants[0].URL
	}
	return ""
}

func assetGenerationPrompt(asset models.Entity, uploadType string) string {
	base := strings.TrimSpace(strings.Join([]string{asset.Name, asset.Description}, "，"))
	switch normalizeUploadType(uploadType) {
	case "full_body":
		return strings.TrimSpace(base + "\n无背景角色全身设定图，完整身体比例，站立中性姿态，服装、发型和主要配饰清晰。")
	case "three_views":
		return strings.TrimSpace(base + "\n角色三视图设定稿，同一角色正面、侧面、背面，服装和比例保持一致。")
	case "head_shot":
		return strings.TrimSpace(base + "\n头像特写设定图，脸部、发型、五官和表情细节清晰。")
	default:
		return base
	}
}

func buildProductionAudit(project *models.Project) *models.ProductionAuditReport {
	report := &models.ProductionAuditReport{
		ProjectID:       project.ID,
		Metrics:         map[string]int{},
		Checks:          map[string]bool{},
		Issues:          []models.ProductionAuditIssue{},
		Recommendations: []string{},
	}
	addIssue := func(severity string, area string, message string, action string) {
		report.Issues = append(report.Issues, models.ProductionAuditIssue{
			Severity: severity,
			Area:     area,
			Message:  message,
			Action:   action,
		})
	}

	report.Metrics["characters"] = len(project.Characters)
	report.Metrics["scenes"] = len(project.Scenes)
	report.Metrics["props"] = len(project.Props)
	report.Metrics["frames"] = len(project.Frames)
	report.Metrics["frame_images"] = countFrameImages(project.Frames)
	report.Metrics["frame_videos"] = countFrameVideos(project.Frames)
	report.Metrics["frame_last_frames"] = countFrameLastFrames(project.Frames)
	report.Metrics["character_full_body"] = countCharactersWithFullBody(project.Characters)
	report.Metrics["character_three_views"] = countCharactersWithThreeViews(project.Characters)
	report.Metrics["character_head_shot"] = countCharactersWithHeadShot(project.Characters)
	report.Metrics["scene_images"] = countEntitiesWithImage(project.Scenes)
	report.Metrics["prop_images"] = countEntitiesWithImage(project.Props)
	report.Metrics["referenced_frame_video_tasks"] = countReferencedFrameVideoTasks(project)
	report.Metrics["expired_reference_urls"] = countExpiredReferenceURLs(project)

	report.Checks["script"] = strings.TrimSpace(project.Script) != ""
	report.Checks["analysis"] = project.Analysis != nil
	report.Checks["art_direction"] = project.ArtDirection != nil
	report.Checks["characters"] = len(project.Characters) > 0
	report.Checks["character_full_body"] = len(project.Characters) == 0 || report.Metrics["character_full_body"] == len(project.Characters)
	report.Checks["character_three_views"] = len(project.Characters) == 0 || report.Metrics["character_three_views"] == len(project.Characters)
	report.Checks["character_head_shot"] = len(project.Characters) == 0 || report.Metrics["character_head_shot"] == len(project.Characters)
	report.Checks["scene_images"] = len(project.Scenes) == 0 || report.Metrics["scene_images"] == len(project.Scenes)
	report.Checks["prop_images"] = len(project.Props) == 0 || report.Metrics["prop_images"] == len(project.Props)
	report.Checks["storyboard"] = len(project.Frames) >= 4
	report.Checks["frame_images"] = len(project.Frames) > 0 && report.Metrics["frame_images"] == len(project.Frames)
	report.Checks["frame_videos"] = len(project.Frames) > 0 && report.Metrics["frame_videos"] == len(project.Frames)
	report.Checks["frame_references"] = len(project.Frames) == 0 || report.Metrics["referenced_frame_video_tasks"] >= len(project.Frames)
	report.Checks["frame_last_frames"] = len(project.Frames) == 0 || report.Metrics["frame_last_frames"] >= maxInt(0, len(project.Frames)-1)
	report.Checks["reference_urls_fresh"] = report.Metrics["expired_reference_urls"] == 0
	report.Checks["merged_video"] = strings.TrimSpace(project.MergedVideoURL) != ""
	report.Checks["model_settings"] = strings.TrimSpace(project.ModelSettings.ImageModel) != "" && strings.TrimSpace(project.ModelSettings.I2VModel) != ""
	report.Checks["frame_entity_refs"] = frameEntityReferencesResolvable(project)

	if !report.Checks["script"] {
		addIssue("error", "剧本", "项目没有剧本原文。", "在剧本页录入或导入剧本。")
	}
	if !report.Checks["analysis"] {
		addIssue("error", "剧本解析", "缺少 AI 剧本解析结果。", "重新运行剧本分析。")
	}
	if !report.Checks["art_direction"] {
		addIssue("warning", "美术方向", "缺少统一美术方向。", "先生成或保存美术方向，再生成资产和分镜。")
	}
	if !report.Checks["characters"] {
		addIssue("error", "角色资产", "没有角色资产，无法建立角色连续性。", "从剧本解析角色或手动添加角色。")
	}
	if !report.Checks["character_full_body"] {
		addIssue("error", "角色资产", "存在角色缺少全身图。", "在资产库补全角色全身图。")
	}
	if !report.Checks["character_three_views"] {
		addIssue("warning", "角色资产", "存在角色缺少三视图，一致性会下降。", "用全身图作为参考继续生成三视图。")
	}
	if !report.Checks["character_head_shot"] {
		addIssue("warning", "角色资产", "存在角色缺少头像特写，近景脸部一致性会下降。", "补全头像特写。")
	}
	if !report.Checks["scene_images"] {
		addIssue("warning", "场景资产", "存在场景缺少基准图。", "补全场景图。")
	}
	if !report.Checks["prop_images"] {
		addIssue("info", "道具资产", "存在道具缺少基准图。", "补全关键道具图。")
	}
	if !report.Checks["storyboard"] {
		addIssue("error", "分镜", "分镜数量不足，短剧至少应拆成 4 个镜头。", "重新运行分镜解析或手动增加镜头。")
	}
	if !report.Checks["frame_entity_refs"] {
		addIssue("warning", "分镜引用", "部分分镜的角色或场景引用无法匹配资产库。", "重新保存分镜角色/场景，或重新运行剧本分析。")
	}
	if !report.Checks["frame_images"] {
		addIssue("error", "首帧", "存在分镜缺少首帧图。", "在成片页执行一键补全或按资产重生成。")
	}
	if !report.Checks["frame_videos"] {
		addIssue("error", "视频", "存在分镜缺少视频片段。", "在成片页执行一键补全。")
	}
	if !report.Checks["frame_references"] {
		addIssue("warning", "一致性", "不是所有分镜视频任务都有资产参考图记录。", "使用“按资产重生成全片”重做分镜视频。")
	}
	if !report.Checks["reference_urls_fresh"] {
		addIssue("warning", "参考资产", "存在已过期或即将过期的临时签名参考图，后续重生成可能无法访问。", "刷新关键资产图，或改用长期可访问的对象存储 URL。")
	}
	if !report.Checks["frame_last_frames"] {
		addIssue("info", "镜头衔接", "部分分镜缺少视频末帧记录，后续镜头无法充分承接上一镜头状态。", "使用新版本“按资产重生成全片”生成并记录末帧参考。")
	}
	if !report.Checks["merged_video"] {
		addIssue("error", "成片", "还没有拼接成片。", "生成所有分镜视频后执行拼接。")
	}
	if !report.Checks["model_settings"] {
		addIssue("error", "模型配置", "图片或视频模型配置不完整。", "在项目设置中配置 image_model 与 i2v_model。")
	}

	weights := map[string]int{
		"script":                6,
		"analysis":              8,
		"art_direction":         8,
		"characters":            6,
		"character_full_body":   10,
		"character_three_views": 6,
		"character_head_shot":   6,
		"scene_images":          6,
		"prop_images":           3,
		"storyboard":            8,
		"frame_entity_refs":     8,
		"frame_images":          8,
		"frame_videos":          10,
		"frame_references":      8,
		"frame_last_frames":     6,
		"reference_urls_fresh":  6,
		"merged_video":          6,
		"model_settings":        3,
	}
	total := 0
	score := 0
	for key, weight := range weights {
		total += weight
		if report.Checks[key] {
			score += weight
		}
	}
	if total > 0 {
		report.Score = score * 100 / total
	}
	report.EfficiencyScore = productionEfficiencyScore(report)
	report.ConsistencyScore = productionConsistencyScore(report)
	switch {
	case report.Score >= 90:
		report.Level = "production_ready"
	case report.Score >= 75:
		report.Level = "review_required"
	default:
		report.Level = "not_ready"
	}
	report.ReadyForPreview = report.Checks["frame_images"] && report.Checks["frame_videos"]
	report.ReadyForExport = report.ReadyForPreview && report.Checks["merged_video"] && report.Score >= 85

	report.WorkflowStages = productionWorkflowStages(project, report)
	report.FrameAudits = productionFrameAudits(project)
	report.Recommendations = productionRecommendations(report)
	return report
}

func productionRecommendations(report *models.ProductionAuditReport) []string {
	recommendations := []string{}
	if !report.Checks["character_full_body"] || !report.Checks["scene_images"] || !report.Checks["prop_images"] {
		recommendations = append(recommendations, "先补齐资产库，再生成分镜首帧，避免每个镜头独立发散。")
	}
	if !report.Checks["character_three_views"] || !report.Checks["character_head_shot"] {
		recommendations = append(recommendations, "角色至少保留全身图、三视图、头像特写三类参考，近景和远景分别使用不同参考。")
	}
	if !report.Checks["frame_references"] {
		recommendations = append(recommendations, "用“按资产重生成全片”重做镜头，使每个视频任务都带 reference_image_urls。")
	}
	if !report.Checks["frame_entity_refs"] {
		recommendations = append(recommendations, "修正分镜中的角色/场景引用，确保每个镜头能自动找到对应资产。")
	}
	if report.Score >= 90 {
		recommendations = append(recommendations, "当前已适合预览交付；最终发布前仍建议人工抽检脸部、服装、关键道具和镜头衔接。")
	}
	return recommendations
}

func productionEfficiencyScore(report *models.ProductionAuditReport) int {
	checks := []string{"script", "analysis", "art_direction", "storyboard", "frame_images", "frame_videos", "merged_video"}
	return percentChecks(report.Checks, checks)
}

func productionConsistencyScore(report *models.ProductionAuditReport) int {
	checks := []string{"character_full_body", "character_three_views", "character_head_shot", "scene_images", "prop_images", "frame_entity_refs", "frame_references", "frame_last_frames"}
	return percentChecks(report.Checks, checks)
}

func percentChecks(values map[string]bool, keys []string) int {
	if len(keys) == 0 {
		return 0
	}
	passed := 0
	for _, key := range keys {
		if values[key] {
			passed++
		}
	}
	return passed * 100 / len(keys)
}

func productionWorkflowStages(project *models.Project, report *models.ProductionAuditReport) []models.ProductionWorkflowStage {
	stage := func(id string, label string, done bool, blocked bool, detail string, action string) models.ProductionWorkflowStage {
		status := "warning"
		if done {
			status = "done"
		}
		if blocked {
			status = "blocked"
		}
		return models.ProductionWorkflowStage{ID: id, Label: label, Status: status, Detail: detail, Action: action}
	}
	return []models.ProductionWorkflowStage{
		stage("script", "剧本与分集", report.Checks["script"], !report.Checks["script"], fmt.Sprintf("剧本长度 %d 字，集数标记 %d", len([]rune(project.Script)), project.EpisodeNumber), "补齐剧本原文和分集信息"),
		stage("analysis", "AI 拆解", report.Checks["analysis"] && report.Checks["art_direction"], !report.Checks["analysis"], fmt.Sprintf("角色 %d，场景 %d，道具 %d", len(project.Characters), len(project.Scenes), len(project.Props)), "重新运行剧本分析和美术方向分析"),
		stage("assets", "资产定稿", report.Checks["character_full_body"] && report.Checks["scene_images"] && report.Checks["prop_images"], !report.Checks["characters"], fmt.Sprintf("角色全身 %d/%d，场景 %d/%d，道具 %d/%d", report.Metrics["character_full_body"], report.Metrics["characters"], report.Metrics["scene_images"], report.Metrics["scenes"], report.Metrics["prop_images"], report.Metrics["props"]), "补齐角色三视图、头像、场景和关键道具"),
		stage("storyboard", "分镜与首帧", report.Checks["storyboard"] && report.Checks["frame_images"], !report.Checks["storyboard"], fmt.Sprintf("首帧 %d/%d", report.Metrics["frame_images"], report.Metrics["frames"]), "补齐分镜并生成首帧"),
		stage("video", "镜头视频", report.Checks["frame_videos"] && report.Checks["frame_references"], !report.Checks["frame_images"], fmt.Sprintf("视频 %d/%d，带参考 %d/%d", report.Metrics["frame_videos"], report.Metrics["frames"], report.Metrics["referenced_frame_video_tasks"], report.Metrics["frames"]), "按资产重生成全片以提高连续性"),
		stage("assembly", "成片交付", report.Checks["merged_video"], !report.Checks["frame_videos"], fmt.Sprintf("导出状态：%s", boolLabel(report.ReadyForExport)), "完成拼接并人工抽检"),
	}
}

func productionFrameAudits(project *models.Project) []models.ProductionFrameAudit {
	referenceCounts := frameReferenceCounts(project)
	audits := make([]models.ProductionFrameAudit, 0, len(project.Frames))
	for _, frame := range project.Frames {
		hasImage := selectedFrameImageURL(frame) != ""
		hasVideo := strings.TrimSpace(frame.VideoURL) != ""
		referenceCount := referenceCounts[frame.ID]
		risk := "low"
		switch {
		case !hasImage || !hasVideo:
			risk = "high"
		case referenceCount == 0:
			risk = "high"
		case referenceCount < 3:
			risk = "medium"
		}
		audits = append(audits, models.ProductionFrameAudit{
			FrameID:        frame.ID,
			Index:          frame.Index,
			HasImage:       hasImage,
			HasVideo:       hasVideo,
			HasLastFrame:   strings.TrimSpace(frame.LastFrameURL) != "",
			ReferenceCount: referenceCount,
			Risk:           risk,
		})
	}
	sort.Slice(audits, func(i, j int) bool { return audits[i].Index < audits[j].Index })
	return audits
}

func frameReferenceCounts(project *models.Project) map[string]int {
	counts := map[string]int{}
	for _, task := range project.VideoTasks {
		if task.Status != models.TaskStatusCompleted || task.FrameID == "" {
			continue
		}
		total := len(task.ReferenceImageURLs) + len(task.ReferenceVideoURLs)
		if total > counts[task.FrameID] {
			counts[task.FrameID] = total
		}
	}
	return counts
}

func boolLabel(value bool) string {
	if value {
		return "可交付"
	}
	return "需复核"
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func countFrameImages(frames []models.StoryboardShot) int {
	count := 0
	for _, frame := range frames {
		if selectedFrameImageURL(frame) != "" {
			count++
		}
	}
	return count
}

func countFrameVideos(frames []models.StoryboardShot) int {
	count := 0
	for _, frame := range frames {
		if strings.TrimSpace(frame.VideoURL) != "" {
			count++
		}
	}
	return count
}

func countFrameLastFrames(frames []models.StoryboardShot) int {
	count := 0
	for _, frame := range frames {
		if strings.TrimSpace(frame.LastFrameURL) != "" {
			count++
		}
	}
	return count
}

func countCharactersWithFullBody(characters []models.Entity) int {
	count := 0
	for _, character := range characters {
		if selectedUnitImageURL(character.FullBody) != "" {
			count++
		}
	}
	return count
}

func countCharactersWithThreeViews(characters []models.Entity) int {
	count := 0
	for _, character := range characters {
		if selectedUnitImageURL(character.ThreeViews) != "" {
			count++
		}
	}
	return count
}

func countCharactersWithHeadShot(characters []models.Entity) int {
	count := 0
	for _, character := range characters {
		if selectedUnitImageURL(character.HeadShot) != "" {
			count++
		}
	}
	return count
}

func countEntitiesWithImage(entities []models.Entity) int {
	count := 0
	for _, entity := range entities {
		if selectedImageURL(entity.Image) != "" {
			count++
		}
	}
	return count
}

func countReferencedFrameVideoTasks(project *models.Project) int {
	frameIDs := map[string]bool{}
	for _, frame := range project.Frames {
		frameIDs[frame.ID] = true
	}
	referenced := map[string]bool{}
	for _, task := range project.VideoTasks {
		if task.Status != models.TaskStatusCompleted || strings.TrimSpace(task.FrameID) == "" || !frameIDs[task.FrameID] {
			continue
		}
		if len(task.ReferenceImageURLs) > 0 || len(task.ReferenceVideoURLs) > 0 {
			referenced[task.FrameID] = true
		}
	}
	return len(referenced)
}

func countExpiredReferenceURLs(project *models.Project) int {
	count := 0
	for _, url := range selectedProjectReferenceURLs(project) {
		if referenceURLExpired(url, 30*time.Minute) {
			count++
		}
	}
	return count
}

func selectedProjectReferenceURLs(project *models.Project) []string {
	urls := []string{}
	for _, character := range project.Characters {
		urls = appendAuditReferenceURL(urls, selectedUnitImageURL(character.FullBody))
		urls = appendAuditReferenceURL(urls, selectedUnitImageURL(character.ThreeViews))
		urls = appendAuditReferenceURL(urls, selectedUnitImageURL(character.HeadShot))
		urls = appendAuditReferenceURL(urls, selectedImageURL(character.Image))
	}
	for _, scene := range project.Scenes {
		urls = appendAuditReferenceURL(urls, selectedImageURL(scene.Image))
	}
	for _, prop := range project.Props {
		urls = appendAuditReferenceURL(urls, selectedImageURL(prop.Image))
	}
	for _, frame := range project.Frames {
		urls = appendAuditReferenceURL(urls, selectedFrameImageURL(frame))
		urls = appendAuditReferenceURL(urls, frame.LastFrameURL)
	}
	return urls
}

func appendAuditReferenceURL(urls []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return urls
	}
	for _, existing := range urls {
		if existing == value {
			return urls
		}
	}
	return append(urls, value)
}

func frameEntityReferencesResolvable(project *models.Project) bool {
	if len(project.Frames) == 0 {
		return false
	}
	characterAliases := entityAliasMap(project.Characters)
	sceneAliases := entityAliasMap(project.Scenes)
	for _, frame := range project.Frames {
		if frame.SceneID != "" && sceneAliases[normalizeReferenceKey(frame.SceneID)] == "" {
			return false
		}
		for _, characterID := range frame.CharacterIDs {
			if characterID != "" && characterAliases[normalizeReferenceKey(characterID)] == "" {
				return false
			}
		}
	}
	return true
}

func cloneMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}

func stringFromAny(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func intFromAny(value any, fallback int) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return fallback
	}
}
