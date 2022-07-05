package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "02-01-2006 15:04:05",
		FullTimestamp:   true,
	})
}

func HandleLambdaEvent(event events.SNSEvent) {
	formatEventMessage(event)
}

func main() {
	lambda.Start(HandleLambdaEvent)
}
