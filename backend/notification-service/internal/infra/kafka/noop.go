package kafka

import "context"

type NoopPublisher struct{}

func (NoopPublisher) Publish(context.Context, string, string, []byte) error { return nil }
