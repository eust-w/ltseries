package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"ltseries/backend/internal/config"
	"ltseries/backend/internal/db"
	"ltseries/backend/internal/httpapi"
	"ltseries/backend/internal/queue"
	"ltseries/backend/internal/services"
)

func main() {
	_ = godotenv.Load(".env")
	_ = godotenv.Overload("../.env")
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	client, err := db.Connect(ctx, cfg.MongoURI)
	if err != nil {
		log.Fatalf("connect MongoDB: %v", err)
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	database := client.Database(cfg.MongoDB)
	taskQueue, err := queue.NewMongoQueue(ctx, database.Collection("tasks"))
	if err != nil {
		log.Fatalf("initialize queue: %v", err)
	}
	projectService, err := services.NewProjectService(ctx, database.Collection("projects"), taskQueue)
	if err != nil {
		log.Fatalf("initialize projects: %v", err)
	}
	seriesService, err := services.NewSeriesService(ctx, database.Collection("series"), projectService, taskQueue)
	if err != nil {
		log.Fatalf("initialize series: %v", err)
	}

	analyzer := services.NewLLMAnalyzer(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel)
	provider := services.NewExternalTaskProvider(cfg.AssetEndpoint, cfg.VideoEndpoint, cfg.AudioEndpoint, cfg.ProviderAPIKey, cfg.ProviderTimeout)
	mediaService := services.NewMediaService(cfg.StorageDir, cfg.PublicBaseURL, cfg.FFmpegPath)
	configService := services.NewRuntimeConfigService(cfg.ConfigPath)
	authService := services.NewAuthServiceFromEnv()
	worker := services.NewWorker(taskQueue, projectService, analyzer, provider, mediaService, cfg.WorkerPoll)
	worker.Start(ctx, cfg.WorkerCount)

	router := httpapi.NewRouter(httpapi.Dependencies{
		ProjectService: projectService,
		SeriesService:  seriesService,
		Queue:          taskQueue,
		MediaService:   mediaService,
		ConfigService:  configService,
		AuthService:    authService,
		Analyzer:       analyzer,
		AllowedOrigin:  cfg.AllowedOrigin,
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: cfg.ServerReadTimeout,
	}

	go func() {
		log.Printf("ltseries API listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
