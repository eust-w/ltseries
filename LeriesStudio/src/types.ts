/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

export type TaskStatus = 'pending' | 'processing' | 'completed' | 'failed';

export type TaskType =
  | 'analysis'
  | 'art_direction'
  | 'asset_generation'
  | 'asset_video'
  | 'storyboard_analysis'
  | 'storyboard_render'
  | 'video_generation'
  | 'audio_generation'
  | 'sfx_generation'
  | 'bgm_generation'
  | 'merge'
  | 'storyboard_pipeline';

export interface Task {
  id: string;
  type: TaskType;
  status: TaskStatus;
  progress: number;
  error?: string;
  project_id?: string;
  payload?: Record<string, unknown>;
  result?: Record<string, unknown>;
  worker_id?: string;
  attempts: number;
  created_at: string;
  updated_at: string;
  started_at?: string;
  finished_at?: string;
}

export interface ImageVariant {
  id: string;
  url: string;
  prompt?: string;
  favorited: boolean;
  uploaded_source: boolean;
  upload_type?: string;
  created_at: string;
}

export interface ImageAsset {
  selected_id?: string;
  variants: ImageVariant[];
}

export interface VideoVariant {
  id: string;
  url: string;
  prompt?: string;
  created_at: string;
}

export interface AssetUnit {
  selected_image_id?: string;
  image_variants: ImageVariant[];
  selected_video_id?: string;
  video_variants: VideoVariant[];
  image_prompt?: string;
  video_prompt?: string;
}

export interface Entity {
  id: string;
  name: string;
  description: string;
  attributes?: Record<string, unknown>;
  locked: boolean;
  status: 'pending' | 'processing' | 'completed' | 'failed' | string;
  image: ImageAsset;
  full_body: AssetUnit;
  three_views: AssetUnit;
  head_shot: AssetUnit;
  video_assets: VideoTask[];
  voice_id?: string;
  voice_name?: string;
  voice_speed?: number;
  voice_pitch?: number;
  voice_volume?: number;
}

export interface StoryboardShot {
  id: string;
  index: number;
  scene_id?: string;
  character_ids?: string[];
  action: string;
  camera?: string;
  dialogue?: string;
  prompt?: string;
  image_url?: string;
  rendered_url?: string;
  video_url?: string;
  last_frame_url?: string;
  audio_url?: string;
  sfx_url?: string;
  selected_video_id?: string;
  image_asset: ImageAsset;
  rendered_asset: ImageAsset;
  locked: boolean;
  status: string;
}

export interface ArtDirection {
  selected_style_id?: string;
  style_config?: {
    positive_prompt?: string;
    negative_prompt?: string;
    palette?: string;
  };
  custom_styles?: any[];
  ai_recommendations?: any[];
}

export interface Project {
  id: string;
  title: string;
  script: string;
  workflow: 'r2v' | 'i2v' | 't2v' | 'novel2script' | string;
  analysis?: any;
  characters: Entity[];
  scenes: Entity[];
  props: Entity[];
  frames: StoryboardShot[];
  video_tasks: VideoTask[];
  style_preset: string;
  style_prompt?: string;
  art_direction?: ArtDirection;
  model_settings: ModelSettings;
  prompt_config: PromptConfig;
  merged_video_url?: string;
  series_id?: string;
  episode_number?: number;
  created_at: string;
  updated_at: string;
}

export interface VideoTask {
  id: string;
  project_id: string;
  frame_id?: string;
  asset_id?: string;
  image_url?: string;
  prompt: string;
  status: TaskStatus;
  error?: string;
  video_url?: string;
  duration: number;
  resolution?: string;
  model?: string;
  generation_mode?: string;
  reference_image_urls?: string[];
  reference_video_urls?: string[];
  created_at: string;
}

export interface ModelSettings {
  t2i_model: string;
  i2i_model: string;
  image_model: string;
  i2v_model: string;
  character_aspect_ratio: string;
  scene_aspect_ratio: string;
  prop_aspect_ratio: string;
  storyboard_aspect_ratio: string;
}

export interface PromptConfig {
  storyboard_polish: string;
  video_polish: string;
  r2v_polish: string;
}

export interface Series {
  id: string;
  title: string;
  description: string;
  characters: Entity[];
  scenes: Entity[];
  props: Entity[];
  art_direction?: ArtDirection;
  prompt_config: PromptConfig;
  model_settings: ModelSettings;
  workflow: string;
  episode_ids: string[];
  created_at: string;
  updated_at: string;
}

export interface AuthStatus {
  enabled: boolean;
  authenticated: boolean;
  username?: string;
}

export interface RuntimeConfig {
  ai_provider: string;
  dashscope_base_url: string;
  dashscope_api_key?: string;
  dashscope_api_key_set: boolean;
  llm_base_url: string;
  llm_model: string;
  llm_api_key_set: boolean;
  default_image_model: string;
  default_video_model: string;
  asset_endpoint_configured: boolean;
  video_endpoint_configured: boolean;
  audio_endpoint_configured: boolean;
}

export interface ProductionAuditIssue {
  severity: 'error' | 'warning' | 'info' | string;
  area: string;
  message: string;
  action?: string;
}

export interface ProductionWorkflowStage {
  id: string;
  label: string;
  status: 'done' | 'warning' | 'blocked' | string;
  detail: string;
  action?: string;
}

export interface ProductionFrameAudit {
  frame_id: string;
  index: number;
  has_image: boolean;
  has_video: boolean;
  has_last_frame: boolean;
  reference_count: number;
  risk: 'low' | 'medium' | 'high' | string;
}

export interface ProductionAuditReport {
  project_id: string;
  score: number;
  efficiency_score: number;
  consistency_score: number;
  level: 'production_ready' | 'review_required' | 'not_ready' | string;
  ready_for_preview: boolean;
  ready_for_export: boolean;
  metrics: Record<string, number>;
  checks: Record<string, boolean>;
  workflow_stages: ProductionWorkflowStage[];
  frame_audits: ProductionFrameAudit[];
  issues: ProductionAuditIssue[];
  recommendations: string[];
}
