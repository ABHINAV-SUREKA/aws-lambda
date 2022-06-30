package app

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
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

func notifySlack(payloadByteArr []byte, alertChan chan int) error {
	for i := 0; i < requestRetries; i++ {
		func() error {
			resp, err := http.Post(slackURL, "application/json", bytes.NewBuffer(payloadByteArr))
			if err != nil {
				return errors.New(fmt.Sprintf("error notifying Slack: %s. Retrying...", err))
			}
			defer resp.Body.Close()
			if statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300; !statusOK {
				return errors.New(fmt.Sprintf("non-OK HTTP status: %v. Retrying...", resp.StatusCode))
			}
			return nil
		}()
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	sb := string(body)
	log.Printf(sb)

	return nil
}

func notifyPagerDuty(payloadByteArr []byte, alertChan chan int) error {
	resp, err := http.Post(pagerDutyURL, "application/json", bytes.NewBuffer(payloadByteArr))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
