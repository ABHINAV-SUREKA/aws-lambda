package main

import (
	"context"
	"github.com/ABHINAV-SUREKA/aws-lambda/internal/app"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/sirupsen/logrus"
)

var (
	config app.Config
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "02-01-2006 15:04:05",
		FullTimestamp:   true,
	})
}

func HandleLambdaEvent(ctx context.Context, event events.SNSEvent) (string, error) {
	config = app.New(ctx, event)
	return config.Handler()
}

func main() {
	lambda.Start(HandleLambdaEvent)
}
