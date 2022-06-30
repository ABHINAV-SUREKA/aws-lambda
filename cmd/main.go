package main

import (
	"context"
	"github.com/ABHINAV-SUREKA/aws-lambda/constants"
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

func HandleLambdaEvent(ctx context.Context, event events.SNSEvent) {
	config = app.New(ctx, event)
	notifyChan := make(chan struct{}, 1)
	payloadByteArr, err := config.FormatEventMessage()
	if err != nil {
		log.Errorf("Failed to format event: %s", err)
	} else {
		go func() {
			httpReq := app.HttpRequest{
				Url: constants.SlackURL,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Method: "POST",
				Body:   payloadByteArr,
			}
			config.SendNotification(httpReq, notifyChan)
		}()

		go func() {
			httpReq := app.HttpRequest{
				Url: constants.PagerDutyURL,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Method: "POST",
				Body:   payloadByteArr,
			}
			config.SendNotification(httpReq, notifyChan)
		}()

		for i := 0; i < 2; i++ {
			<-notifyChan
		}
	}
}

func main() {
	lambda.Start(HandleLambdaEvent)
}
