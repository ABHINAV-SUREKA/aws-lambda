package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ABHINAV-SUREKA/aws-lambda/constants"
	"github.com/aws/aws-lambda-go/events"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

type payload struct {
	Text Message `json:"text"`
}

type Message map[string]interface{}

type HttpRequest struct {
	URL     string
	Headers map[string]string
	Method  string
	Body    []byte
}

func (config *config) FormatEventMessage() ([]byte, error) {
	message := Message{}

	if len(config.event.(events.SNSEvent).Records) > 0 {
		eventMsg := config.event.(events.SNSEvent).Records[0].SNS.Message
		eventMsgItems := strings.Split(eventMsg, "\n")
		for _, eventMsgItem := range eventMsgItems {
			if !strings.Contains(eventMsgItem, ":") {
				continue
			}
			item := strings.Split(eventMsgItem, ":")
			key := strings.TrimSpace(item[0])
			val := strings.TrimSpace(item[1])
			log.Infof("%s:%v", key, val)
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

		if strings.Index(config.event.(events.SNSEvent).Records[0].SNS.Subject, "[RESOLVED]") != -1 {
			message["event_action"] = "resolve"
		} else {
			message["event_action"] = "trigger"
		}
	}

	message["source"] = "amp-alerting"

	payloadByteArr, err := json.MarshalIndent(payload{Text: message}, "", "  ")
	if err != nil {
		return []byte{}, err
	}

	return payloadByteArr, nil
}

func (config *config) SendNotification(httpReq HttpRequest, notifyChan chan struct{}) {
	i := 0
	log.Info("string(httpReq.Body): ", string(httpReq.Body))
	for ; i < constants.RequestRetries; i++ {
		if err := func() error {
			req, err := http.NewRequest(httpReq.Method, httpReq.URL, bytes.NewBuffer(httpReq.Body))
			if err != nil {
				return errors.New(fmt.Sprintf("Error sending notification to %s: %s. Retrying...", httpReq.URL, err))
			}
			for key, val := range httpReq.Headers {
				req.Header.Add(key, val)
			}

			ctx, cancel := context.WithTimeout(req.Context(), constants.RequestTimeout*time.Second)
			defer cancel()
			req = req.WithContext(ctx)

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return errors.New(fmt.Sprintf("Error sending notification to %s: %s. Retrying...", httpReq.URL, err))
			}
			defer resp.Body.Close()
			if statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300; !statusOK {
				return errors.New(fmt.Sprintf("Error sending notification to %s: non-OK HTTP status: %v. Retrying...", httpReq.URL, resp.StatusCode))
			}
			return nil

		}(); err != nil {
			log.Error(err)
			time.Sleep(constants.RequestSleep * time.Second)
			continue
		}
		break
	}
	if i == constants.RequestRetries {
		log.Errorf("Failed to send notification to %s: request retries exhausted", httpReq.URL)
	} else {
		log.Infof("Successfully sent notification to %s", httpReq.URL)
	}
	notifyChan <- struct{}{}
}
