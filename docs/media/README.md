# LTSeries Showcase Media

This directory documents the public product showcase. Screenshots are real application captures, not UI mockups. The videos are edited presentations, not uninterrupted recordings of model latency.

## Watch and Download

- [Landscape walkthrough, Chinese narration](https://github.com/eust-w/ltseries/releases/download/showcase-2026-10/ltseries-landscape-zh.mp4)
- [Portrait walkthrough, Chinese narration](https://github.com/eust-w/ltseries/releases/download/showcase-2026-10/ltseries-portrait-zh.mp4)
- [Real generated film, 30 seconds](https://github.com/eust-w/ltseries/releases/download/showcase-2026-10/ltseries-generated-sample.mp4)
- [Chinese subtitles](https://github.com/eust-w/ltseries/releases/download/showcase-2026-10/ltseries-zh.srt)
- [English subtitles](https://github.com/eust-w/ltseries/releases/download/showcase-2026-10/ltseries-en.srt)
- [All media and covers](https://github.com/eust-w/ltseries/releases/tag/showcase-2026-10)

Video files are release assets, not Git blobs. This keeps normal clones small. Screenshots and compressed GIF previews live in the repository.

## What the Media Shows

| Material | Provenance | What it does not prove |
| --- | --- | --- |
| Script, art, asset, storyboard and task screenshots | A real October 25, 2026 run of the local LTSeries application using its backend and DashScope integration | Every provider or model working, production-scale throughput, or a fully completed new episode |
| Script analysis | The [example script](../examples/rain-letter.md), resulting in 1 character, 3 scenes, 3 props and 6 shots | Guaranteed extraction counts or a quality benchmark |
| Character images | Three completed image-generation tasks for the full-body, three-view and headshot slots | That the model perfectly obeyed framing, anatomy or identity constraints |
| First frame | One completed storyboard-render task | That all six new shot videos were generated |
| Episodes | A real series created and linked to the current demo project | Automatic long-form episode splitting in this particular run |
| Assembly screenshot | The current incomplete project's real readiness report | A 100-point quality pass or a newly completed film |
| Film excerpt and 30-second sample | A historical LTSeries six-shot, 1080 x 1920 assembled video from the project's existing local generated media | Current-run regeneration, guaranteed character consistency, or absence of visual artifacts |
| Workflow GIF | Edited sequence of real screenshots | A continuous screen recording or generation-speed measurement |
| Narration | macOS Tingting synthetic speech, added in post-production | A built-in, turnkey application voice-generation feature |

The screenshot project and the historical film are separate productions. They use a related fictional rain-night story, but the film is not presented as the direct result of the new screenshot run.

No account settings, credentials, signed provider URLs, private filesystem paths or database exports are included in the public media package. There is no third-party music track. Screen text and AI imagery may contain model artifacts; review generated text and incidental brand-like shapes before reusing them in commercial work.

## Reproduce the Presentation

Rendering uses Node.js, the isolated `@napi-rs/canvas` dependency, FFmpeg and macOS `say`. It does not call an AI API and does not require Python. To reproduce on another operating system, replace the narration step with your own licensed audio or TTS implementation and provide a CJK-capable font.

```bash
npm ci --prefix scripts/promo
node scripts/promo/render.mjs \
  --sample /absolute/path/to/your-generated-film.mp4 \
  --output artifacts/promo
node scripts/promo/poster.mjs artifacts/promo
```

Optional flags:

- `--font /absolute/path/to/CJK-font.ttc`: font used for the presentation text.
- `--voice Tingting`: installed macOS narration voice.

The source sample is deliberately not bundled in Git. Download the release sample, or supply your own rights-cleared output. The script uses the checked-in real screenshots. Replacing them should involve capturing actual application states, not editing success statuses or inventing generated results.

Sources:

- [Narration, scene order, captions and translations](storyboard.json)
- [Renderer](../../scripts/promo/render.mjs)
- [README poster compositor](../../scripts/promo/poster.mjs)
- [Renderer dependencies](../../scripts/promo/package.json)
- [Platform copy and publishing checklist](publishing-kit.md)

Outputs include landscape and portrait H.264/AAC MP4s, JPEG covers, Chinese and English SRT files, the sample MP4 and `verification.json` with codec, size and duration metadata. Local renders go into ignored `artifacts/`; no build cache, raw capture or credentials should be committed.

## Publishing State

The showcase release is the distribution point for this repository. **The presence of an export does not mean it has been posted to a social account.** Social-platform destination, account and visibility must be confirmed separately, and a successful publication needs its own public URL or platform status.

The software's MIT license does not override model-provider terms, font licenses, voice rights or third-party rights in inputs. The images and film are AI-generated and should be disclosed as such where appropriate.
