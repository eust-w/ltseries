package services

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"ltseries/backend/internal/models"
)

type MediaService struct {
	storageDir string
	publicBase string
	ffmpegPath string
	httpClient *http.Client
}

func NewMediaService(storageDir string, publicBase string, ffmpegPath string) *MediaService {
	return &MediaService{
		storageDir: storageDir,
		publicBase: strings.TrimRight(publicBase, "/"),
		ffmpegPath: ffmpegPath,
		httpClient: &http.Client{Timeout: 10 * time.Minute},
	}
}

func (m *MediaService) StorageDir() string {
	return m.storageDir
}

func (m *MediaService) SaveUploadedFile(reader io.Reader, filename string) (string, error) {
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".bin"
	}
	name := uuid.NewString() + ext
	dir := filepath.Join(m.storageDir, "uploads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := io.Copy(file, reader); err != nil {
		return "", err
	}
	return "/files/uploads/" + name, nil
}

func (m *MediaService) MergeProject(ctx context.Context, project *models.Project) (string, error) {
	if project == nil {
		return "", errors.New("project is required")
	}
	videoURLs := selectedFrameVideoURLs(project)
	if len(videoURLs) == 0 {
		return "", errors.New("no selected or completed frame videos to merge")
	}

	workDir := filepath.Join(m.storageDir, "merge", project.ID)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", err
	}

	listPath := filepath.Join(workDir, "concat.txt")
	listFile, err := os.Create(listPath)
	if err != nil {
		return "", err
	}
	defer listFile.Close()

	for _, url := range videoURLs {
		path, err := m.resolveVideo(ctx, url, workDir)
		if err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(listFile, "file '%s'\n", strings.ReplaceAll(path, "'", "'\\''")); err != nil {
			return "", err
		}
	}
	if err := listFile.Close(); err != nil {
		return "", err
	}

	outputName := fmt.Sprintf("merged_%s_%d.mp4", project.ID, time.Now().Unix())
	outputDir := filepath.Join(m.storageDir, "video")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	outputPath := filepath.Join(outputDir, outputName)
	cmd := exec.CommandContext(
		ctx,
		m.ffmpegPath,
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", listPath,
		"-c:v", "libx264",
		"-crf", "23",
		"-preset", "fast",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		outputPath,
	)
	combined, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg merge failed: %w: %s", err, string(combined))
	}
	return "/files/video/" + outputName, nil
}

func (m *MediaService) ExtractLastFrame(ctx context.Context, videoURL string, projectID string, frameID string) (string, error) {
	if strings.TrimSpace(videoURL) == "" {
		return "", errors.New("video url is required")
	}
	workDir := filepath.Join(m.storageDir, "extract", projectID)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", err
	}
	inputPath, err := m.resolveVideo(ctx, videoURL, workDir)
	if err != nil {
		return "", err
	}
	outputDir := filepath.Join(m.storageDir, "uploads")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	outputName := fmt.Sprintf("lastframe_%s_%s_%d.png", projectID, frameID, time.Now().UnixNano())
	outputPath := filepath.Join(outputDir, outputName)
	cmd := exec.CommandContext(ctx, m.ffmpegPath, "-y", "-sseof", "-0.1", "-i", inputPath, "-frames:v", "1", outputPath)
	combined, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg extract last frame failed: %w: %s", err, string(combined))
	}
	return "/files/uploads/" + outputName, nil
}

func (m *MediaService) resolveVideo(ctx context.Context, url string, workDir string) (string, error) {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		hash := sha1.Sum([]byte(url))
		target := filepath.Join(workDir, hex.EncodeToString(hash[:])+".mp4")
		if _, err := os.Stat(target); err == nil {
			return filepath.Abs(target)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		resp, err := m.httpClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("download video failed: status=%d", resp.StatusCode)
		}
		file, err := os.Create(target)
		if err != nil {
			return "", err
		}
		defer file.Close()
		if _, err := io.Copy(file, resp.Body); err != nil {
			return "", err
		}
		return filepath.Abs(target)
	}
	clean := strings.TrimPrefix(url, m.publicBase)
	clean = strings.TrimPrefix(clean, "/files/")
	path := filepath.Join(m.storageDir, clean)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("video file not found: %s", url)
	}
	return filepath.Abs(path)
}

func selectedFrameVideoURLs(project *models.Project) []string {
	byID := map[string]models.VideoTask{}
	for _, task := range project.VideoTasks {
		byID[task.ID] = task
	}
	urls := []string{}
	for _, frame := range project.Frames {
		if frame.SelectedVideoID != "" {
			if task, ok := byID[frame.SelectedVideoID]; ok && task.VideoURL != "" {
				urls = append(urls, task.VideoURL)
				continue
			}
		}
		if frame.VideoURL != "" {
			urls = append(urls, frame.VideoURL)
			continue
		}
		for _, task := range project.VideoTasks {
			if task.FrameID == frame.ID && task.Status == models.TaskStatusCompleted && task.VideoURL != "" {
				urls = append(urls, task.VideoURL)
				break
			}
		}
	}
	return urls
}
