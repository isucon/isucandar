package pubsub

import (
	"context"
	"sync"
)

type PubSub struct {
	Capacity int

	mu sync.RWMutex
	ch []chan interface{}
}

func NewPubSub() *PubSub {
	return &PubSub{
		Capacity: 10,
		mu:       sync.RWMutex{},
		ch:       []chan interface{}{},
	}
}

func (p *PubSub) Publish(payload interface{}) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, ch := range p.ch {
		ch <- payload
	}
}

func (p *PubSub) Subscribe(ctx context.Context, f func(interface{})) <-chan bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	ch := make(chan interface{}, p.Capacity)
	p.ch = append(p.ch, ch)

	sub := &Subscription{
		pubsub: p,
		f:      f,
		ch:     ch,
	}

	waiter := make(chan bool)

	go func() {
	L:
		for ctx.Err() == nil {
			select {
			case payload, ok := <-sub.ch:
				if ok {
					sub.f(payload)
				}
			case <-ctx.Done():
				sub.close()
				break L
			}
		}
		close(waiter)
	}()

	return waiter
}

type Subscription struct {
	pubsub *PubSub
	f      func(interface{})
	ch     chan interface{}
}

func (s *Subscription) close() {
	s.pubsub.mu.Lock()
	defer s.pubsub.mu.Unlock()

	for idx, ch := range s.pubsub.ch {
		if ch == s.ch {
			deleted := append(s.pubsub.ch[:idx], s.pubsub.ch[idx+1:]...)
			channels := make([]chan interface{}, len(deleted))
			copy(channels, deleted)
			s.pubsub.ch = channels
		}
	}

	close(s.ch)

	s.ch = nil
}
