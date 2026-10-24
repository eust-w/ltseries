package queue_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"ltseries/backend/internal/models"
	"ltseries/backend/internal/queue"
)

func testMongoURI() string {
	if uri := os.Getenv("LTSERIES_TEST_MONGODB_URI"); uri != "" {
		return uri
	}
	return "mongodb://127.0.0.1:27019"
}

func newTestQueue(t *testing.T) (*queue.MongoQueue, func()) {
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

	dbName := "ltseries_queue_test_" + uuid.NewString()
	q, err := queue.NewMongoQueue(ctx, client.Database(dbName).Collection("tasks"))
	if err != nil {
		cancel()
		_ = client.Database(dbName).Drop(context.Background())
		_ = client.Disconnect(context.Background())
		t.Fatalf("create queue: %v", err)
	}

	cleanup := func() {
		_ = client.Database(dbName).Drop(context.Background())
		_ = client.Disconnect(context.Background())
		cancel()
	}
	return q, cleanup
}

func TestMongoQueueEnqueueClaimAndPersistFailure(t *testing.T) {
	q, cleanup := newTestQueue(t)
	defer cleanup()

	ctx := context.Background()
	taskID := uuid.NewString()
	task := &models.Task{
		ID:        taskID,
		Type:      models.TaskTypeAnalysis,
		Status:    models.TaskStatusPending,
		Progress:  0,
		ProjectID: "project-1",
		Payload: map[string]any{
			"text": "第一集：角色进入海边实验室。",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := q.Enqueue(ctx, task); err != nil {
		t.Fatalf("enqueue task: %v", err)
	}
	if err := q.Enqueue(ctx, task); err != nil {
		t.Fatalf("idempotent enqueue task: %v", err)
	}

	claimed, err := q.ClaimNext(ctx, "worker-1")
	if err != nil {
		t.Fatalf("claim next: %v", err)
	}
	if claimed == nil {
		t.Fatal("expected one claimed task")
	}
	if claimed.ID != taskID {
		t.Fatalf("claimed task id = %s, want %s", claimed.ID, taskID)
	}
	if claimed.Status != models.TaskStatusProcessing {
		t.Fatalf("claimed status = %s, want processing", claimed.Status)
	}
	if claimed.WorkerID != "worker-1" {
		t.Fatalf("worker id = %s, want worker-1", claimed.WorkerID)
	}
	if claimed.Attempts != 1 {
		t.Fatalf("attempts = %d, want 1", claimed.Attempts)
	}

	if err := q.UpdateStatus(ctx, taskID, models.TaskStatusFailed, 0, "provider rejected request", nil); err != nil {
		t.Fatalf("update status: %v", err)
	}

	stored, err := q.Get(ctx, taskID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if stored.Status != models.TaskStatusFailed {
		t.Fatalf("stored status = %s, want failed", stored.Status)
	}
	if stored.Error != "provider rejected request" {
		t.Fatalf("stored error = %q", stored.Error)
	}
	if stored.FinishedAt == nil {
		t.Fatal("finished_at should be persisted for failed task")
	}

	tasks, err := q.List(ctx, queue.TaskFilter{ProjectID: "project-1", Limit: 20})
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != taskID {
		t.Fatalf("listed tasks = %#v, want exactly %s", tasks, taskID)
	}
}

func TestMongoQueueListEmptyReturnsNonNilSlice(t *testing.T) {
	q, cleanup := newTestQueue(t)
	defer cleanup()

	tasks, err := q.List(context.Background(), queue.TaskFilter{ProjectID: "missing-project", Limit: 20})
	if err != nil {
		t.Fatalf("list empty tasks: %v", err)
	}
	if tasks == nil {
		t.Fatal("empty task list should be an empty slice, not nil")
	}
	if len(tasks) != 0 {
		t.Fatalf("empty task list length = %d, want 0", len(tasks))
	}
}
