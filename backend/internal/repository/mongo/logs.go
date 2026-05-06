package mongo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type cursor interface {
	Next(ctx context.Context) bool
	Decode(val any) error
	Err() error
	Close(ctx context.Context) error
}

type collection interface {
	Find(ctx context.Context, filter any, opts ...*options.FindOptions) (cursor, error)
}

type LogsRepository struct {
	col collection
}

var ErrCollectionNotConfigured = errors.New("mongo logs collection is not configured")

func NewLogsRepository(col collection) *LogsRepository {
	return &LogsRepository{col: col}
}

type logDoc struct {
	ID          any       `bson:"_id"`
	ServiceID   string    `bson:"service_id"`
	UserID      string    `bson:"user_id"`
	Environment string    `bson:"environment"`
	Level       string    `bson:"level"`
	Message     string    `bson:"message"`
	Timestamp   int64     `bson:"timestamp"`
	CreatedAt   int64     `bson:"created_at"`
	At          int64     `bson:"at"`
	TS          any       `bson:"ts"`
	Time        any       `bson:"time"`
	Date        any       `bson:"date"`
	RawTime     any       `bson:"raw_time"`
	ParsedTime  any       `bson:"parsed_time"`
	UnixTime    any       `bson:"unix_time"`
	Nanotime    any       `bson:"nanotime"`
	RecordedAt  any       `bson:"recorded_at"`
	EventTime   any       `bson:"event_time"`
}

func (r *LogsRepository) ListByService(ctx context.Context, serviceID uuid.UUID, filter models.LogFilter) ([]models.LogEntry, error) {
	if r.col == nil {
		return nil, ErrCollectionNotConfigured
	}
	q := bson.M{"service_id": serviceID.String()}
	if filter.Level != "" {
		q["level"] = filter.Level
	}
	if filter.Text != "" {
		q["message"] = bson.M{"$regex": filter.Text, "$options": "i"}
	}
	if filter.TimeFrom != nil || filter.TimeTo != nil {
		t := bson.M{}
		if filter.TimeFrom != nil {
			t["$gte"] = filter.TimeFrom.Unix()
		}
		if filter.TimeTo != nil {
			t["$lte"] = filter.TimeTo.Unix()
		}
		q["timestamp"] = t
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 200
	}
	cur, err := r.col.Find(ctx, q, options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	out := make([]models.LogEntry, 0)
	for cur.Next(ctx) {
		var d logDoc
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		id, err := uuid.Parse(d.ServiceID)
		if err != nil {
			return nil, err
		}
		out = append(out, models.LogEntry{
			ID:          toID(d.ID),
			ServiceID:   id,
			UserID:      d.UserID,
			Environment: d.Environment,
			Level:       d.Level,
			Message:     d.Message,
			Timestamp:   unixToTime(d.Timestamp),
		})
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
