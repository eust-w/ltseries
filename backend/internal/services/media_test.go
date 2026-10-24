package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveVideoReturnsAbsolutePathForConcat(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	storageDir := "storage"
	videoDir := filepath.Join(storageDir, "video")
	if err := os.MkdirAll(videoDir, 0o755); err != nil {
		t.Fatalf("create video dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(videoDir, "clip.mp4"), []byte("not-real-video"), 0o644); err != nil {
		t.Fatalf("write clip: %v", err)
	}

	service := NewMediaService(storageDir, "http://localhost:18080", "ffmpeg")
	path, err := service.resolveVideo(context.Background(), "/files/video/clip.mp4", t.TempDir())
	if err != nil {
		t.Fatalf("resolve video: %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("resolved path = %q, want absolute path for ffmpeg concat", path)
	}
}
