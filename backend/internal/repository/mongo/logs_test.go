package mongo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/models"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type testCursor struct {
	items []logDoc
	idx   int
	err   error
}

func (c *testCursor) Next(context.Context) bool {
	return c.idx < len(c.items)
}

func (c *testCursor) Decode(val any) error {
	if c.err != nil {
		return c.err
	}
	d, ok := val.(*logDoc)
	if !ok {
		return errors.New("bad decode type")
	}
	*d = c.items[c.idx]
	c.idx++
	return nil
}

func (c *testCursor) Err() error { return c.err }
func (c *testCursor) Close(context.Context) error { return nil }

type testCollection struct {
	cur cursor
	err error
}

func (c testCollection) Find(context.Context, any, ...*options.FindOptions) (cursor, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.cur, nil
}

func TestLogsRepositoryListByService(t *testing.T) {
	serviceID := uuid.New()
	now := time.Now().UTC().Unix()

	repo := NewLogsRepository(testCollection{
		cur: &testCursor{
			items: []logDoc{
				{ID: "1", ServiceID: serviceID.String(), Environment: "dev", Level: "info", Message: "ok", Timestamp: now},
			},
		},
	})

	got, err := repo.ListByService(context.Background(), serviceID, models.LogFilter{})
	if err != nil {
		t.Fatalf("ListByService() err = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].ServiceID != serviceID {
		t.Fatalf("service_id = %v, want %v", got[0].ServiceID, serviceID)
	}
	if got[0].Level != "info" {
		t.Fatalf("level = %s, want info", got[0].Level)
	}
}

