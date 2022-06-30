package app

type Config interface {
	FormatEventMessage() ([]byte, error)
	SendNotification(HttpRequest, chan struct{})
}

type config struct {
	ctx   interface{}
	event interface{}
}

func New(ctx interface{}, event interface{}) Config {
	return &config{
		ctx:   ctx,
		event: event,
	}
}
