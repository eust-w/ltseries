package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port              string
	MongoURI          string
	MongoDB           string
	WorkerCount       int
	WorkerPoll        time.Duration
	AIProvider        string
	ConfigPath        string
	DashScopeAPIKey   string
	DashScopeBaseURL  string
	LLMBaseURL        string
	LLMAPIKey         string
	LLMModel          string
	DefaultImageModel string
	DefaultVideoModel string
	AssetEndpoint     string
	VideoEndpoint     string
	AudioEndpoint     string
	ProviderAPIKey    string
	ProviderTimeout   time.Duration
	StorageDir        string
	PublicBaseURL     string
	FFmpegPath        string
	AllowedOrigin     string
	ServerReadTimeout time.Duration
}

func Load() Config {
	return Config{
		Port:              env("PORT", "18080"),
		MongoURI:          env("MONGODB_URI", "mongodb://127.0.0.1:27019"),
		MongoDB:           env("MONGODB_DB", "ltseries"),
		WorkerCount:       envInt("TASK_WORKERS", 2),
		WorkerPoll:        time.Duration(envInt("TASK_WORKER_POLL_MS", 1000)) * time.Millisecond,
		AIProvider:        env("AI_PROVIDER", "dashscope"),
		ConfigPath:        env("CONFIG_PATH", "../.env"),
		DashScopeAPIKey:   env("DASHSCOPE_API_KEY", os.Getenv("LLM_API_KEY")),
		DashScopeBaseURL:  env("DASHSCOPE_BASE_URL", "https://dashscope.aliyuncs.com"),
		LLMBaseURL:        env("LLM_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1"),
		LLMAPIKey:         env("LLM_API_KEY", os.Getenv("DASHSCOPE_API_KEY")),
		LLMModel:          env("LLM_MODEL", "qwen-plus"),
		DefaultImageModel: env("DEFAULT_IMAGE_MODEL", "wan2.7-image-pro"),
		DefaultVideoModel: env("DEFAULT_VIDEO_MODEL", "wan2.7-i2v"),
		AssetEndpoint:     os.Getenv("ASSET_PROVIDER_ENDPOINT"),
		VideoEndpoint:     os.Getenv("VIDEO_PROVIDER_ENDPOINT"),
		AudioEndpoint:     os.Getenv("AUDIO_PROVIDER_ENDPOINT"),
		ProviderAPIKey:    os.Getenv("PROVIDER_API_KEY"),
		ProviderTimeout:   time.Duration(envInt("PROVIDER_TIMEOUT_SECONDS", 600)) * time.Second,
		StorageDir:        env("STORAGE_DIR", "../storage"),
		PublicBaseURL:     env("PUBLIC_BASE_URL", "http://localhost:18080"),
		FFmpegPath:        env("FFMPEG_PATH", "ffmpeg"),
		AllowedOrigin:     env("CORS_ALLOWED_ORIGIN", "*"),
		ServerReadTimeout: 30 * time.Second,
	}
}

func env(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
