package app

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
)

type Config interface {
	Handler() (string, error)
}

type config struct {
	ctx   context.Context
	event events.SNSEvent
}

func New(ctx context.Context, event events.SNSEvent) Config {
	return &config{
		ctx:   ctx,
		event: event,
	}
}
