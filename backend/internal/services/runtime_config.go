package services

import (
	"os"
	"strings"
	"sync"
)

type RuntimeConfigService struct {
	path string
	mu   sync.Mutex
}

type RuntimeConfig struct {
	AIProvider              string `json:"ai_provider"`
	DashScopeBaseURL        string `json:"dashscope_base_url"`
	DashScopeAPIKey         string `json:"dashscope_api_key,omitempty"`
	DashScopeAPIKeySet      bool   `json:"dashscope_api_key_set"`
	LLMBaseURL              string `json:"llm_base_url"`
	LLMModel                string `json:"llm_model"`
	LLMAPIKeySet            bool   `json:"llm_api_key_set"`
	DefaultImageModel       string `json:"default_image_model"`
	DefaultVideoModel       string `json:"default_video_model"`
	AssetEndpointConfigured bool   `json:"asset_endpoint_configured"`
	VideoEndpointConfigured bool   `json:"video_endpoint_configured"`
	AudioEndpointConfigured bool   `json:"audio_endpoint_configured"`
}

func NewRuntimeConfigService(path string) *RuntimeConfigService {
	if strings.TrimSpace(path) == "" {
		path = "../.env"
	}
	return &RuntimeConfigService{path: path}
}

func (s *RuntimeConfigService) Get() RuntimeConfig {
	return RuntimeConfig{
		AIProvider:              envString("AI_PROVIDER", "dashscope"),
		DashScopeBaseURL:        envString("DASHSCOPE_BASE_URL", "https://dashscope.aliyuncs.com"),
		DashScopeAPIKeySet:      strings.TrimSpace(os.Getenv("DASHSCOPE_API_KEY")) != "",
		LLMBaseURL:              envString("LLM_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1"),
		LLMModel:                envString("LLM_MODEL", "qwen-plus"),
		LLMAPIKeySet:            strings.TrimSpace(os.Getenv("LLM_API_KEY")) != "",
		DefaultImageModel:       envString("DEFAULT_IMAGE_MODEL", "wan2.7-image-pro"),
		DefaultVideoModel:       envString("DEFAULT_VIDEO_MODEL", "wan2.7-i2v"),
		AssetEndpointConfigured: strings.TrimSpace(os.Getenv("ASSET_PROVIDER_ENDPOINT")) != "",
		VideoEndpointConfigured: strings.TrimSpace(os.Getenv("VIDEO_PROVIDER_ENDPOINT")) != "",
		AudioEndpointConfigured: strings.TrimSpace(os.Getenv("AUDIO_PROVIDER_ENDPOINT")) != "",
	}
}

func (s *RuntimeConfigService) Save(input RuntimeConfig) (RuntimeConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	updates := map[string]string{}
	setIfNotBlank(updates, "AI_PROVIDER", input.AIProvider)
	setIfNotBlank(updates, "DASHSCOPE_BASE_URL", input.DashScopeBaseURL)
	setIfNotBlank(updates, "LLM_BASE_URL", input.LLMBaseURL)
	setIfNotBlank(updates, "LLM_MODEL", input.LLMModel)
	setIfNotBlank(updates, "DEFAULT_IMAGE_MODEL", input.DefaultImageModel)
	setIfNotBlank(updates, "DEFAULT_VIDEO_MODEL", input.DefaultVideoModel)
	if strings.TrimSpace(input.DashScopeAPIKey) != "" {
		updates["DASHSCOPE_API_KEY"] = strings.TrimSpace(input.DashScopeAPIKey)
		updates["LLM_API_KEY"] = strings.TrimSpace(input.DashScopeAPIKey)
	}
	for key, value := range updates {
		_ = os.Setenv(key, value)
	}
	if len(updates) > 0 {
		if err := mergeDotEnv(s.path, updates); err != nil {
			return RuntimeConfig{}, err
		}
	}
	return s.Get(), nil
}

func setIfNotBlank(updates map[string]string, key string, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		updates[key] = value
	}
}

func mergeDotEnv(path string, updates map[string]string) error {
	data, _ := os.ReadFile(path)
	lines := strings.Split(string(data), "\n")
	seen := map[string]bool{}
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || !strings.Contains(trimmed, "=") {
			continue
		}
		key := strings.TrimSpace(strings.SplitN(trimmed, "=", 2)[0])
		if value, ok := updates[key]; ok {
			lines[i] = key + "=" + value
			seen[key] = true
		}
	}
	for key, value := range updates {
		if !seen[key] {
			lines = append(lines, key+"="+value)
		}
	}
	content := strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0o600)
}
