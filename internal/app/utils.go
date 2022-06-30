package app

import (
	"github.com/aws/aws-lambda-go/events"
	"strings"
)

func formatEvent(event events.SNSEvent) payload {
	message := Message{}

	if len(event.Records) > 0 {
		eventMsg := event.Records[0].SNS.Message
		eventMsgItems := strings.Split(eventMsg, "\n")
		for _, eventMsgItem := range eventMsgItems {
			item := strings.Split(eventMsgItem, ":")
			key := strings.TrimSpace(item[0])
			val := strings.TrimSpace(item[1])
			switch {
			case strings.Contains(strings.ToLower(key), "client_url"),
				strings.Contains(strings.ToLower(key), "severity"),
				strings.Contains(strings.ToLower(key), "description"):
				message[key] = val
			case strings.Contains(strings.ToLower(key), "links"):
				if val != "" {
					message["links"] = val
				}
			case strings.Contains(strings.ToLower(key), "details"):
				if val != "" {
					message["custom_details"] = val
				}
			}
		}

		if strings.Index(event.Records[0].SNS.Subject, "[RESOLVED]") != -1 {
			message["event_action"] = "resolve"
		} else {
			message["event_action"] = "trigger"
		}
	}

	return payload{
		Payload: message,
	}
}
