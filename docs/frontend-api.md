# ltseries Frontend API 文档

本文档面向前端开发，描述 ltseries Go/Gin 后端当前暴露的 HTTP API、认证方式、通用数据结构、任务轮询方式、媒体 URL 处理方式以及主要业务调用顺序。

## 1. 基础信息

- 默认 API Base URL: `http://localhost:18080/api`
- 默认前端 URL: `http://localhost:3108`
- 媒体文件 Base URL: API Base 去掉 `/api` 后拼接后端返回的 `/files/...`
- 当前前端封装文件: `frontend/src/lib/api.ts`
- JSON 请求默认使用 `Content-Type: application/json`
- 文件上传使用 `multipart/form-data`

前端处理媒体 URL 的规则：

```ts
function mediaUrl(url?: string) {
  if (!url) return '';
  if (/^https?:\/\//i.test(url)) return url;
  return `${MEDIA_BASE}${url.startsWith('/') ? url : `/${url}`}`;
}
```

## 2. 认证

认证开关由后端环境变量控制。未启用认证时，所有 API 等同已登录；启用后，除 `/health`、`/auth/status`、`/auth/login` 外均需要 `Authorization: Bearer <token>`。

### GET `/auth/status`

用途：检查认证是否启用以及当前 token 是否有效。

响应：

```ts
type AuthStatus = {
  enabled: boolean;
  authenticated: boolean;
  username?: string;
};
```

### POST `/auth/login`

请求：

```json
{
  "username": "admin",
  "password": "123456"
}
```

响应：

```ts
type LoginResponse = {
  token: string;
  expires_at: number;
  auth_disabled?: boolean;
};
```

前端要求：

- 登录成功后保存 token 到 `localStorage`。
- 请求返回 unauthorized 时清除 token 并跳转登录页。

## 3. 通用错误格式

大多数错误返回：

```json
{
  "error": "message"
}
```

前端要求：

- `400`: 参数错误，直接展示错误信息。
- `401`: 登录失效，清 token 并跳转登录页。
- `404`: 资源不存在，刷新项目/任务列表。
- `500`: 后端或 provider 配置/执行错误，展示错误信息。

## 4. 核心类型

### Task

```ts
type TaskStatus = 'pending' | 'processing' | 'completed' | 'failed';

type Task = {
  id: string;
  type:
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
};
```

### ProductionAuditReport

`GET /projects/:id/production-audit` 返回项目生产体检结果，用于前端判断是否适合预览、交付，以及哪些镜头存在一致性风险。

```ts
type ProductionAuditReport = {
  project_id: string;
  score: number;
  efficiency_score: number;
  consistency_score: number;
  level: 'production_ready' | 'review_required' | 'not_ready' | string;
  ready_for_preview: boolean;
  ready_for_export: boolean;
  metrics: Record<string, number>;
  checks: Record<string, boolean>;
  workflow_stages: Array<{
    id: string;
    label: string;
    status: 'done' | 'warning' | 'blocked' | string;
    detail: string;
    action?: string;
  }>;
  frame_audits: Array<{
    frame_id: string;
    index: number;
    has_image: boolean;
    has_video: boolean;
    has_last_frame: boolean;
    reference_count: number;
    risk: 'low' | 'medium' | 'high' | string;
  }>;
  issues: Array<{ severity: string; area: string; message: string; action?: string }>;
  recommendations: string[];
};
```

任务状态流转：

```text
pending -> processing -> completed
pending -> processing -> failed
```

### Project

```ts
type Project = {
  id: string;
  title: string;
  script: string;
  workflow: 'r2v' | 'i2v' | 't2v' | string;
  analysis?: ScriptAnalysis;
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
};
```

### Entity

`Entity` 表示角色、场景或道具。

```ts
type Entity = {
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
};
```

角色使用 `full_body`、`three_views`、`head_shot` 三个主槽位；场景和道具主要使用 `image`。

### ImageAsset / AssetUnit

```ts
type ImageVariant = {
  id: string;
  url: string;
  prompt?: string;
  favorited: boolean;
  uploaded_source: boolean;
  upload_type?: string;
  created_at: string;
};

type ImageAsset = {
  selected_id?: string;
  variants: ImageVariant[];
};

type AssetUnit = {
  selected_image_id?: string;
  image_variants: ImageVariant[];
  selected_video_id?: string;
  video_variants: VideoVariant[];
  image_prompt?: string;
  video_prompt?: string;
};
```

可用 `upload_type`：

- `image`: 通用图、场景图、道具图。
- `full_body`: 角色全身图。
- `three_views`: 角色三视图。
- `head_shot`: 角色头像特写。

### StoryboardShot

```ts
type StoryboardShot = {
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
  audio_url?: string;
  sfx_url?: string;
  selected_video_id?: string;
  image_asset: ImageAsset;
  rendered_asset: ImageAsset;
  locked: boolean;
  status: string;
};
```

## 5. 系统与配置 API

### GET `/health`

用途：健康检查。

响应：

```json
{
  "status": "ok",
  "service": "ltseries"
}
```

### GET `/system/check`

用途：检查系统运行状态和 provider 路由。

响应字段：

- `service`: 服务名。
- `mongo`: MongoDB 队列是否可用。
- `storage_dir`: 后端文件存储目录。
- `provider_routing`: 当前 AI provider 路由。

### GET `/models`

用途：获取模型目录。

响应：

```ts
type ModelCatalog = {
  llm: ModelOption[];
  image: ModelOption[];
  video: ModelOption[];
};
```

### GET `/config`

用途：读取运行时配置。敏感 key 不会明文返回，只返回 `*_key_set`。

响应：

```ts
type RuntimeConfig = {
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
};
```

### PUT `/config`

用途：保存运行时配置。

请求：`RuntimeConfig`

前端要求：

- 密钥输入框为空时不要覆盖已有密钥，除非用户明确清空。
- 保存后重新读取配置并刷新模型选择默认值。

## 6. 通用上传 API

### POST `/upload`

用途：上传单个文件，返回可预览 URL。

请求：`multipart/form-data`

- `file`: 文件。

响应：

```json
{
  "url": "/files/uploads/<filename>"
}
```

## 7. 项目 API

### POST `/projects`

用途：创建项目。

请求：

```json
{
  "title": "项目标题",
  "script": "剧本原文",
  "workflow": "r2v"
}
```

响应：`Project`

### GET `/projects?limit=50`

用途：获取项目列表，默认按创建时间倒序。

响应：`Project[]`

### GET `/projects/:id`

用途：获取项目详情。

响应：`Project`

### PUT `/projects/:id`

用途：修改项目基础信息。

请求：

```json
{
  "title": "新标题",
  "script": "新的剧本原文",
  "workflow": "i2v"
}
```

响应：`Project`

### DELETE `/projects/:id`

用途：删除项目。

响应：

```json
{
  "status": "deleted",
  "id": "<project_id>"
}
```

## 8. 剧本分析与美术方向 API

### POST `/projects/:id/analyze`

用途：从剧本自动生成分析、角色、场景、道具、分镜和艺术方向。

响应：

```json
{
  "task_id": "<task_id>",
  "task": {}
}
```

任务类型：`analysis`

### POST `/projects/:id/reparse`

用途：重新解析当前剧本。

任务类型：`analysis`

### POST `/projects/:id/art-direction/analyze`

用途：单独生成/刷新美术方向。

任务类型：`art_direction`

### POST `/projects/:id/art-direction`

用途：手动保存美术方向。

请求：

```json
{
  "selected_style_id": "cinematic",
  "style_config": {
    "positive_prompt": "...",
    "negative_prompt": "...",
    "palette": "..."
  },
  "custom_styles": [],
  "ai_recommendations": []
}
```

响应：`Project`

### GET `/art-direction/presets`

用途：获取内置风格预设。

响应：

```json
{
  "presets": []
}
```

## 9. 资产 API

`asset_type` 可用值：

- `character`
- `scene`
- `prop`

### GET `/projects/:id/assets`

用途：获取项目资产集合。

响应：

```json
{
  "characters": [],
  "scenes": [],
  "props": []
}
```

### POST `/projects/:id/assets/:type`

用途：新增角色/场景/道具。

请求：

```json
{
  "name": "角色名",
  "description": "外观、身份、服装、连续性描述"
}
```

响应：`Project`

### PUT `/projects/:id/assets/:type/:asset_id`

用途：修改资产字段。

请求示例：

```json
{
  "name": "新名称",
  "description": "新描述",
  "voice_id": "voice_xxx"
}
```

响应：`Project`

### DELETE `/projects/:id/assets/:type/:asset_id`

用途：删除资产。

响应：`Project`

### POST `/projects/:id/assets/:type/:asset_id/toggle-lock`

用途：锁定/解锁资产。锁定资产不参与批量补齐生成。

响应：`Project`

### POST `/projects/:id/assets/:type/:asset_id/upload?upload_type=full_body`

用途：上传资产图片变体。

请求：`multipart/form-data`

- `file`: 图片文件。
- `prompt`: 图片说明，可选。

响应：

```json
{
  "url": "/files/uploads/<filename>",
  "project": {}
}
```

### POST `/projects/:id/assets/:type/:asset_id/generate`

用途：为指定资产槽位生成图片。

请求：

```json
{
  "asset_type": "full_body",
  "prompt": "生成提示词",
  "model": "wan2.7-image-pro",
  "inputs": {
    "reference_image_urls": ["/files/uploads/ref.png"]
  },
  "parameters": {
    "aspect_ratio": "9:16",
    "n": 1,
    "watermark": false
  }
}
```

响应：`{ "task_id": string, "task": Task }`

任务类型：`asset_generation`

### POST `/projects/:id/generate_assets`

用途：一键补齐缺失资产。规则：

- 角色无全身图时，先生成 `full_body`。
- 角色已有全身图但无三视图时，使用全身图作为参考生成 `three_views`。
- 角色已有全身图但无头像特写时，使用全身图作为参考生成 `head_shot`。
- 场景和道具无图时生成 `image`。
- `locked=true` 的资产跳过。

请求：

```json
{
  "model": "wan2.7-image-pro",
  "parameters": {
    "aspect_ratio": "9:16"
  }
}
```

响应：

```json
{
  "task_ids": ["..."],
  "tasks": []
}
```

### POST `/projects/:id/assets/:type/:asset_id/variant/select`

用途：选择图片变体。

请求：

```json
{
  "upload_type": "full_body",
  "variant_id": "variant_xxx"
}
```

响应：`Project`

### POST `/projects/:id/assets/variant/favorite`

用途：收藏/取消收藏图片变体。

请求：

```json
{
  "asset_id": "char_xxx",
  "asset_type": "character",
  "variant_id": "variant_xxx",
  "upload_type": "full_body",
  "is_favorited": true
}
```

响应：`Project`

### POST `/projects/:id/assets/variant/delete`

用途：删除图片变体。若删除的是当前选中图，后端会自动切换到剩余第一张或清空选中。

请求：

```json
{
  "asset_id": "char_xxx",
  "asset_type": "character",
  "variant_id": "variant_xxx",
  "upload_type": "full_body"
}
```

响应：`Project`

### POST `/projects/:id/assets/:type/:asset_id/generate_video`

用途：基于当前选中的资产图生成资产参考视频。

请求：

```json
{
  "prompt": "角色转身展示服装",
  "upload_type": "full_body",
  "model": "wan2.7-i2v",
  "generation_mode": "i2v",
  "duration": 5,
  "inputs": {},
  "parameters": {
    "ratio": "9:16"
  }
}
```

响应：`{ "task_id": string, "task": Task }`

任务类型：`asset_video`

### DELETE `/projects/:id/assets/:type/:asset_id/videos/:video_id?upload_type=full_body`

用途：删除资产视频。

响应：`Project`

## 10. 视频任务 API

### POST `/projects/:id/videos`

用途：旧式视频任务入口。建议新前端优先使用 `/video-tasks`。

请求：

```json
{
  "prompt": "...",
  "model": "wan2.7-i2v",
  "inputs": {},
  "parameters": {}
}
```

响应：`{ "task_id": string, "task": Task }`

### POST `/projects/:id/video-tasks`

用途：为镜头创建视频生成任务。

请求：

```json
{
  "frame_id": "frame_xxx",
  "image_url": "/files/uploads/shot.png",
  "prompt": "视频提示词",
  "model": "wan2.7-i2v",
  "generation_mode": "i2v",
  "duration": 5,
  "generate_audio": true,
  "parameters": {
    "ratio": "9:16",
    "resolution": "720P"
  }
}
```

响应：`{ "task_id": string, "task": Task }`

任务类型：`video_generation`

## 11. 分镜 API

### POST `/projects/:id/storyboard/analyze`

用途：将剧本文本拆成可编辑分镜。

请求：

```json
{
  "action": "要拆分的剧本文本；为空时使用项目 script"
}
```

响应：`{ "task_id": string, "task": Task }`

任务类型：`storyboard_analysis`

### POST `/projects/:id/storyboard/refine_prompt`

用途：润色分镜图 prompt。

请求：

```json
{
  "prompt": "原始 prompt"
}
```

响应：

```json
{
  "prompt": "润色后的 prompt"
}
```

### POST `/projects/:id/storyboard/render`

用途：生成分镜图。

请求：

```json
{
  "frame_id": "frame_xxx",
  "prompt": "分镜图提示词",
  "references": ["/files/uploads/ref.png"]
}
```

响应：`{ "task_id": string, "task": Task }`

任务类型：`storyboard_render`

### POST `/projects/:id/frames`

用途：手动新增镜头。

请求：

```json
{
  "action": "镜头动作",
  "camera": "medium shot",
  "scene_id": "scene_xxx"
}
```

响应：`Project`

### PUT `/projects/:id/frames/:frame_id`

用途：修改镜头。

请求示例：

```json
{
  "action": "新的动作",
  "prompt": "新的 prompt",
  "dialogue": "台词",
  "camera": "close-up"
}
```

响应：`Project`

### POST `/projects/:id/frames/:frame_id/toggle-lock`

用途：锁定/解锁镜头。

响应：`Project`

### POST `/projects/:id/frames/toggle_lock`

用途：LumenX 兼容形式。

请求：

```json
{
  "frame_id": "frame_xxx"
}
```

响应：`Project`

### DELETE `/projects/:id/frames/:frame_id`

用途：删除镜头。

响应：`Project`

### POST `/projects/:id/frames/:frame_id/copy`

用途：复制镜头。

响应：`Project`

### PUT `/projects/:id/frames/reorder`

用途：调整镜头顺序。

请求：

```json
{
  "frame_ids": ["frame_1", "frame_2"]
}
```

响应：`Project`

### POST `/projects/:id/frames/:frame_id/upload_image`

用途：上传或替换分镜图。

请求：`multipart/form-data`

- `file`: 图片。
- `prompt`: 可选。

响应：

```json
{
  "url": "/files/uploads/<filename>",
  "project": {}
}
```

### POST `/projects/:id/frames/:frame_id/extract_last_frame`

用途：从当前选中视频或指定视频任务抽取末帧，并写入当前镜头分镜图。

请求：

```json
{
  "video_task_id": "video_xxx"
}
```

响应：

```json
{
  "url": "/files/uploads/lastframe_xxx.png",
  "project": {}
}
```

### POST `/projects/:id/frames/:frame_id/select-video`

用途：选择镜头视频。

请求：

```json
{
  "video_task_id": "video_xxx"
}
```

响应：`Project`

## 12. 音频、音效、BGM API

### POST `/projects/:id/frames/:frame_id/audio`

用途：为镜头生成对白/旁白音频。

请求：

```json
{
  "audio_type": "dialogue",
  "text": "台词或旁白内容",
  "voice_id": "voice_xxx"
}
```

响应：`{ "task_id": string, "task": Task }`

任务类型：`audio_generation`

### POST `/projects/:id/mix/generate_sfx`

用途：生成音效。

请求：

```json
{
  "frame_id": "frame_xxx",
  "prompt": "雨声、脚步声、门被推开"
}
```

响应：`{ "task_id": string, "task": Task }`

任务类型：`sfx_generation`

### POST `/projects/:id/mix/generate_bgm`

用途：生成 BGM。

请求：

```json
{
  "frame_id": "frame_xxx",
  "prompt": "紧张悬疑氛围音乐"
}
```

响应：`{ "task_id": string, "task": Task }`

任务类型：`bgm_generation`

## 13. 拼接与导出 API

### POST `/projects/:id/merge`

用途：使用 FFmpeg 拼接已选镜头视频。

响应：

- 如果已创建任务：`{ "task_id": string, "task": Task }`
- 任务完成后项目会写入 `merged_video_url`

任务类型：`merge`

### POST `/projects/:id/export`

用途：导出成片。如果已有 `merged_video_url`，直接返回 URL；否则创建拼接任务。

响应：

```json
{
  "url": "/files/video/merged_xxx.mp4"
}
```

或：

```json
{
  "task_id": "...",
  "task": {}
}
```

## 14. 项目模型与 Prompt 配置 API

### PUT `/projects/:id/model-settings`

请求：

```json
{
  "t2i_model": "wan2.7-image-pro",
  "i2i_model": "wan2.7-image-pro",
  "image_model": "wan2.7-image-pro",
  "i2v_model": "wan2.7-i2v",
  "character_aspect_ratio": "9:16",
  "scene_aspect_ratio": "16:9",
  "prop_aspect_ratio": "1:1",
  "storyboard_aspect_ratio": "9:16"
}
```

响应：`Project`

### PUT `/projects/:id/prompt-config`

请求：

```json
{
  "storyboard_polish": "...",
  "video_polish": "...",
  "r2v_polish": "..."
}
```

响应：`Project`

## 15. Prompt 润色 API

### POST `/video/polish_prompt`

用途：润色 I2V/T2V 视频 prompt。

请求：

```json
{
  "prompt": "原始视频 prompt"
}
```

响应：

```json
{
  "prompt": "润色后的 prompt"
}
```

### POST `/video/polish_r2v_prompt`

用途：润色 R2V prompt。

请求同上。

响应同上。

## 16. 角色声音 API

### GET `/voices`

用途：获取声音列表及 provider 状态。

响应：

```json
{
  "voices": [],
  "provider_configured": false
}
```

### POST `/projects/:id/characters/:char_id/voice`

用途：为角色绑定声音。

请求：

```json
{
  "voice_id": "voice_xxx",
  "voice_name": "声音名称"
}
```

响应：`Project`

### PUT `/projects/:id/characters/:char_id/voice_params`

用途：设置角色声音参数。

请求：

```json
{
  "voice_speed": 1,
  "voice_pitch": 1,
  "voice_volume": 50
}
```

响应：`Project`

## 17. 任务 API

### GET `/tasks?project_id=<project_id>&status=&type=&limit=100`

用途：获取任务历史。刷新页面后任务仍从 MongoDB 恢复。

响应：`Task[]`

### GET `/tasks/:id`

用途：获取单任务详情。

响应：`Task`

### DELETE `/tasks/:id`

用途：删除任务记录。

响应：

```json
{
  "status": "deleted",
  "id": "<task_id>"
}
```

前端轮询要求：

- 项目页每 3 秒刷新当前项目任务列表。
- 任务完成或失败后刷新项目详情，回显生成的图片、视频、音频、拼接结果。
- `failed` 任务必须展示 `error`。

## 18. 系列与分集 API

### POST `/series`

用途：创建系列。

请求：

```json
{
  "title": "系列标题",
  "description": "系列描述",
  "workflow": "r2v"
}
```

响应：`Series`

### GET `/series`

用途：获取系列列表。

响应：`Series[]`

### GET `/series/:id`

用途：获取系列详情。

响应：`Series`

### PUT `/series/:id`

用途：更新系列字段。

请求示例：

```json
{
  "title": "新系列标题",
  "description": "新描述"
}
```

响应：`Series`

### DELETE `/series/:id`

用途：删除系列。

响应：

```json
{
  "status": "deleted"
}
```

### POST `/series/:id/episodes`

用途：把项目加入系列，后端自动维护 `episode_number`。

请求：

```json
{
  "project_id": "project_xxx"
}
```

响应：`Series`

### DELETE `/series/:id/episodes/:project_id`

用途：从系列中移除项目。

响应：`Series`

## 19. 前端推荐调用流程

### 新项目完整流程

```text
POST /projects
POST /projects/:id/analyze
GET /tasks/:task_id 或 GET /tasks?project_id=:id
GET /projects/:id
POST /projects/:id/art-direction 或 POST /projects/:id/art-direction/analyze
POST /projects/:id/generate_assets
轮询任务并刷新 Project
POST /projects/:id/storyboard/analyze
逐镜头 PUT /frames/:frame_id
POST /projects/:id/storyboard/render
POST /projects/:id/video-tasks
POST /projects/:id/frames/:frame_id/audio
POST /projects/:id/merge
POST /projects/:id/export
```

### 资产页推荐逻辑

- 创建资产后立即回显空槽位。
- 上传图片后用响应中的 `project` 刷新本地状态。
- 生成任务入队后刷新任务列表，不要假设立即有图片。
- 任务完成后重新拉取 `GET /projects/:id`，使用对应 `selected_*_id` 展示图/视频。

### 分镜页推荐逻辑

- `frames` 为空时展示手动添加和 AI 拆镜入口。
- `rendered_url` 优先作为分镜图预览。
- `video_url` 优先作为视频预览；如有多视频任务，应结合 `selected_video_id` 和 `video_tasks` 展示可选列表。
- 删除/复制/锁定后使用返回的 `Project` 或重新拉取项目。

## 20. 不应在前端硬编码的内容

- API key。
- provider endpoint。
- DashScope base URL。
- 模型 ID 默认值之外的业务判断。
- 任务成功后的媒体 URL。
- 假图片、假视频、假音频。

前端只应展示后端真实返回的数据；任务失败时展示任务 `error`。
