package queue

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"ltseries/backend/internal/models"
)

type TaskFilter struct {
	ProjectID string
	Status    models.TaskStatus
	Type      models.TaskType
	Limit     int64
}

type MongoQueue struct {
	collection *mongo.Collection
}

func NewMongoQueue(ctx context.Context, collection *mongo.Collection) (*MongoQueue, error) {
	q := &MongoQueue{collection: collection}
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: 1}}},
		{Keys: bson.D{{Key: "project_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "type", Value: 1}, {Key: "created_at", Value: -1}}},
	}
	if _, err := collection.Indexes().CreateMany(ctx, indexes); err != nil {
		return nil, err
	}
	return q, nil
}

func (q *MongoQueue) Enqueue(ctx context.Context, task *models.Task) error {
	now := time.Now().UTC()
	if task.Status == "" {
		task.Status = models.TaskStatusPending
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = now
	}
	if task.Progress < 0 {
		task.Progress = 0
	}
	if task.Status == models.TaskStatusCompleted && task.Progress == 0 {
		task.Progress = 100
	}
	if task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusFailed {
		if task.FinishedAt == nil {
			task.FinishedAt = &now
		}
	}

	_, err := q.collection.UpdateOne(
		ctx,
		bson.M{"_id": task.ID},
		bson.M{"$setOnInsert": task},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (q *MongoQueue) ClaimNext(ctx context.Context, workerID string) (*models.Task, error) {
	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"status":     models.TaskStatusProcessing,
			"worker_id":  workerID,
			"started_at": now,
			"updated_at": now,
			"error":      "",
		},
		"$inc": bson.M{"attempts": 1},
	}
	opts := options.FindOneAndUpdate().
		SetSort(bson.D{{Key: "created_at", Value: 1}}).
		SetReturnDocument(options.After)

	var task models.Task
	err := q.collection.FindOneAndUpdate(
		ctx,
		bson.M{"status": models.TaskStatusPending},
		update,
		opts,
	).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (q *MongoQueue) UpdateStatus(
	ctx context.Context,
	taskID string,
	status models.TaskStatus,
	progress int,
	errorMessage string,
	result map[string]any,
) error {
	now := time.Now().UTC()
	set := bson.M{
		"status":     status,
		"progress":   progress,
		"error":      errorMessage,
		"updated_at": now,
	}
	if result != nil {
		set["result"] = result
	}
	if status == models.TaskStatusProcessing {
		set["started_at"] = now
	}
	if status == models.TaskStatusCompleted {
		if progress == 0 {
			set["progress"] = 100
		}
		set["finished_at"] = now
	}
	if status == models.TaskStatusFailed {
		set["finished_at"] = now
	}

	_, err := q.collection.UpdateOne(ctx, bson.M{"_id": taskID}, bson.M{"$set": set})
	return err
}

func (q *MongoQueue) Get(ctx context.Context, taskID string) (*models.Task, error) {
	var task models.Task
	err := q.collection.FindOne(ctx, bson.M{"_id": taskID}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (q *MongoQueue) Delete(ctx context.Context, taskID string) (bool, error) {
	result, err := q.collection.DeleteOne(ctx, bson.M{"_id": taskID})
	if err != nil {
		return false, err
	}
	return result.DeletedCount > 0, nil
}

func (q *MongoQueue) List(ctx context.Context, filter TaskFilter) ([]models.Task, error) {
	query := bson.M{}
	if filter.ProjectID != "" {
		query["project_id"] = filter.ProjectID
	}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if filter.Type != "" {
		query["type"] = filter.Type
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	cursor, err := q.collection.Find(
		ctx,
		query,
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	tasks := make([]models.Task, 0)
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (q *MongoQueue) RequeueProcessing(ctx context.Context, olderThan time.Duration) (int64, error) {
	if olderThan <= 0 {
		return 0, nil
	}
	cutoff := time.Now().UTC().Add(-olderThan)
	result, err := q.collection.UpdateMany(
		ctx,
		bson.M{"status": models.TaskStatusProcessing, "updated_at": bson.M{"$lt": cutoff}},
		bson.M{
			"$set": bson.M{
				"status":     models.TaskStatusPending,
				"updated_at": time.Now().UTC(),
				"error":      "Task was requeued after worker timeout.",
			},
			"$unset": bson.M{"worker_id": "", "started_at": ""},
		},
	)
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}
