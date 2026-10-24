package models

import "time"

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
)

type TaskType string

const (
	TaskTypeAnalysis           TaskType = "analysis"
	TaskTypeArtDirection       TaskType = "art_direction"
	TaskTypeAssetGeneration    TaskType = "asset_generation"
	TaskTypeAssetVideo         TaskType = "asset_video"
	TaskTypeStoryboardAnalysis TaskType = "storyboard_analysis"
	TaskTypeStoryboardRender   TaskType = "storyboard_render"
	TaskTypeVideoGeneration    TaskType = "video_generation"
	TaskTypeAudioGeneration    TaskType = "audio_generation"
	TaskTypeSFXGeneration      TaskType = "sfx_generation"
	TaskTypeBGMGeneration      TaskType = "bgm_generation"
	TaskTypeMerge              TaskType = "merge"
	TaskTypeStoryboardPipeline TaskType = "storyboard_pipeline"
)

type GenerationStatus string

const (
	GenerationStatusPending    GenerationStatus = "pending"
	GenerationStatusProcessing GenerationStatus = "processing"
	GenerationStatusCompleted  GenerationStatus = "completed"
	GenerationStatusFailed     GenerationStatus = "failed"
)

type ImageVariant struct {
	ID             string    `bson:"id" json:"id"`
	URL            string    `bson:"url" json:"url"`
	Prompt         string    `bson:"prompt,omitempty" json:"prompt,omitempty"`
	Favorited      bool      `bson:"favorited" json:"favorited"`
	UploadedSource bool      `bson:"uploaded_source" json:"uploaded_source"`
	UploadType     string    `bson:"upload_type,omitempty" json:"upload_type,omitempty"`
	CreatedAt      time.Time `bson:"created_at" json:"created_at"`
}

type ImageAsset struct {
	SelectedID string         `bson:"selected_id,omitempty" json:"selected_id,omitempty"`
	Variants   []ImageVariant `bson:"variants" json:"variants"`
}

type VideoVariant struct {
	ID        string    `bson:"id" json:"id"`
	URL       string    `bson:"url" json:"url"`
	Prompt    string    `bson:"prompt,omitempty" json:"prompt,omitempty"`
	AudioURL  string    `bson:"audio_url,omitempty" json:"audio_url,omitempty"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

type AssetUnit struct {
	SelectedImageID string         `bson:"selected_image_id,omitempty" json:"selected_image_id,omitempty"`
	ImageVariants   []ImageVariant `bson:"image_variants" json:"image_variants"`
	SelectedVideoID string         `bson:"selected_video_id,omitempty" json:"selected_video_id,omitempty"`
	VideoVariants   []VideoVariant `bson:"video_variants" json:"video_variants"`
	ImagePrompt     string         `bson:"image_prompt,omitempty" json:"image_prompt,omitempty"`
	VideoPrompt     string         `bson:"video_prompt,omitempty" json:"video_prompt,omitempty"`
}

type Entity struct {
	ID          string           `bson:"id" json:"id"`
	Name        string           `bson:"name" json:"name"`
	Description string           `bson:"description" json:"description"`
	Attributes  map[string]any   `bson:"attributes,omitempty" json:"attributes,omitempty"`
	Locked      bool             `bson:"locked" json:"locked"`
	Status      GenerationStatus `bson:"status" json:"status"`
	Image       ImageAsset       `bson:"image" json:"image"`
	FullBody    AssetUnit        `bson:"full_body" json:"full_body"`
	ThreeViews  AssetUnit        `bson:"three_views" json:"three_views"`
	HeadShot    AssetUnit        `bson:"head_shot" json:"head_shot"`
	VideoAssets []VideoTask      `bson:"video_assets" json:"video_assets"`
	VoiceID     string           `bson:"voice_id,omitempty" json:"voice_id,omitempty"`
	VoiceName   string           `bson:"voice_name,omitempty" json:"voice_name,omitempty"`
	VoiceSpeed  float64          `bson:"voice_speed,omitempty" json:"voice_speed,omitempty"`
	VoicePitch  float64          `bson:"voice_pitch,omitempty" json:"voice_pitch,omitempty"`
	VoiceVolume int              `bson:"voice_volume,omitempty" json:"voice_volume,omitempty"`
}

type StoryboardShot struct {
	ID              string           `bson:"id" json:"id"`
	Index           int              `bson:"index" json:"index"`
	SceneID         string           `bson:"scene_id,omitempty" json:"scene_id,omitempty"`
	CharacterIDs    []string         `bson:"character_ids,omitempty" json:"character_ids,omitempty"`
	Action          string           `bson:"action" json:"action"`
	Camera          string           `bson:"camera,omitempty" json:"camera,omitempty"`
	Dialogue        string           `bson:"dialogue,omitempty" json:"dialogue,omitempty"`
	Prompt          string           `bson:"prompt,omitempty" json:"prompt,omitempty"`
	ImageURL        string           `bson:"image_url,omitempty" json:"image_url,omitempty"`
	RenderedURL     string           `bson:"rendered_url,omitempty" json:"rendered_url,omitempty"`
	VideoURL        string           `bson:"video_url,omitempty" json:"video_url,omitempty"`
	LastFrameURL    string           `bson:"last_frame_url,omitempty" json:"last_frame_url,omitempty"`
	AudioURL        string           `bson:"audio_url,omitempty" json:"audio_url,omitempty"`
	SFXURL          string           `bson:"sfx_url,omitempty" json:"sfx_url,omitempty"`
	SelectedVideoID string           `bson:"selected_video_id,omitempty" json:"selected_video_id,omitempty"`
	ImageAsset      ImageAsset       `bson:"image_asset" json:"image_asset"`
	RenderedAsset   ImageAsset       `bson:"rendered_asset" json:"rendered_asset"`
	Locked          bool             `bson:"locked" json:"locked"`
	Status          GenerationStatus `bson:"status" json:"status"`
}

type ScriptAnalysis struct {
	Synopsis     string           `bson:"synopsis" json:"synopsis"`
	ArtDirection *ArtDirection    `bson:"art_direction,omitempty" json:"art_direction,omitempty"`
	Characters   []Entity         `bson:"characters" json:"characters"`
	Scenes       []Entity         `bson:"scenes" json:"scenes"`
	Props        []Entity         `bson:"props" json:"props"`
	Shots        []StoryboardShot `bson:"shots" json:"shots"`
}

type ModelSettings struct {
	T2IModel              string `bson:"t2i_model" json:"t2i_model"`
	I2IModel              string `bson:"i2i_model" json:"i2i_model"`
	ImageModel            string `bson:"image_model" json:"image_model"`
	I2VModel              string `bson:"i2v_model" json:"i2v_model"`
	CharacterAspectRatio  string `bson:"character_aspect_ratio" json:"character_aspect_ratio"`
	SceneAspectRatio      string `bson:"scene_aspect_ratio" json:"scene_aspect_ratio"`
	PropAspectRatio       string `bson:"prop_aspect_ratio" json:"prop_aspect_ratio"`
	StoryboardAspectRatio string `bson:"storyboard_aspect_ratio" json:"storyboard_aspect_ratio"`
}

type ArtDirection struct {
	SelectedStyleID   string           `bson:"selected_style_id,omitempty" json:"selected_style_id,omitempty"`
	StyleConfig       map[string]any   `bson:"style_config,omitempty" json:"style_config,omitempty"`
	CustomStyles      []map[string]any `bson:"custom_styles" json:"custom_styles"`
	AIRecommendations []map[string]any `bson:"ai_recommendations" json:"ai_recommendations"`
}

type PromptConfig struct {
	StoryboardPolish string `bson:"storyboard_polish" json:"storyboard_polish"`
	VideoPolish      string `bson:"video_polish" json:"video_polish"`
	R2VPolish        string `bson:"r2v_polish" json:"r2v_polish"`
}

type VideoTask struct {
	ID                 string         `bson:"id" json:"id"`
	ProjectID          string         `bson:"project_id" json:"project_id"`
	FrameID            string         `bson:"frame_id,omitempty" json:"frame_id,omitempty"`
	AssetID            string         `bson:"asset_id,omitempty" json:"asset_id,omitempty"`
	ImageURL           string         `bson:"image_url,omitempty" json:"image_url,omitempty"`
	Prompt             string         `bson:"prompt" json:"prompt"`
	Status             TaskStatus     `bson:"status" json:"status"`
	Error              string         `bson:"error,omitempty" json:"error,omitempty"`
	VideoURL           string         `bson:"video_url,omitempty" json:"video_url,omitempty"`
	Duration           int            `bson:"duration" json:"duration"`
	Seed               int64          `bson:"seed,omitempty" json:"seed,omitempty"`
	Resolution         string         `bson:"resolution,omitempty" json:"resolution,omitempty"`
	GenerateAudio      bool           `bson:"generate_audio" json:"generate_audio"`
	AudioURL           string         `bson:"audio_url,omitempty" json:"audio_url,omitempty"`
	NegativePrompt     string         `bson:"negative_prompt,omitempty" json:"negative_prompt,omitempty"`
	Model              string         `bson:"model,omitempty" json:"model,omitempty"`
	GenerationMode     string         `bson:"generation_mode,omitempty" json:"generation_mode,omitempty"`
	ReferenceImageURLs []string       `bson:"reference_image_urls" json:"reference_image_urls"`
	ReferenceVideoURLs []string       `bson:"reference_video_urls" json:"reference_video_urls"`
	Parameters         map[string]any `bson:"parameters,omitempty" json:"parameters,omitempty"`
	CreatedAt          time.Time      `bson:"created_at" json:"created_at"`
}

type Project struct {
	ID             string           `bson:"_id" json:"id"`
	Title          string           `bson:"title" json:"title"`
	Script         string           `bson:"script" json:"script"`
	Workflow       string           `bson:"workflow" json:"workflow"`
	Analysis       *ScriptAnalysis  `bson:"analysis,omitempty" json:"analysis,omitempty"`
	Characters     []Entity         `bson:"characters" json:"characters"`
	Scenes         []Entity         `bson:"scenes" json:"scenes"`
	Props          []Entity         `bson:"props" json:"props"`
	Frames         []StoryboardShot `bson:"frames" json:"frames"`
	VideoTasks     []VideoTask      `bson:"video_tasks" json:"video_tasks"`
	StylePreset    string           `bson:"style_preset" json:"style_preset"`
	StylePrompt    string           `bson:"style_prompt,omitempty" json:"style_prompt,omitempty"`
	ArtDirection   *ArtDirection    `bson:"art_direction,omitempty" json:"art_direction,omitempty"`
	ModelSettings  ModelSettings    `bson:"model_settings" json:"model_settings"`
	PromptConfig   PromptConfig     `bson:"prompt_config" json:"prompt_config"`
	MergedVideoURL string           `bson:"merged_video_url,omitempty" json:"merged_video_url,omitempty"`
	SeriesID       string           `bson:"series_id,omitempty" json:"series_id,omitempty"`
	EpisodeNumber  int              `bson:"episode_number,omitempty" json:"episode_number,omitempty"`
	CreatedAt      time.Time        `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time        `bson:"updated_at" json:"updated_at"`
}

type ProductionAuditIssue struct {
	Severity string `json:"severity"`
	Area     string `json:"area"`
	Message  string `json:"message"`
	Action   string `json:"action,omitempty"`
}

type ProductionWorkflowStage struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Status string `json:"status"`
	Detail string `json:"detail"`
	Action string `json:"action,omitempty"`
}

type ProductionFrameAudit struct {
	FrameID        string `json:"frame_id"`
	Index          int    `json:"index"`
	HasImage       bool   `json:"has_image"`
	HasVideo       bool   `json:"has_video"`
	HasLastFrame   bool   `json:"has_last_frame"`
	ReferenceCount int    `json:"reference_count"`
	Risk           string `json:"risk"`
}

type ProductionAuditReport struct {
	ProjectID        string                    `json:"project_id"`
	Score            int                       `json:"score"`
	EfficiencyScore  int                       `json:"efficiency_score"`
	ConsistencyScore int                       `json:"consistency_score"`
	Level            string                    `json:"level"`
	ReadyForPreview  bool                      `json:"ready_for_preview"`
	ReadyForExport   bool                      `json:"ready_for_export"`
	Metrics          map[string]int            `json:"metrics"`
	Checks           map[string]bool           `json:"checks"`
	WorkflowStages   []ProductionWorkflowStage `json:"workflow_stages"`
	FrameAudits      []ProductionFrameAudit    `json:"frame_audits"`
	Issues           []ProductionAuditIssue    `json:"issues"`
	Recommendations  []string                  `json:"recommendations"`
}

type Series struct {
	ID            string        `bson:"_id" json:"id"`
	Title         string        `bson:"title" json:"title"`
	Description   string        `bson:"description" json:"description"`
	Characters    []Entity      `bson:"characters" json:"characters"`
	Scenes        []Entity      `bson:"scenes" json:"scenes"`
	Props         []Entity      `bson:"props" json:"props"`
	ArtDirection  *ArtDirection `bson:"art_direction,omitempty" json:"art_direction,omitempty"`
	PromptConfig  PromptConfig  `bson:"prompt_config" json:"prompt_config"`
	ModelSettings ModelSettings `bson:"model_settings" json:"model_settings"`
	Workflow      string        `bson:"workflow" json:"workflow"`
	EpisodeIDs    []string      `bson:"episode_ids" json:"episode_ids"`
	CreatedAt     time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time     `bson:"updated_at" json:"updated_at"`
}

type Task struct {
	ID         string         `bson:"_id" json:"id"`
	Type       TaskType       `bson:"type" json:"type"`
	Status     TaskStatus     `bson:"status" json:"status"`
	Progress   int            `bson:"progress" json:"progress"`
	Error      string         `bson:"error,omitempty" json:"error,omitempty"`
	ProjectID  string         `bson:"project_id,omitempty" json:"project_id,omitempty"`
	Payload    map[string]any `bson:"payload,omitempty" json:"payload,omitempty"`
	Result     map[string]any `bson:"result,omitempty" json:"result,omitempty"`
	WorkerID   string         `bson:"worker_id,omitempty" json:"worker_id,omitempty"`
	Attempts   int            `bson:"attempts" json:"attempts"`
	CreatedAt  time.Time      `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time      `bson:"updated_at" json:"updated_at"`
	StartedAt  *time.Time     `bson:"started_at,omitempty" json:"started_at,omitempty"`
	FinishedAt *time.Time     `bson:"finished_at,omitempty" json:"finished_at,omitempty"`
}
