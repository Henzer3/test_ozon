package subscription

import (
	"context"
	"sync"

	"github.com/Henzer3/test-ozon/post-service/internal/entity"
)

type Broker struct {
	mu          sync.RWMutex
	subscribers map[int64]map[*subscriber]struct{}
}

type subscriber struct {
	mu     sync.Mutex
	output chan entity.Comment
	wake   chan struct{}
	done   chan struct{}
	queue  []entity.Comment
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[int64]map[*subscriber]struct{}),
	}
}

func (b *Broker) Publish(comment entity.Comment) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for subscriber := range b.subscribers[comment.PostID] {
		subscriber.enqueue(comment)
	}
}

func (b *Broker) Subscribe(ctx context.Context, postID int64) <-chan entity.Comment {
	sub := &subscriber{
		output: make(chan entity.Comment, 1),
		wake:   make(chan struct{}, 1),
		done:   make(chan struct{}),
	}

	b.mu.Lock()
	if b.subscribers[postID] == nil {
		b.subscribers[postID] = make(map[*subscriber]struct{})
	}
	b.subscribers[postID][sub] = struct{}{}
	b.mu.Unlock()

	go sub.run()
	go func() {
		<-ctx.Done()

		b.mu.Lock()
		postSubscribers, ok := b.subscribers[postID]
		if !ok {
			b.mu.Unlock()
			return
		}
		if _, ok := postSubscribers[sub]; !ok {
			b.mu.Unlock()
			return
		}

		delete(postSubscribers, sub)
		if len(postSubscribers) == 0 {
			delete(b.subscribers, postID)
		}
		b.mu.Unlock()

		close(sub.done)
	}()

	return sub.output
}

func (s *subscriber) enqueue(comment entity.Comment) {
	s.mu.Lock()
	s.queue = append(s.queue, comment)
	s.mu.Unlock()

	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *subscriber) run() {
	defer close(s.output)

	for {
		comment, ok := s.dequeue()
		if ok {
			select {
			case s.output <- comment:
			case <-s.done:
				return
			}
			continue
		}

		select {
		case <-s.wake:
		case <-s.done:
			return
		}
	}
}

func (s *subscriber) dequeue() (entity.Comment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.queue) == 0 {
		return entity.Comment{}, false
	}

	comment := s.queue[0]
	s.queue[0] = entity.Comment{}
	s.queue = s.queue[1:]
	return comment, true
}
