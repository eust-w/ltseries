package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"ltseries/backend/internal/models"
	"ltseries/backend/internal/queue"
	"ltseries/backend/internal/services"
)

type Dependencies struct {
	ProjectService *services.ProjectService
	SeriesService  *services.SeriesService
	Queue          *queue.MongoQueue
	MediaService   *services.MediaService
	ConfigService  *services.RuntimeConfigService
	AuthService    *services.AuthService
	Analyzer       *services.LLMAnalyzer
	AllowedOrigin  string
}

type createProjectRequest struct {
	Title    string `json:"title"`
	Script   string `json:"script"`
	Workflow string `json:"workflow"`
}

type createTaskRequest struct {
	Name       string         `json:"name"`
	AssetType  string         `json:"asset_type"`
	Prompt     string         `json:"prompt"`
	Model      string         `json:"model"`
	Inputs     map[string]any `json:"inputs"`
	Parameters map[string]any `json:"parameters"`
}

type entityRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Attributes  map[string]any `json:"attributes"`
}

type updateEntityRequest struct {
	Attributes map[string]any `json:"attributes"`
}

type variantRequest struct {
	AssetID        string `json:"asset_id"`
	AssetType      string `json:"asset_type"`
	VariantID      string `json:"variant_id"`
	GenerationType string `json:"generation_type"`
	UploadType     string `json:"upload_type"`
	IsFavorited    bool   `json:"is_favorited"`
}

type batchAssetsRequest struct {
	Model      string         `json:"model"`
	Force      bool           `json:"force"`
	Parameters map[string]any `json:"parameters"`
}

type assetVideoRequest struct {
	Prompt         string         `json:"prompt"`
	UploadType     string         `json:"upload_type"`
	Model          string         `json:"model"`
	Duration       int            `json:"duration"`
	GenerationMode string         `json:"generation_mode"`
	Inputs         map[string]any `json:"inputs"`
	Parameters     map[string]any `json:"parameters"`
}

type frameRequest struct {
	Action      string         `json:"action"`
	Camera      string         `json:"camera"`
	SceneID     string         `json:"scene_id"`
	Updates     map[string]any `json:"updates"`
	FrameIDs    []string       `json:"frame_ids"`
	Prompt      string         `json:"prompt"`
	References  []string       `json:"references"`
	FrameID     string         `json:"frame_id"`
	VideoTaskID string         `json:"video_task_id"`
	AudioType   string         `json:"audio_type"`
}

type storyboardPipelineRequest struct {
	Model            string         `json:"model"`
	Duration         int            `json:"duration"`
	RegenerateImages bool           `json:"regenerate_images"`
	RegenerateVideos bool           `json:"regenerate_videos"`
	Parameters       map[string]any `json:"parameters"`
}

type artDirectionRequest struct {
	SelectedStyleID   string           `json:"selected_style_id"`
	StyleConfig       map[string]any   `json:"style_config"`
	CustomStyles      []map[string]any `json:"custom_styles"`
	AIRecommendations []map[string]any `json:"ai_recommendations"`
}

type seriesRequest struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Workflow    string         `json:"workflow"`
	Updates     map[string]any `json:"updates"`
	ProjectID   string         `json:"project_id"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewRouter(deps Dependencies) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware(deps.AllowedOrigin))
	if deps.MediaService != nil {
		router.Static("/files", deps.MediaService.StorageDir())
	}

	api := router.Group("/api")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "ltseries"})
	})

	api.GET("/auth/status", func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		if deps.AuthService == nil {
			c.JSON(http.StatusOK, services.AuthStatus{Enabled: false, Authenticated: true})
			return
		}
		c.JSON(http.StatusOK, deps.AuthService.Status(token))
	})

	api.POST("/auth/login", func(c *gin.Context) {
		if deps.AuthService == nil || !deps.AuthService.Enabled() {
			c.JSON(http.StatusOK, gin.H{"token": "", "expires_at": 0, "auth_disabled": true})
			return
		}
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		token, expiresAt, err := deps.AuthService.Login(req.Username, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token, "expires_at": expiresAt})
	})

	api.Use(authMiddleware(deps.AuthService))

	api.GET("/system/check", func(c *gin.Context) {
		providerRouting := "environment"
		if deps.ConfigService != nil {
			providerRouting = deps.ConfigService.Get().AIProvider
		}
		c.JSON(http.StatusOK, gin.H{
			"service":          "ltseries",
			"mongo":            deps.Queue != nil,
			"storage_dir":      storageDir(deps),
			"provider_routing": providerRouting,
		})
	})

	api.GET("/models", func(c *gin.Context) {
		c.JSON(http.StatusOK, services.DefaultModelCatalog())
	})

	api.GET("/config", func(c *gin.Context) {
		if deps.ConfigService == nil {
			c.JSON(http.StatusOK, services.NewRuntimeConfigService("").Get())
			return
		}
		c.JSON(http.StatusOK, deps.ConfigService.Get())
	})

	api.PUT("/config", func(c *gin.Context) {
		if deps.ConfigService == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "config service is not configured"})
			return
		}
		var req services.RuntimeConfig
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		next, err := deps.ConfigService.Save(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, next)
	})

	api.POST("/upload", func(c *gin.Context) {
		if deps.MediaService == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "media service is not configured"})
			return
		}
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		opened, err := file.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		defer opened.Close()
		url, err := deps.MediaService.SaveUploadedFile(opened, file.Filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"url": url})
	})

	api.POST("/projects", func(c *gin.Context) {
		var req createProjectRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		project, err := deps.ProjectService.Create(c.Request.Context(), req.Title, req.Script, req.Workflow)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, project)
	})

	api.GET("/projects", func(c *gin.Context) {
		projects, err := deps.ProjectService.List(c.Request.Context(), parseLimit(c.Query("limit"), 50))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, projects)
	})

	api.GET("/projects/:id", func(c *gin.Context) {
		project, err := deps.ProjectService.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, services.ErrNotFound) {
				status = http.StatusNotFound
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.GET("/projects/:id/production-audit", func(c *gin.Context) {
		report, err := deps.ProjectService.ProductionAudit(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, report)
	})

	api.PUT("/projects/:id", func(c *gin.Context) {
		var updates map[string]any
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		project, err := deps.ProjectService.Update(c.Request.Context(), c.Param("id"), updates)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.DELETE("/projects/:id", func(c *gin.Context) {
		projectID := c.Param("id")
		if err := deps.ProjectService.Delete(c.Request.Context(), projectID); err != nil {
			writeServiceError(c, err)
			return
		}
		if deps.SeriesService != nil {
			if err := deps.SeriesService.RemoveProjectReferences(c.Request.Context(), projectID); err != nil {
				writeServiceError(c, err)
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted", "id": projectID})
	})

	api.POST("/projects/:id/analyze", func(c *gin.Context) {
		task, err := deps.ProjectService.EnqueueAnalysis(c.Request.Context(), c.Param("id"))
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, services.ErrNotFound) {
				status = http.StatusNotFound
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "task": task})
	})

	api.POST("/projects/:id/reparse", func(c *gin.Context) {
		task, err := deps.ProjectService.EnqueueAnalysis(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "task": task})
	})

	api.POST("/projects/:id/art-direction/analyze", func(c *gin.Context) {
		task, err := deps.ProjectService.EnqueueArtDirection(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "task": task})
	})

	api.POST("/projects/:id/art-direction", func(c *gin.Context) {
		var req artDirectionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		project, err := deps.ProjectService.SaveArtDirection(c.Request.Context(), c.Param("id"), models.ArtDirection{
			SelectedStyleID:   req.SelectedStyleID,
			StyleConfig:       req.StyleConfig,
			CustomStyles:      req.CustomStyles,
			AIRecommendations: req.AIRecommendations,
		})
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.GET("/art-direction/presets", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"presets": []map[string]any{
			{"id": "cinematic", "name": "Cinematic", "positive_prompt": "cinematic lighting, detailed, high production value"},
			{"id": "anime", "name": "Anime", "positive_prompt": "anime style, expressive characters, clean line art"},
			{"id": "realistic", "name": "Realistic", "positive_prompt": "photorealistic, natural light, high detail"},
		}})
	})

	api.POST("/projects/:id/assets", func(c *gin.Context) {
		var req createTaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		task, err := deps.ProjectService.EnqueueAsset(c.Request.Context(), c.Param("id"), taskPayload(req))
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, services.ErrNotFound) {
				status = http.StatusNotFound
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "task": task})
	})

	api.GET("/projects/:id/assets", func(c *gin.Context) {
		project, err := deps.ProjectService.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"characters": project.Characters, "scenes": project.Scenes, "props": project.Props})
	})

	api.POST("/projects/:id/generate_assets", func(c *gin.Context) {
		var req batchAssetsRequest
		_ = c.ShouldBindJSON(&req)
		tasks, err := deps.ProjectService.EnqueueAllAssetGenerations(c.Request.Context(), c.Param("id"), req.Model, req.Parameters, req.Force)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		taskIDs := make([]string, 0, len(tasks))
		for _, task := range tasks {
			taskIDs = append(taskIDs, task.ID)
		}
		c.JSON(http.StatusAccepted, gin.H{"task_ids": taskIDs, "tasks": tasks})
	})

	api.POST("/projects/:id/assets/:type", func(c *gin.Context) {
		var req entityRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		project, err := deps.ProjectService.AddEntity(c.Request.Context(), c.Param("id"), c.Param("type"), req.Name, req.Description)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, project)
	})

	api.DELETE("/projects/:id/assets/:type/:asset_id", func(c *gin.Context) {
		project, err := deps.ProjectService.DeleteEntity(c.Request.Context(), c.Param("id"), c.Param("type"), c.Param("asset_id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.PUT("/projects/:id/assets/:type/:asset_id", func(c *gin.Context) {
		var updates map[string]any
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		project, err := deps.ProjectService.UpdateEntity(c.Request.Context(), c.Param("id"), c.Param("type"), c.Param("asset_id"), updates)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.POST("/projects/:id/assets/:type/:asset_id/toggle-lock", func(c *gin.Context) {
		project, err := deps.ProjectService.ToggleEntityLock(c.Request.Context(), c.Param("id"), c.Param("type"), c.Param("asset_id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.POST("/projects/:id/assets/:type/:asset_id/upload", func(c *gin.Context) {
		if deps.MediaService == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "media service is not configured"})
			return
		}
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		opened, err := file.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		defer opened.Close()
		url, err := deps.MediaService.SaveUploadedFile(opened, file.Filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		project, err := deps.ProjectService.AddUploadedVariant(c.Request.Context(), c.Param("id"), c.Param("type"), c.Param("asset_id"), c.Query("upload_type"), url, c.PostForm("prompt"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"url": url, "project": project})
	})

	api.POST("/projects/:id/assets/:type/:asset_id/generate", func(c *gin.Context) {
		var req createTaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		task, err := deps.ProjectService.EnqueueAssetGeneration(c.Request.Context(), c.Param("id"), c.Param("asset_id"), c.Param("type"), req.AssetType, req.Prompt, req.Model, req.Inputs, req.Parameters)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "task": task})
	})

	api.POST("/projects/:id/assets/:type/:asset_id/generate_video", func(c *gin.Context) {
		var req assetVideoRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		payload := map[string]any{
			"prompt":          req.Prompt,
			"model":           req.Model,
			"duration":        req.Duration,
			"generation_mode": req.GenerationMode,
			"inputs":          req.Inputs,
			"parameters":      req.Parameters,
		}
		task, err := deps.ProjectService.EnqueueAssetVideo(c.Request.Context(), c.Param("id"), c.Param("type"), c.Param("asset_id"), req.UploadType, payload)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "task": task})
	})

	api.DELETE("/projects/:id/assets/:type/:asset_id/videos/:video_id", func(c *gin.Context) {
		project, err := deps.ProjectService.DeleteAssetVideo(c.Request.Context(), c.Param("id"), c.Param("type"), c.Param("asset_id"), c.Query("upload_type"), c.Param("video_id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.POST("/projects/:id/assets/:type/:asset_id/variant/select", func(c *gin.Context) {
		var req struct {
			UploadType string `json:"upload_type"`
			VariantID  string `json:"variant_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		project, err := deps.ProjectService.SelectVariant(c.Request.Context(), c.Param("id"), c.Param("type"), c.Param("asset_id"), req.UploadType, req.VariantID)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.POST("/projects/:id/assets/variant/favorite", func(c *gin.Context) {
		var req variantRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		uploadType := req.UploadType
		if uploadType == "" {
			uploadType = req.GenerationType
		}
		project, err := deps.ProjectService.FavoriteVariant(c.Request.Context(), c.Param("id"), req.AssetType, req.AssetID, uploadType, req.VariantID, req.IsFavorited)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.POST("/projects/:id/assets/variant/delete", func(c *gin.Context) {
		var req variantRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		uploadType := req.UploadType
		if uploadType == "" {
			uploadType = req.GenerationType
		}
		project, err := deps.ProjectService.DeleteVariant(c.Request.Context(), c.Param("id"), req.AssetType, req.AssetID, uploadType, req.VariantID)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.POST("/projects/:id/videos", func(c *gin.Context) {
		var req createTaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		task, err := deps.ProjectService.EnqueueVideo(c.Request.Context(), c.Param("id"), taskPayload(req))
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, services.ErrNotFound) {
				status = http.StatusNotFound
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "task": task})
	})

	api.POST("/projects/:id/video-tasks", func(c *gin.Context) {
		var payload map[string]any
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		task, err := deps.ProjectService.EnqueueVideoTask(c.Request.Context(), c.Param("id"), payload)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "task": task})
	})

	api.POST("/projects/:id/storyboard/analyze", func(c *gin.Context) {
		var req frameRequest
		_ = c.ShouldBindJSON(&req)
		task, err := deps.ProjectService.EnqueueStoryboardAnalysis(c.Request.Context(), c.Param("id"), req.Action)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "task": task})
	})

	api.POST("/projects/:id/storyboard/refine_prompt", func(c *gin.Context) {
		var payload map[string]any
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		prompt := stringFromPayload(payload, "prompt", "text")
		refined, err := analyzerFor(deps).CompleteText(c.Request.Context(), storyboardPromptSystem, prompt)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"prompt": refined})
	})

	api.POST("/projects/:id/storyboard/render", func(c *gin.Context) {
		var req frameRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		task, err := deps.ProjectService.EnqueueStoryboardRender(c.Request.Context(), c.Param("id"), req.FrameID, req.Prompt, req.References)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "task": task})
	})

	api.POST("/projects/:id/storyboard/complete", func(c *gin.Context) {
		var req storyboardPipelineRequest
		_ = c.ShouldBindJSON(&req)
		task, err := deps.ProjectService.EnqueueStoryboardPipeline(c.Request.Context(), c.Param("id"), map[string]any{
			"model":             req.Model,
			"duration":          req.Duration,
			"regenerate_images": req.RegenerateImages,
			"regenerate_videos": req.RegenerateVideos,
			"parameters":        req.Parameters,
		})
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "task": task})
	})

	api.POST("/projects/:id/frames", func(c *gin.Context) {
		var req frameRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		project, err := deps.ProjectService.AddFrame(c.Request.Context(), c.Param("id"), req.Action, req.Camera, req.SceneID)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, project)
	})

	api.POST("/projects/:id/frames/toggle_lock", func(c *gin.Context) {
		var req frameRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		project, err := deps.ProjectService.ToggleFrameLock(c.Request.Context(), c.Param("id"), req.FrameID)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.PUT("/projects/:id/frames/:frame_id", func(c *gin.Context) {
		var updates map[string]any
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		project, err := deps.ProjectService.UpdateFrame(c.Request.Context(), c.Param("id"), c.Param("frame_id"), updates)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.POST("/projects/:id/frames/:frame_id/toggle-lock", func(c *gin.Context) {
		project, err := deps.ProjectService.ToggleFrameLock(c.Request.Context(), c.Param("id"), c.Param("frame_id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.DELETE("/projects/:id/frames/:frame_id", func(c *gin.Context) {
		project, err := deps.ProjectService.DeleteFrame(c.Request.Context(), c.Param("id"), c.Param("frame_id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.POST("/projects/:id/frames/:frame_id/upload_image", func(c *gin.Context) {
		if deps.MediaService == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "media service is not configured"})
			return
		}
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		opened, err := file.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		defer opened.Close()
		url, err := deps.MediaService.SaveUploadedFile(opened, file.Filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		project, err := deps.ProjectService.AddFrameImageVariant(c.Request.Context(), c.Param("id"), c.Param("frame_id"), url, c.PostForm("prompt"), true)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"url": url, "project": project})
	})

	api.POST("/projects/:id/frames/:frame_id/extract_last_frame", func(c *gin.Context) {
		if deps.MediaService == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "media service is not configured"})
			return
		}
		var req frameRequest
		_ = c.ShouldBindJSON(&req)
		videoURL, err := deps.ProjectService.FrameVideoURL(c.Request.Context(), c.Param("id"), c.Param("frame_id"), req.VideoTaskID)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		url, err := deps.MediaService.ExtractLastFrame(c.Request.Context(), videoURL, c.Param("id"), c.Param("frame_id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		project, err := deps.ProjectService.AddFrameImageVariant(c.Request.Context(), c.Param("id"), c.Param("frame_id"), url, "视频末帧", false)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"url": url, "project": project})
	})

	api.POST("/projects/:id/frames/:frame_id/copy", func(c *gin.Context) {
		project, err := deps.ProjectService.CopyFrame(c.Request.Context(), c.Param("id"), c.Param("frame_id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.PUT("/projects/:id/frames/reorder", func(c *gin.Context) {
		var req frameRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		project, err := deps.ProjectService.ReorderFrames(c.Request.Context(), c.Param("id"), req.FrameIDs)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.POST("/projects/:id/frames/:frame_id/select-video", func(c *gin.Context) {
		var req frameRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		project, err := deps.ProjectService.SelectFrameVideo(c.Request.Context(), c.Param("id"), c.Param("frame_id"), req.VideoTaskID)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.POST("/projects/:id/frames/:frame_id/audio", func(c *gin.Context) {
		var payload map[string]any
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		audioType := "dialogue"
		if raw, ok := payload["audio_type"].(string); ok {
			audioType = raw
		}
		task, err := deps.ProjectService.EnqueueAudio(c.Request.Context(), c.Param("id"), c.Param("frame_id"), audioType, payload)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "task": task})
	})

	api.POST("/projects/:id/mix/generate_sfx", func(c *gin.Context) {
		var payload map[string]any
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		frameID := stringFromPayload(payload, "frame_id")
		task, err := deps.ProjectService.EnqueueAudio(c.Request.Context(), c.Param("id"), frameID, "sfx", payload)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "task": task})
	})

	api.POST("/projects/:id/mix/generate_bgm", func(c *gin.Context) {
		var payload map[string]any
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		frameID := stringFromPayload(payload, "frame_id")
		task, err := deps.ProjectService.EnqueueAudio(c.Request.Context(), c.Param("id"), frameID, "bgm", payload)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "task": task})
	})

	api.POST("/projects/:id/merge", func(c *gin.Context) {
		task, err := deps.ProjectService.EnqueueMerge(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "task": task})
	})

	api.POST("/projects/:id/export", func(c *gin.Context) {
		project, err := deps.ProjectService.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		if project.MergedVideoURL != "" {
			c.JSON(http.StatusOK, gin.H{"url": project.MergedVideoURL})
			return
		}
		task, err := deps.ProjectService.EnqueueMerge(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"task_id": task.ID, "task": task})
	})

	api.PUT("/projects/:id/model-settings", func(c *gin.Context) {
		var settings models.ModelSettings
		if err := c.ShouldBindJSON(&settings); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		project, err := deps.ProjectService.UpdateModelSettings(c.Request.Context(), c.Param("id"), settings)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.PUT("/projects/:id/prompt-config", func(c *gin.Context) {
		var promptConfig models.PromptConfig
		if err := c.ShouldBindJSON(&promptConfig); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		project, err := deps.ProjectService.UpdatePromptConfig(c.Request.Context(), c.Param("id"), promptConfig)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.GET("/voices", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"voices": []map[string]any{}, "provider_configured": false})
	})

	api.POST("/projects/:id/characters/:char_id/voice", func(c *gin.Context) {
		var payload map[string]any
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updates := map[string]any{}
		if value := stringFromPayload(payload, "voice_id"); value != "" {
			updates["voice_id"] = value
		}
		if value := stringFromPayload(payload, "voice_name", "name"); value != "" {
			updates["voice_name"] = value
		}
		project, err := deps.ProjectService.UpdateEntity(c.Request.Context(), c.Param("id"), "character", c.Param("char_id"), updates)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.PUT("/projects/:id/characters/:char_id/voice_params", func(c *gin.Context) {
		var payload map[string]any
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updates := map[string]any{}
		for _, key := range []string{"voice_speed", "voice_pitch", "voice_volume"} {
			if value, ok := payload[key]; ok {
				updates[key] = value
			}
		}
		project, err := deps.ProjectService.UpdateEntity(c.Request.Context(), c.Param("id"), "character", c.Param("char_id"), updates)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, project)
	})

	api.POST("/video/polish_prompt", func(c *gin.Context) {
		var payload map[string]any
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		prompt := stringFromPayload(payload, "prompt", "text")
		refined, err := analyzerFor(deps).CompleteText(c.Request.Context(), videoPromptSystem, prompt)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"prompt": refined})
	})

	api.POST("/video/polish_r2v_prompt", func(c *gin.Context) {
		var payload map[string]any
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		prompt := stringFromPayload(payload, "prompt", "text")
		refined, err := analyzerFor(deps).CompleteText(c.Request.Context(), r2vPromptSystem, prompt)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"prompt": refined})
	})

	api.GET("/tasks", func(c *gin.Context) {
		tasks, err := deps.Queue.List(c.Request.Context(), queue.TaskFilter{
			ProjectID: c.Query("project_id"),
			Status:    models.TaskStatus(c.Query("status")),
			Type:      models.TaskType(c.Query("type")),
			Limit:     parseLimit(c.Query("limit"), 100),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, tasks)
	})

	api.GET("/tasks/:id", func(c *gin.Context) {
		task, err := deps.Queue.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if task == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusOK, task)
	})

	api.DELETE("/tasks/:id", func(c *gin.Context) {
		deleted, err := deps.Queue.Delete(c.Request.Context(), c.Param("id"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !deleted {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted", "id": c.Param("id")})
	})

	if deps.SeriesService != nil {
		api.POST("/series", func(c *gin.Context) {
			var req seriesRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			series, err := deps.SeriesService.Create(c.Request.Context(), req.Title, req.Description, req.Workflow)
			if err != nil {
				writeServiceError(c, err)
				return
			}
			c.JSON(http.StatusCreated, series)
		})
		api.GET("/series", func(c *gin.Context) {
			series, err := deps.SeriesService.List(c.Request.Context())
			if err != nil {
				writeServiceError(c, err)
				return
			}
			c.JSON(http.StatusOK, series)
		})
		api.GET("/series/:id", func(c *gin.Context) {
			series, err := deps.SeriesService.Get(c.Request.Context(), c.Param("id"))
			if err != nil {
				writeServiceError(c, err)
				return
			}
			c.JSON(http.StatusOK, series)
		})
		api.PUT("/series/:id", func(c *gin.Context) {
			var updates map[string]any
			if err := c.ShouldBindJSON(&updates); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			series, err := deps.SeriesService.Update(c.Request.Context(), c.Param("id"), updates)
			if err != nil {
				writeServiceError(c, err)
				return
			}
			c.JSON(http.StatusOK, series)
		})
		api.DELETE("/series/:id", func(c *gin.Context) {
			if err := deps.SeriesService.Delete(c.Request.Context(), c.Param("id")); err != nil {
				writeServiceError(c, err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "deleted"})
		})
		api.POST("/series/:id/episodes", func(c *gin.Context) {
			var req seriesRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			series, err := deps.SeriesService.AddEpisode(c.Request.Context(), c.Param("id"), req.ProjectID)
			if err != nil {
				writeServiceError(c, err)
				return
			}
			c.JSON(http.StatusOK, series)
		})
		api.DELETE("/series/:id/episodes/:project_id", func(c *gin.Context) {
			series, err := deps.SeriesService.RemoveEpisode(c.Request.Context(), c.Param("id"), c.Param("project_id"))
			if err != nil {
				writeServiceError(c, err)
				return
			}
			c.JSON(http.StatusOK, series)
		})
	}

	return router
}

func corsMiddleware(origin string) gin.HandlerFunc {
	if origin == "" {
		origin = "*"
	}
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin,Content-Type,Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func authMiddleware(auth *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if auth == nil || !auth.Enabled() {
			c.Next()
			return
		}
		if !auth.Verify(bearerToken(c.GetHeader("Authorization"))) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return header
}

func parseLimit(raw string, fallback int64) int64 {
	if raw == "" {
		return fallback
	}
	limit, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || limit <= 0 {
		return fallback
	}
	if limit > 1000 {
		return 1000
	}
	return limit
}

func taskPayload(req createTaskRequest) map[string]any {
	payload := map[string]any{
		"name":       req.Name,
		"asset_type": req.AssetType,
		"prompt":     req.Prompt,
		"model":      req.Model,
	}
	if req.Inputs != nil {
		payload["inputs"] = req.Inputs
	}
	if req.Parameters != nil {
		payload["parameters"] = req.Parameters
	}
	return payload
}

const storyboardPromptSystem = "你是短剧分镜导演。把用户输入润色成可直接用于生成分镜图的中文提示词，保留人物、场景、动作和镜头意图，补充构图、光线、景别和连续性要求。只返回润色后的提示词。"
const videoPromptSystem = "你是短剧视频生成提示词导演。把用户输入润色成 I2V/T2V 视频生成提示词，明确主体动作、镜头运动、节奏、光线、画面质量和负向约束。只返回润色后的提示词。"
const r2vPromptSystem = "你是参考视频生成导演。把用户输入润色成 R2V 生成提示词，强调参考视频/参考图的一致性、动作变化、镜头运动和身份保持。只返回润色后的提示词。"

func analyzerFor(deps Dependencies) *services.LLMAnalyzer {
	if deps.Analyzer != nil {
		return deps.Analyzer
	}
	return services.NewLLMAnalyzer("", "", "")
}

func stringFromPayload(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func writeServiceError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, services.ErrNotFound) {
		status = http.StatusNotFound
	} else if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") {
		status = http.StatusBadRequest
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

func storageDir(deps Dependencies) string {
	if deps.MediaService == nil {
		return ""
	}
	return deps.MediaService.StorageDir()
}
