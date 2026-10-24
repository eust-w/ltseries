# LumenX Feature Parity Matrix for ltseries

## LumenX Functional Areas

1. Project and script
   - Create/list/get/delete projects.
   - Parse or reparse scripts into characters, scenes, props, and shots.
   - Sync descriptions from script entities into asset workspaces.

2. Series
   - Create/list/update/delete series.
   - Attach/detach episode projects.
   - Store shared characters, scenes, props, art direction, model settings, and prompt config.
   - Import assets from another series.
   - Split imported files into episodes.

3. Art direction
   - Analyze script for visual style recommendations.
   - Save selected style, custom style config, and AI recommendations.
   - Load built-in presets.

4. Assets
   - Character, scene, and prop CRUD.
   - Generate asset image variants through a provider.
   - Upload source images.
   - Select, favorite, and delete variants.
   - Lock/unlock assets.
   - Generate motion reference videos for assets.

5. Storyboard
   - Analyze text into storyboard frames.
   - Add, update, copy, delete, reorder, and lock frames.
   - Refine storyboard prompts through LLM.
   - Render storyboard images through an image provider.
   - Upload/select frame images.

6. Video
   - Create image-to-video/reference-to-video tasks.
   - Persist success/failure task records.
   - Select video per frame.
   - Extract last frame from a completed video.
   - Polish video prompts through LLM.

7. Audio and final mix
   - Assign voice to characters.
   - Generate dialogue audio.
   - Generate SFX and BGM.

8. Assembly and export
   - Merge selected frame videos with FFmpeg.
   - Export merged result.

9. Settings
   - Environment/provider configuration.
   - Model settings per project/series.
   - Prompt configuration per project/series.
   - System check endpoint.

## ltseries Coverage Strategy

`ltseries` implements the same workflow surface using Go, Gin, MongoDB, and React. Provider-backed generation never returns fabricated media: DashScope/Bailian is the built-in provider for script analysis, image assets, storyboard renders, and video tasks. Optional external provider endpoints can still be configured for custom media services. If the key, required media input, or provider route is missing, the MongoDB task is marked `failed` with the concrete error.

MongoDB stores durable project, series, and task state. Refreshing the frontend restores success/failure history from `/api/tasks`.

Current validated path:

- Script analysis through `qwen-plus` produced art direction, assets, and storyboard frames.
- Asset image generation through `wan2.7-image-pro` completed and stored a selected image variant.
- Merge/export uses FFmpeg over selected or completed frame videos and records the merge task result.
