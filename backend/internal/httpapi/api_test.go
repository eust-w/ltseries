package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"ltseries/backend/internal/httpapi"
	"ltseries/backend/internal/queue"
	"ltseries/backend/internal/services"
)

func testMongoURI() string {
	if uri := os.Getenv("LTSERIES_TEST_MONGODB_URI"); uri != "" {
		return uri
	}
	return "mongodb://127.0.0.1:27019"
}

func newTestRouter(t *testing.T) (http.Handler, func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(options.Client().ApplyURI(testMongoURI()))
	if err != nil {
		cancel()
		t.Fatalf("connect mongo: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		cancel()
		_ = client.Disconnect(context.Background())
		t.Fatalf("ping mongo: %v", err)
	}

	dbName := "ltseries_api_test_" + uuid.NewString()
	database := client.Database(dbName)
	q, err := queue.NewMongoQueue(ctx, database.Collection("tasks"))
	if err != nil {
		cancel()
		_ = client.Database(dbName).Drop(context.Background())
		_ = client.Disconnect(context.Background())
		t.Fatalf("create queue: %v", err)
	}
	projectService, err := services.NewProjectService(ctx, database.Collection("projects"), q)
	if err != nil {
		cancel()
		_ = client.Database(dbName).Drop(context.Background())
		_ = client.Disconnect(context.Background())
		t.Fatalf("create project service: %v", err)
	}
	seriesService, err := services.NewSeriesService(ctx, database.Collection("series"), projectService, q)
	if err != nil {
		cancel()
		_ = client.Database(dbName).Drop(context.Background())
		_ = client.Disconnect(context.Background())
		t.Fatalf("create series service: %v", err)
	}

	router := httpapi.NewRouter(httpapi.Dependencies{
		ProjectService: projectService,
		SeriesService:  seriesService,
		Queue:          q,
		MediaService:   services.NewMediaService(t.TempDir(), "", "ffmpeg"),
	})

	cleanup := func() {
		_ = client.Database(dbName).Drop(context.Background())
		_ = client.Disconnect(context.Background())
		cancel()
	}
	return router, cleanup
}

func createTestProject(t *testing.T, router http.Handler) string {
	t.Helper()
	createBody := []byte(`{"title":"测试项目","script":"第一集：主角进入实验室。","workflow":"r2v"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create project status = %d body=%s", createRec.Code, createRec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created project: %v", err)
	}
	projectID, _ := created["id"].(string)
	if projectID == "" {
		t.Fatalf("created project missing id: %#v", created)
	}
	return projectID
}

func addTestAsset(t *testing.T, router http.Handler, projectID string, assetType string) string {
	t.Helper()
	body := []byte(`{"name":"测试资产","description":"黑色风衣，银色徽章，冷峻表情"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+projectID+"/assets/"+assetType, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("add %s status = %d body=%s", assetType, rec.Code, rec.Body.String())
	}
	var project map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &project); err != nil {
		t.Fatalf("decode project after asset: %v", err)
	}
	field := assetType
	if assetType == "character" {
		field = "characters"
	} else if assetType == "scene" {
		field = "scenes"
	} else if assetType == "prop" {
		field = "props"
	}
	assets, _ := project[field].([]any)
	if len(assets) == 0 {
		t.Fatalf("project missing %s: %#v", field, project)
	}
	asset, _ := assets[0].(map[string]any)
	id, _ := asset["id"].(string)
	if id == "" {
		t.Fatalf("asset missing id: %#v", asset)
	}
	return id
}

func uploadTestAssetVariant(t *testing.T, router http.Handler, projectID string, assetType string, assetID string, uploadType string) string {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, err := writer.CreateFormFile("file", "asset.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fileWriter.Write([]byte("fake-image-data")); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.WriteField("prompt", "测试上传图"); err != nil {
		t.Fatalf("write prompt field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+projectID+"/assets/"+assetType+"/"+assetID+"/upload?upload_type="+uploadType, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload asset variant status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	project, _ := response["project"].(map[string]any)
	assets, _ := project["characters"].([]any)
	asset, _ := assets[0].(map[string]any)
	unit, _ := asset["full_body"].(map[string]any)
	variants, _ := unit["image_variants"].([]any)
	if len(variants) == 0 {
		t.Fatalf("uploaded variant missing: %#v", response)
	}
	variant, _ := variants[0].(map[string]any)
	id, _ := variant["id"].(string)
	if id == "" {
		t.Fatalf("variant missing id: %#v", variant)
	}
	return id
}

func newTestRouterWithAuth(t *testing.T) (http.Handler, func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(options.Client().ApplyURI(testMongoURI()))
	if err != nil {
		cancel()
		t.Fatalf("connect mongo: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		cancel()
		_ = client.Disconnect(context.Background())
		t.Fatalf("ping mongo: %v", err)
	}

	dbName := "ltseries_auth_api_test_" + uuid.NewString()
	database := client.Database(dbName)
	q, err := queue.NewMongoQueue(ctx, database.Collection("tasks"))
	if err != nil {
		cancel()
		_ = client.Database(dbName).Drop(context.Background())
		_ = client.Disconnect(context.Background())
		t.Fatalf("create queue: %v", err)
	}
	projectService, err := services.NewProjectService(ctx, database.Collection("projects"), q)
	if err != nil {
		cancel()
		_ = client.Database(dbName).Drop(context.Background())
		_ = client.Disconnect(context.Background())
		t.Fatalf("create project service: %v", err)
	}

	router := httpapi.NewRouter(httpapi.Dependencies{
		ProjectService: projectService,
		Queue:          q,
		AuthService:    services.NewAuthService(true, "admin", "secret-pass", "test-secret", time.Hour),
	})

	cleanup := func() {
		_ = client.Database(dbName).Drop(context.Background())
		_ = client.Disconnect(context.Background())
		cancel()
	}
	return router, cleanup
}

func TestProjectAnalyzeAPIEnqueuesMongoTaskAndListsHistory(t *testing.T) {
	router, cleanup := newTestRouter(t)
	defer cleanup()

	createBody := []byte(`{"title":"我的短剧","script":"第一集：主角进入海边实验室。","workflow":"r2v"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", createRec.Code, createRec.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created project: %v", err)
	}
	projectID, _ := created["id"].(string)
	if projectID == "" {
		t.Fatalf("created project missing id: %#v", created)
	}

	analyzeReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+projectID+"/analyze", bytes.NewReader([]byte(`{}`)))
	analyzeReq.Header.Set("Content-Type", "application/json")
	analyzeRec := httptest.NewRecorder()
	router.ServeHTTP(analyzeRec, analyzeReq)
	if analyzeRec.Code != http.StatusAccepted {
		t.Fatalf("analyze status = %d body=%s", analyzeRec.Code, analyzeRec.Body.String())
	}

	var analyzeResp map[string]any
	if err := json.Unmarshal(analyzeRec.Body.Bytes(), &analyzeResp); err != nil {
		t.Fatalf("decode analyze response: %v", err)
	}
	taskID, _ := analyzeResp["task_id"].(string)
	if taskID == "" {
		t.Fatalf("analyze response missing task_id: %#v", analyzeResp)
	}

	taskReq := httptest.NewRequest(http.MethodGet, "/api/tasks/"+taskID, nil)
	taskRec := httptest.NewRecorder()
	router.ServeHTTP(taskRec, taskReq)
	if taskRec.Code != http.StatusOK {
		t.Fatalf("task status = %d body=%s", taskRec.Code, taskRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/tasks?project_id="+projectID, nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", listRec.Code, listRec.Body.String())
	}

	var listResp []map[string]any
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode task list: %v", err)
	}
	if len(listResp) != 1 || listResp[0]["id"] != taskID {
		t.Fatalf("task list = %#v, want task %s", listResp, taskID)
	}
}

func TestProjectUpdateScriptAPI(t *testing.T) {
	router, cleanup := newTestRouter(t)
	defer cleanup()

	createBody := []byte(`{"title":"原始标题","script":"第一集：原始剧本。","workflow":"r2v"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", createRec.Code, createRec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created project: %v", err)
	}
	projectID, _ := created["id"].(string)

	updateBody := []byte(`{"title":"更新标题","script":"第一集：更新后的剧本原文。","workflow":"i2v"}`)
	updateReq := httptest.NewRequest(http.MethodPut, "/api/projects/"+projectID, bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d body=%s", updateRec.Code, updateRec.Body.String())
	}
	var updated map[string]any
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated project: %v", err)
	}
	if updated["title"] != "更新标题" || updated["script"] != "第一集：更新后的剧本原文。" || updated["workflow"] != "i2v" {
		t.Fatalf("updated project = %#v", updated)
	}
}

func TestProjectListReturnsEmptyArrayAfterDeletingLastProject(t *testing.T) {
	router, cleanup := newTestRouter(t)
	defer cleanup()

	projectID := createTestProject(t, router)

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/projects/"+projectID, nil)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete project status = %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list projects status = %d body=%s", listRec.Code, listRec.Body.String())
	}
	if got := bytes.TrimSpace(listRec.Body.Bytes()); !bytes.Equal(got, []byte("[]")) {
		t.Fatalf("project list body = %s, want []", got)
	}
}

func TestSeriesListReturnsEmptyArrayWhenNoSeriesExist(t *testing.T) {
	router, cleanup := newTestRouter(t)
	defer cleanup()

	listReq := httptest.NewRequest(http.MethodGet, "/api/series", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list series status = %d body=%s", listRec.Code, listRec.Body.String())
	}
	if got := bytes.TrimSpace(listRec.Body.Bytes()); !bytes.Equal(got, []byte("[]")) {
		t.Fatalf("series list body = %s, want []", got)
	}
}

func TestTaskDeleteAPI(t *testing.T) {
	router, cleanup := newTestRouter(t)
	defer cleanup()

	createBody := []byte(`{"title":"任务删除","script":"第一集：主角进入房间。","workflow":"r2v"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", createRec.Code, createRec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created project: %v", err)
	}
	projectID, _ := created["id"].(string)

	analyzeReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+projectID+"/analyze", bytes.NewReader([]byte(`{}`)))
	analyzeReq.Header.Set("Content-Type", "application/json")
	analyzeRec := httptest.NewRecorder()
	router.ServeHTTP(analyzeRec, analyzeReq)
	if analyzeRec.Code != http.StatusAccepted {
		t.Fatalf("analyze status = %d body=%s", analyzeRec.Code, analyzeRec.Body.String())
	}
	var analyzeResp map[string]any
	if err := json.Unmarshal(analyzeRec.Body.Bytes(), &analyzeResp); err != nil {
		t.Fatalf("decode analyze response: %v", err)
	}
	taskID, _ := analyzeResp["task_id"].(string)

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/tasks/"+taskID, nil)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status = %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/tasks/"+taskID, nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusNotFound {
		t.Fatalf("get deleted status = %d body=%s", getRec.Code, getRec.Body.String())
	}
}

func TestSeriesEpisodeAPIAddsAndRemovesProject(t *testing.T) {
	router, cleanup := newTestRouter(t)
	defer cleanup()

	createProjectBody := []byte(`{"title":"第一集","script":"第一集：开场。","workflow":"r2v"}`)
	createProjectReq := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(createProjectBody))
	createProjectReq.Header.Set("Content-Type", "application/json")
	createProjectRec := httptest.NewRecorder()
	router.ServeHTTP(createProjectRec, createProjectReq)
	if createProjectRec.Code != http.StatusCreated {
		t.Fatalf("create project status = %d body=%s", createProjectRec.Code, createProjectRec.Body.String())
	}
	var createdProject map[string]any
	if err := json.Unmarshal(createProjectRec.Body.Bytes(), &createdProject); err != nil {
		t.Fatalf("decode created project: %v", err)
	}
	projectID, _ := createdProject["id"].(string)

	createSeriesBody := []byte(`{"title":"侦探短剧","description":"雨夜案件","workflow":"r2v"}`)
	createSeriesReq := httptest.NewRequest(http.MethodPost, "/api/series", bytes.NewReader(createSeriesBody))
	createSeriesReq.Header.Set("Content-Type", "application/json")
	createSeriesRec := httptest.NewRecorder()
	router.ServeHTTP(createSeriesRec, createSeriesReq)
	if createSeriesRec.Code != http.StatusCreated {
		t.Fatalf("create series status = %d body=%s", createSeriesRec.Code, createSeriesRec.Body.String())
	}
	var createdSeries map[string]any
	if err := json.Unmarshal(createSeriesRec.Body.Bytes(), &createdSeries); err != nil {
		t.Fatalf("decode created series: %v", err)
	}
	seriesID, _ := createdSeries["id"].(string)

	addBody := []byte(`{"project_id":"` + projectID + `"}`)
	addReq := httptest.NewRequest(http.MethodPost, "/api/series/"+seriesID+"/episodes", bytes.NewReader(addBody))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	router.ServeHTTP(addRec, addReq)
	if addRec.Code != http.StatusOK {
		t.Fatalf("add episode status = %d body=%s", addRec.Code, addRec.Body.String())
	}

	projectReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+projectID, nil)
	projectRec := httptest.NewRecorder()
	router.ServeHTTP(projectRec, projectReq)
	if projectRec.Code != http.StatusOK {
		t.Fatalf("get project status = %d body=%s", projectRec.Code, projectRec.Body.String())
	}
	var project map[string]any
	if err := json.Unmarshal(projectRec.Body.Bytes(), &project); err != nil {
		t.Fatalf("decode project: %v", err)
	}
	if project["series_id"] != seriesID || project["episode_number"].(float64) != 1 {
		t.Fatalf("project series fields = %#v", project)
	}

	removeReq := httptest.NewRequest(http.MethodDelete, "/api/series/"+seriesID+"/episodes/"+projectID, nil)
	removeRec := httptest.NewRecorder()
	router.ServeHTTP(removeRec, removeReq)
	if removeRec.Code != http.StatusOK {
		t.Fatalf("remove episode status = %d body=%s", removeRec.Code, removeRec.Body.String())
	}
	projectRec = httptest.NewRecorder()
	router.ServeHTTP(projectRec, projectReq)
	if projectRec.Code != http.StatusOK {
		t.Fatalf("get project after remove status = %d body=%s", projectRec.Code, projectRec.Body.String())
	}
	project = map[string]any{}
	if err := json.Unmarshal(projectRec.Body.Bytes(), &project); err != nil {
		t.Fatalf("decode project after remove: %v", err)
	}
	if project["series_id"] != nil && project["series_id"] != "" {
		t.Fatalf("project should be removed from series: %#v", project)
	}
}

func TestProjectDeleteRemovesProjectFromSeriesEpisodes(t *testing.T) {
	router, cleanup := newTestRouter(t)
	defer cleanup()

	projectID := createTestProject(t, router)
	createSeriesBody := []byte(`{"title":"删除联动剧集","description":"验证删除项目时清理剧集关系","workflow":"r2v"}`)
	createSeriesReq := httptest.NewRequest(http.MethodPost, "/api/series", bytes.NewReader(createSeriesBody))
	createSeriesReq.Header.Set("Content-Type", "application/json")
	createSeriesRec := httptest.NewRecorder()
	router.ServeHTTP(createSeriesRec, createSeriesReq)
	if createSeriesRec.Code != http.StatusCreated {
		t.Fatalf("create series status = %d body=%s", createSeriesRec.Code, createSeriesRec.Body.String())
	}
	var createdSeries map[string]any
	if err := json.Unmarshal(createSeriesRec.Body.Bytes(), &createdSeries); err != nil {
		t.Fatalf("decode created series: %v", err)
	}
	seriesID, _ := createdSeries["id"].(string)

	addReq := httptest.NewRequest(http.MethodPost, "/api/series/"+seriesID+"/episodes", bytes.NewReader([]byte(`{"project_id":"`+projectID+`"}`)))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	router.ServeHTTP(addRec, addReq)
	if addRec.Code != http.StatusOK {
		t.Fatalf("add episode status = %d body=%s", addRec.Code, addRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/projects/"+projectID, nil)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete project status = %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}

	seriesReq := httptest.NewRequest(http.MethodGet, "/api/series/"+seriesID, nil)
	seriesRec := httptest.NewRecorder()
	router.ServeHTTP(seriesRec, seriesReq)
	if seriesRec.Code != http.StatusOK {
		t.Fatalf("get series status = %d body=%s", seriesRec.Code, seriesRec.Body.String())
	}
	var series map[string]any
	if err := json.Unmarshal(seriesRec.Body.Bytes(), &series); err != nil {
		t.Fatalf("decode series: %v", err)
	}
	episodeIDs, _ := series["episode_ids"].([]any)
	if len(episodeIDs) != 0 {
		t.Fatalf("series should not retain deleted project id: %#v", series)
	}
}

func TestModelCatalogAPIExposesDashScopeModels(t *testing.T) {
	router, cleanup := newTestRouter(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("model catalog status = %d body=%s", rec.Code, rec.Body.String())
	}

	var body map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode model catalog: %v", err)
	}
	if len(body["image"]) < 5 {
		t.Fatalf("image models = %d, want at least 5", len(body["image"]))
	}
	if len(body["video"]) < 8 {
		t.Fatalf("video models = %d, want at least 8", len(body["video"]))
	}
	if body["image"][0]["id"] == "" || body["video"][0]["id"] == "" {
		t.Fatalf("catalog entries should include ids: %#v", body)
	}
}

func TestAssetVariantFavoriteAndDeleteAPI(t *testing.T) {
	router, cleanup := newTestRouter(t)
	defer cleanup()

	projectID := createTestProject(t, router)
	assetID := addTestAsset(t, router, projectID, "character")
	variantID := uploadTestAssetVariant(t, router, projectID, "character", assetID, "full_body")

	favoriteBody := []byte(`{"asset_id":"` + assetID + `","asset_type":"character","variant_id":"` + variantID + `","generation_type":"full_body","is_favorited":true}`)
	favoriteReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+projectID+"/assets/variant/favorite", bytes.NewReader(favoriteBody))
	favoriteReq.Header.Set("Content-Type", "application/json")
	favoriteRec := httptest.NewRecorder()
	router.ServeHTTP(favoriteRec, favoriteReq)
	if favoriteRec.Code != http.StatusOK {
		t.Fatalf("favorite status = %d body=%s", favoriteRec.Code, favoriteRec.Body.String())
	}
	var favoriteProject map[string]any
	if err := json.Unmarshal(favoriteRec.Body.Bytes(), &favoriteProject); err != nil {
		t.Fatalf("decode favorite project: %v", err)
	}
	characters, _ := favoriteProject["characters"].([]any)
	character, _ := characters[0].(map[string]any)
	fullBody, _ := character["full_body"].(map[string]any)
	variants, _ := fullBody["image_variants"].([]any)
	variant, _ := variants[0].(map[string]any)
	if variant["favorited"] != true {
		t.Fatalf("variant was not favorited: %#v", variant)
	}

	deleteBody := []byte(`{"asset_id":"` + assetID + `","asset_type":"character","variant_id":"` + variantID + `","generation_type":"full_body"}`)
	deleteReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+projectID+"/assets/variant/delete", bytes.NewReader(deleteBody))
	deleteReq.Header.Set("Content-Type", "application/json")
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete variant status = %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}
	var deleteProject map[string]any
	if err := json.Unmarshal(deleteRec.Body.Bytes(), &deleteProject); err != nil {
		t.Fatalf("decode delete project: %v", err)
	}
	characters, _ = deleteProject["characters"].([]any)
	character, _ = characters[0].(map[string]any)
	fullBody, _ = character["full_body"].(map[string]any)
	variants, _ = fullBody["image_variants"].([]any)
	if len(variants) != 0 || fullBody["selected_image_id"] != nil {
		t.Fatalf("variant should be removed and selection cleared: %#v", fullBody)
	}
}

func TestGenerateAllAssetsAPIEnqueuesMissingAssetTasks(t *testing.T) {
	router, cleanup := newTestRouter(t)
	defer cleanup()

	projectID := createTestProject(t, router)
	_ = addTestAsset(t, router, projectID, "character")
	_ = addTestAsset(t, router, projectID, "scene")
	_ = addTestAsset(t, router, projectID, "prop")

	body := []byte(`{"model":"wan2.7-image-pro","parameters":{"aspect_ratio":"9:16"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+projectID+"/generate_assets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("generate assets status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		TaskIDs []string         `json:"task_ids"`
		Tasks   []map[string]any `json:"tasks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode generate assets response: %v", err)
	}
	if len(response.TaskIDs) != 3 || len(response.Tasks) != 3 {
		t.Fatalf("queued tasks = %#v", response)
	}
	for _, task := range response.Tasks {
		if task["type"] != string("asset_generation") {
			t.Fatalf("unexpected task type: %#v", task)
		}
	}
}

func TestAssetVideoGenerationAPIEnqueuesAssetVideoTask(t *testing.T) {
	router, cleanup := newTestRouter(t)
	defer cleanup()

	projectID := createTestProject(t, router)
	assetID := addTestAsset(t, router, projectID, "character")
	_ = uploadTestAssetVariant(t, router, projectID, "character", assetID, "full_body")

	body := []byte(`{"prompt":"角色转身展示服装","upload_type":"full_body","model":"wan2.7-i2v","duration":5}`)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+projectID+"/assets/character/"+assetID+"/generate_video", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("asset video status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		TaskID string         `json:"task_id"`
		Task   map[string]any `json:"task"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode asset video response: %v", err)
	}
	if response.TaskID == "" || response.Task["type"] != string("asset_video") {
		t.Fatalf("unexpected asset video task: %#v", response)
	}
	payload, _ := response.Task["payload"].(map[string]any)
	if payload["asset_id"] != assetID || payload["image_url"] == "" {
		t.Fatalf("asset video payload missing selected image: %#v", payload)
	}
}

func TestStoryboardCompleteAPIEnqueuesPipelineTask(t *testing.T) {
	router, cleanup := newTestRouter(t)
	defer cleanup()

	projectID := createTestProject(t, router)
	body := []byte(`{"model":"wan2.7-i2v","duration":5,"parameters":{"ratio":"9:16","audio":false}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+projectID+"/storyboard/complete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("storyboard complete status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		TaskID string         `json:"task_id"`
		Task   map[string]any `json:"task"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode storyboard complete response: %v", err)
	}
	if response.TaskID == "" || response.Task["type"] != string("storyboard_pipeline") {
		t.Fatalf("unexpected storyboard pipeline task: %#v", response)
	}
}

func TestProductionAuditAPIReportsReadinessGaps(t *testing.T) {
	router, cleanup := newTestRouter(t)
	defer cleanup()

	projectID := createTestProject(t, router)
	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+projectID+"/production-audit", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("audit status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		ProjectID        string                   `json:"project_id"`
		ReadyForExport   bool                     `json:"ready_for_export"`
		EfficiencyScore  int                      `json:"efficiency_score"`
		ConsistencyScore int                      `json:"consistency_score"`
		Metrics          map[string]int           `json:"metrics"`
		Checks           map[string]bool          `json:"checks"`
		WorkflowStages   []map[string]interface{} `json:"workflow_stages"`
		FrameAudits      []map[string]interface{} `json:"frame_audits"`
		Issues           []map[string]interface{} `json:"issues"`
		Recommendations  []string                 `json:"recommendations"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode audit: %v", err)
	}
	if response.ProjectID != projectID {
		t.Fatalf("project_id = %q, want %q", response.ProjectID, projectID)
	}
	if response.ReadyForExport {
		t.Fatalf("new project should not be export ready: %#v", response)
	}
	if response.Metrics["frames"] != 0 {
		t.Fatalf("frames metric = %d, want 0", response.Metrics["frames"])
	}
	if response.Checks["storyboard"] {
		t.Fatalf("storyboard check should be false: %#v", response.Checks)
	}
	if response.EfficiencyScore <= 0 || response.ConsistencyScore < 0 {
		t.Fatalf("audit scores not populated: %#v", response)
	}
	if len(response.WorkflowStages) == 0 {
		t.Fatalf("workflow stages should be populated: %#v", response)
	}
	if len(response.Issues) == 0 || len(response.Recommendations) == 0 {
		t.Fatalf("audit should include issues and recommendations: %#v", response)
	}
}

func TestAuthProtectsAPIWhenEnabled(t *testing.T) {
	router, cleanup := newTestRouterWithAuth(t)
	defer cleanup()

	protectedReq := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	protectedRec := httptest.NewRecorder()
	router.ServeHTTP(protectedRec, protectedReq)
	if protectedRec.Code != http.StatusUnauthorized {
		t.Fatalf("protected status = %d body=%s", protectedRec.Code, protectedRec.Body.String())
	}

	loginBody := []byte(`{"username":"admin","password":"secret-pass"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", loginRec.Code, loginRec.Body.String())
	}
	var loginResp map[string]any
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	token, _ := loginResp["token"].(string)
	if token == "" {
		t.Fatalf("login response missing token: %#v", loginResp)
	}

	authorizedReq := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	authorizedReq.Header.Set("Authorization", "Bearer "+token)
	authorizedRec := httptest.NewRecorder()
	router.ServeHTTP(authorizedRec, authorizedReq)
	if authorizedRec.Code != http.StatusOK {
		t.Fatalf("authorized status = %d body=%s", authorizedRec.Code, authorizedRec.Body.String())
	}
}
