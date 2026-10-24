package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestExternalTaskProviderGeneratesDashScopeImage(t *testing.T) {
	t.Setenv("AI_PROVIDER", "dashscope")
	t.Setenv("DASHSCOPE_API_KEY", "test-key")
	t.Setenv("DASHSCOPE_POLL_INTERVAL_MS", "1")

	var createPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization header = %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/api/v1/services/aigc/image-generation/generation":
			if r.Header.Get("X-DashScope-Async") != "enable" {
				t.Fatalf("async header = %q", r.Header.Get("X-DashScope-Async"))
			}
			if err := json.NewDecoder(r.Body).Decode(&createPayload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"output": map[string]any{"task_id": "image-task"}})
		case "/api/v1/tasks/image-task":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"output": map[string]any{
					"task_status": "SUCCEEDED",
					"choices": []map[string]any{
						{"message": map[string]any{"content": []map[string]any{{"image": "https://cdn.example.com/image.png"}}}},
					},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("DASHSCOPE_BASE_URL", server.URL)

	provider := NewExternalTaskProvider("", "", "", "", 10*time.Second)
	result, err := provider.GenerateAsset(context.Background(), map[string]any{
		"prompt": "female detective in neon rain",
		"model":  "wan2.7-image-pro",
		"parameters": map[string]any{
			"aspect_ratio": "9:16",
		},
	})
	if err != nil {
		t.Fatalf("GenerateAsset returned error: %v", err)
	}
	if result["image_url"] != "https://cdn.example.com/image.png" {
		t.Fatalf("image_url = %#v", result["image_url"])
	}
	if createPayload["model"] != "wan2.7-image-pro" {
		t.Fatalf("model = %#v", createPayload["model"])
	}
	params := createPayload["parameters"].(map[string]any)
	if params["size"] != "720*1280" {
		t.Fatalf("size = %#v, want 720*1280", params["size"])
	}
}

func TestExternalTaskProviderGeneratesDashScopeI2V(t *testing.T) {
	t.Setenv("AI_PROVIDER", "dashscope")
	t.Setenv("DASHSCOPE_API_KEY", "test-key")
	t.Setenv("DASHSCOPE_POLL_INTERVAL_MS", "1")

	var createPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/services/aigc/video-generation/video-synthesis":
			if err := json.NewDecoder(r.Body).Decode(&createPayload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"output": map[string]any{"task_id": "video-task"}})
		case "/api/v1/tasks/video-task":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"output": map[string]any{
					"task_status": "SUCCEEDED",
					"video_url":   "https://cdn.example.com/shot.mp4",
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("DASHSCOPE_BASE_URL", server.URL)

	provider := NewExternalTaskProvider("", "", "", "", 10*time.Second)
	result, err := provider.GenerateVideo(context.Background(), map[string]any{
		"prompt":          "slow dolly in, detective turns",
		"model":           "wan2.7-i2v",
		"image_url":       "https://cdn.example.com/shot.png",
		"duration":        float64(5),
		"generate_audio":  true,
		"generation_mode": "i2v",
		"parameters": map[string]any{
			"ratio": "9:16",
		},
	})
	if err != nil {
		t.Fatalf("GenerateVideo returned error: %v", err)
	}
	if result["video_url"] != "https://cdn.example.com/shot.mp4" {
		t.Fatalf("video_url = %#v", result["video_url"])
	}
	input := createPayload["input"].(map[string]any)
	media := input["media"].([]any)
	firstMedia := media[0].(map[string]any)
	if firstMedia["type"] != "first_frame" || firstMedia["url"] != "https://cdn.example.com/shot.png" {
		t.Fatalf("media = %#v", input["media"])
	}
	params := createPayload["parameters"].(map[string]any)
	if params["ratio"] != "9:16" {
		t.Fatalf("ratio = %#v", params["ratio"])
	}
	if params["audio"] != true {
		t.Fatalf("audio = %#v", params["audio"])
	}
}
