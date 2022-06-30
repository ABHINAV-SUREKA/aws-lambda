package app

type Config interface {
	FormatEventMessage() ([]byte, error)
	SendNotification(HttpRequest, chan struct{})
}

type config struct {
	event interface{}
}

func New(event interface{}) Config {
	return &config{
		event: event,
	}
}
