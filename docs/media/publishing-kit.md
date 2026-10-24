# LTSeries 发布素材包

以下是可直接调整使用的发布文案，不是已发布记录。横屏、竖屏、封面和字幕统一在 [Showcase Release](https://github.com/eust-w/ltseries/releases/tag/showcase-2026-10) 下载。

## 素材选择

| 用途 | 文件 | 内容 |
| --- | --- | --- |
| 横屏完整介绍 | `ltseries-landscape-zh.mp4` | 1920 x 1080，中文合成解说，英文画面要点 |
| 竖屏产品介绍 | `ltseries-portrait-zh.mp4` | 1080 x 1920，中文合成解说与画面要点 |
| 单独展示生成结果 | `ltseries-generated-sample.mp4` | 30 秒，1080 x 1920，无音轨，历史真实生成样片 |
| 横屏 / 竖屏封面 | `cover-landscape.jpg` / `cover-portrait.jpg` | 与视频对应的实际样片画面 |
| 完整解说字幕 | `ltseries-zh.srt` / `ltseries-en.srt` | 中文字幕与英文翻译，按章节对齐 |

这些是通用导出规格，不代表所有平台、账号和上传入口的限制完全相同。发布前以当前平台校验结果为准；不要把横屏画面强行拉伸成竖屏。

## B站

**标题：** 我把 AI 短剧生产流程做成了开源工作台：剧本、资产、分镜到成片 | LTSeries

**简介：**

生成一段视频之后，角色参考放哪里？分镜怎么改？失败任务怎么追？

LTSeries 把这些步骤放到同一个自托管工作台：中文剧本分析、美术方向、角色三视图/头像参考、分镜首帧、镜头视频，以及 FFmpeg 拼接。后端 Go + Gin + MongoDB，前端 React，接入真实百炼 API。

视频包含真实界面、实际图像生成结果和一段历史六镜头样片。它不是实时速度测试，也不保证角色百分百一致；云模型需要自己的 API Key 和额度。

源码与部署：https://github.com/eust-w/ltseries

欢迎提交可复现的问题、模型适配或创作流程反馈。

**标签：** 开源项目、AI视频、AI漫剧、独立开发、Go、React

## 抖音 / 视频号

**标题：** AI 短剧不止生成视频，还需要一个生产工作台

**正文：**

把剧本、美术方向、角色参考、分镜和任务记录放在一起。LTSeries 已开源，Go + React + MongoDB，自托管运行，真实模型 API 调用。

画面为 AI 生成样片及真实软件界面，解说为合成语音。模型结果仍需人工审核，不承诺百分百角色一致性。源码：GitHub 搜索 eust-w/ltseries。

**话题：** #AI短剧 #AI漫剧 #开源项目 #独立开发 #AI视频

## 小红书

**标题：** 把 AI 漫剧流程做成了开源工作台

**正文：**

做 AI 影像时，生成按钮之外还有很多重复工作：整理角色参考、改分镜、找结果、追失败任务。

我把这些流程放进了 LTSeries：

1. 输入剧本，提取角色、场景、道具和分镜。
2. 调整美术方向，建立角色全身图、三视图、头像参考。
3. 逐镜编辑动作和镜头语言，先看首帧再生成视频。
4. 任务记录保存在数据库里，刷新后也能继续追踪。
5. 选好镜头，拼接成片。

Go + React + MongoDB，MIT 开源，可以自己部署。需要自备模型 API Key 和额度；角色一致性仍要人工检查。演示含 AI 生成画面与合成解说。

GitHub：eust-w/ltseries

**话题：** #AI创作 #AI漫剧 #开源工具 #独立开发 #工作流

## YouTube

**Title:** LTSeries: An Open-Source AI Short-Film Studio | Script to Assets, Storyboards & Assembly

**Description:**

An AI video workflow needs more than a prompt box. LTSeries brings scripts, art direction, reusable character references, shot editing, generation jobs and FFmpeg assembly into one self-hosted workspace.

Stack: Go, Gin, MongoDB, React and TypeScript. Generation uses real external APIs; you bring your own credentials and provider credits.

This walkthrough contains actual application screens and provider-generated images, plus a historical six-shot sample. It is edited, not a real-time generation benchmark. Visual consistency still requires human review. Chinese synthetic narration; English subtitles are provided.

Source and setup: https://github.com/eust-w/ltseries

Contributions, reproducible issues and creator feedback are welcome.

**Tags:** LTSeries, open source, AI filmmaking, AI video, storyboard, self hosted, Go, React

## X

**Post:**

AI video needs more than a prompt box.

LTSeries puts scripts, character references, storyboards, durable jobs and FFmpeg assembly in one self-hosted studio.

Go + React + MongoDB. Real APIs. Human review still matters.

Code + demo: https://github.com/eust-w/ltseries

## 发布前检查

- 确认平台、账号、标题、可见性和素材版本。
- 开启平台要求的 AI 生成内容标识；按要求注明合成解说。
- 检查封面和字幕安全区、音量、中文字体、画面比例。
- 模型样片不要标为实拍，不要宣称实测节省比例、全自动零返工或百分百一致。
- 不在画面、字幕、简介或评论中放入真实 API Key、后台登录信息、签名资源链接。
- 上传完成不等于发布成功；分别记录上传、处理、审核、公开状态和最终链接。
