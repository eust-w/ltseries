package services

type ModelOption struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	Capability  string `json:"capability"`
	Mode        string `json:"mode,omitempty"`
	Description string `json:"description,omitempty"`
	Recommended bool   `json:"recommended"`
}

type ModelCatalog struct {
	LLM   []ModelOption `json:"llm"`
	Image []ModelOption `json:"image"`
	Video []ModelOption `json:"video"`
}

func DefaultModelCatalog() ModelCatalog {
	return ModelCatalog{
		LLM: []ModelOption{
			{ID: "qwen-plus", Name: "Qwen Plus", Provider: "dashscope", Capability: "llm", Description: "默认剧本解析与分镜拆解模型", Recommended: true},
			{ID: "qwen-turbo", Name: "Qwen Turbo", Provider: "dashscope", Capability: "llm", Description: "更快的草稿解析"},
			{ID: "qwen-max", Name: "Qwen Max", Provider: "dashscope", Capability: "llm", Description: "更强的长文本理解"},
			{ID: "qwen-long", Name: "Qwen Long", Provider: "dashscope", Capability: "llm", Description: "适合长剧本输入"},
		},
		Image: []ModelOption{
			{ID: "wan2.7-image-pro", Name: "Wan 2.7 Image Pro", Provider: "dashscope", Capability: "image", Mode: "t2i/i2i", Description: "高质量首帧、角色、场景资产", Recommended: true},
			{ID: "wan2.7-image", Name: "Wan 2.7 Image", Provider: "dashscope", Capability: "image", Mode: "t2i/i2i", Description: "通用资产生成"},
			{ID: "qwen-image-2.0-pro", Name: "Qwen Image 2.0 Pro", Provider: "dashscope", Capability: "image", Mode: "t2i/i2i", Description: "文字理解更强的图片生成"},
			{ID: "qwen-image-2.0", Name: "Qwen Image 2.0", Provider: "dashscope", Capability: "image", Mode: "t2i/i2i"},
			{ID: "wan2.6-image", Name: "Wan 2.6 Image", Provider: "dashscope", Capability: "image", Mode: "i2i"},
			{ID: "wan2.6-t2i", Name: "Wan 2.6 T2I", Provider: "dashscope", Capability: "image", Mode: "t2i"},
			{ID: "wan2.2-t2i-plus", Name: "Wan 2.2 T2I Plus", Provider: "dashscope", Capability: "image", Mode: "t2i"},
			{ID: "wan2.2-t2i-flash", Name: "Wan 2.2 T2I Flash", Provider: "dashscope", Capability: "image", Mode: "t2i"},
		},
		Video: []ModelOption{
			{ID: "wan2.7-i2v", Name: "Wan 2.7 I2V", Provider: "dashscope", Capability: "video", Mode: "i2v", Description: "首帧生视频，适合镜头生成", Recommended: true},
			{ID: "wan2.7-r2v", Name: "Wan 2.7 R2V", Provider: "dashscope", Capability: "video", Mode: "r2v", Description: "参考图生视频，适合风格一致性"},
			{ID: "wan2.7-t2v", Name: "Wan 2.7 T2V", Provider: "dashscope", Capability: "video", Mode: "t2v", Description: "文本生视频"},
			{ID: "wan2.6-i2v", Name: "Wan 2.6 I2V", Provider: "dashscope", Capability: "video", Mode: "i2v"},
			{ID: "wan2.6-r2v", Name: "Wan 2.6 R2V", Provider: "dashscope", Capability: "video", Mode: "r2v"},
			{ID: "wan2.6-i2v-flash", Name: "Wan 2.6 I2V Flash", Provider: "dashscope", Capability: "video", Mode: "i2v"},
			{ID: "happyhorse-1.0-i2v", Name: "HappyHorse 1.0 I2V", Provider: "dashscope", Capability: "video", Mode: "i2v"},
			{ID: "happyhorse-1.0-r2v", Name: "HappyHorse 1.0 R2V", Provider: "dashscope", Capability: "video", Mode: "r2v"},
			{ID: "kling-v3-i2v", Name: "Kling V3 I2V", Provider: "dashscope", Capability: "video", Mode: "i2v"},
			{ID: "kling-v3-r2v", Name: "Kling V3 R2V", Provider: "dashscope", Capability: "video", Mode: "r2v"},
			{ID: "viduq3-pro-i2v", Name: "Vidu Q3 Pro I2V", Provider: "dashscope", Capability: "video", Mode: "i2v"},
			{ID: "viduq3-pro-r2v", Name: "Vidu Q3 Pro R2V", Provider: "dashscope", Capability: "video", Mode: "r2v"},
			{ID: "pixverse-c1-i2v", Name: "PixVerse C1 I2V", Provider: "dashscope", Capability: "video", Mode: "i2v"},
			{ID: "pixverse-c1-r2v", Name: "PixVerse C1 R2V", Provider: "dashscope", Capability: "video", Mode: "r2v"},
		},
	}
}
