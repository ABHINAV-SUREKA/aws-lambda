package main

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

type keyVal map[string]interface{}

type payload struct {
	keyVal
}

type pdPayload struct {
	Payload payload `json:"payload"`
	keyVal
}

type HttpRequest struct {
	URL     string
	Headers map[string]string
	Method  string
	Body    []byte
}

func formatEventMessage(event events.SNSEvent) {

	httpReq, err := func() (*HttpRequest, error) {
		httpReq := HttpRequest{}

		if len(event.Records) > 0 {
			eventMsg := event.Records[0].SNS.Message
			eventSub := event.Records[0].SNS.Subject

			if strings.Contains(strings.ToLower(eventMsg), "routing_key") {
				// format event message for pager duty
				payload := payload{}
				keyVal := keyVal{}

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
						strings.Contains(strings.ToLower(key), "links"):
						if val != "" {
							keyVal[key] = val
						}
					case strings.Contains(strings.ToLower(key), "severity"):
						payload.keyVal[key] = val
					case strings.Contains(strings.ToLower(key), "description"):
						payload.keyVal["summary"] = val
					case strings.Contains(strings.ToLower(key), "details"):
						if val != "" {
							payload.keyVal["custom_details"] = val
						}
					case strings.Contains(strings.ToLower(key), "routing_key"):
						httpReq.Headers["x-routing-key"] = val
					}
				}

				if strings.Contains(strings.ToLower(eventSub), "resolve") {
					keyVal["event_action"] = "resolve"
				} else {
					keyVal["event_action"] = "trigger"
				}

				bodyByteArr, err := json.MarshalIndent(pdPayload{Payload: payload, keyVal: keyVal}, "", "  ")
				if err != nil {
					return nil, err
				}

				httpReq.Body = bodyByteArr
				httpReq.URL = constants.PagerDutyURL
				httpReq.Method = "POST"
				httpReq.Headers["Content-Type"] = "application/json"
				return &httpReq, nil

			} else if strings.Contains(strings.ToLower(eventMsg), "channel") {
				// format event message for slack
				return nil, nil
			}
		}
		return nil, nil
	}()

	if err != nil {
		log.Error(err)
	}

	sendNotification(*httpReq)
}

func sendNotification(httpReq HttpRequest) {
	var (
		i      = 0
		err    error
		req    *http.Request
		resp   *http.Response
		client = &http.Client{}
	)
	log.Info("string(httpReq.Body): ", string(httpReq.Body))

	for ; i < constants.RequestRetries; i++ {
		if err = func() error {
			req, err = http.NewRequest(httpReq.Method, httpReq.URL, bytes.NewBuffer(httpReq.Body))
			if err != nil {
				return errors.New(fmt.Sprintf("Error sending notification to %s: %s. Retrying...", httpReq.URL, err))
			}
			for key, val := range httpReq.Headers {
				req.Header.Add(key, val)
			}

			ctx, cancel := context.WithTimeout(req.Context(), constants.RequestTimeout*time.Second)
			defer cancel()
			req = req.WithContext(ctx)

			resp, err = client.Do(req)
			if err != nil {
				return errors.New(fmt.Sprintf("Error sending notification to %s: %s. Retrying...", httpReq.URL, err))
			}

			defer resp.Body.Close()
			if statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300; !statusOK {
				return errors.New(fmt.Sprintf("Error sending notification to %s: non-OK HTTP status: %v. Retrying...", httpReq.URL, resp.StatusCode))
			}

			return nil

		}(); err == nil {
			break
		}

		log.Error(err)
		time.Sleep(constants.RequestSleep * time.Second)
	}

	if i == constants.RequestRetries {
		log.Errorf("Failed to send notification to %s: request retries exhausted", httpReq.URL)
	} else {
		log.Infof("Successfully sent notification to %s", httpReq.URL)
	}
}
