package subscription

import (
	"context"
	"testing"
	"time"

	"github.com/Henzer3/test-ozon/post-service/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestBrokerPublishesToPostSubscribers(t *testing.T) {
	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	firstSubscriber := broker.Subscribe(ctx, 10)
	secondSubscriber := broker.Subscribe(ctx, 10)
	otherPostSubscriber := broker.Subscribe(ctx, 20)
	want := entity.Comment{ID: 1, PostID: 10, Content: "comment"}

	broker.Publish(want)

	require.Equal(t, want, <-firstSubscriber)
	require.Equal(t, want, <-secondSubscriber)
	select {
	case comment := <-otherPostSubscriber:
		t.Fatalf("received comment for another post: %+v", comment)
	default:
	}
}

func TestBrokerRemovesCanceledSubscriber(t *testing.T) {
	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	subscriber := broker.Subscribe(ctx, 10)

	cancel()

	select {
	case _, open := <-subscriber:
		require.False(t, open)
	case <-time.After(time.Second):
		t.Fatal("subscriber channel was not closed after context cancellation")
	}

	broker.mu.RLock()
	_, exists := broker.subscribers[10]
	broker.mu.RUnlock()
	require.False(t, exists)
}

func TestBrokerDoesNotBlockOnSlowSubscriber(t *testing.T) {
	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	subscriber := broker.Subscribe(ctx, 10)
	const commentsCount = 32

	done := make(chan struct{})
	go func() {
		for id := int64(1); id <= commentsCount; id++ {
			broker.Publish(entity.Comment{ID: id, PostID: 10})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publishing blocked on a slow subscriber")
	}

	for id := int64(1); id <= commentsCount; id++ {
		select {
		case comment := <-subscriber:
			require.Equal(t, id, comment.ID)
		case <-time.After(time.Second):
			t.Fatalf("comment %d was not delivered", id)
		}
	}
}
