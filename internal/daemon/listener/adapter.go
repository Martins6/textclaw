package listener

import (
	"context"
	"time"
)

type Message struct {
	Sender      string
	Content     string
	ContentType string
	ChatID      string
	Timestamp   time.Time
	Raw         interface{}
}

type MessageHandler func(ctx context.Context, msg Message) error

type Adapter interface {
	Listen(ctx context.Context, handler MessageHandler) error
	Send(target string, message string) error
	Name() string
}
