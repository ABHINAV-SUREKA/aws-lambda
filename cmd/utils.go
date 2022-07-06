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
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"strings"
	"time"
)

type HttpRequest struct {
	URL     string
	Headers map[string]interface{}
	Method  string
	Body    []byte
}

func formatEventMessage(event events.SNSEvent) {

	httpReq, err := func() (*HttpRequest, error) {
		httpReq := HttpRequest{}
		headers := make(map[string]interface{})

		if len(event.Records) > 0 {
			eventMsg := event.Records[0].SNS.Message
			eventSub := event.Records[0].SNS.Subject
			log.Info(eventMsg)
			log.Info(eventSub)

			data := make(map[string]interface{})
			err := yaml.Unmarshal([]byte(eventMsg), &data)
			if err != nil {
				return nil, err
			}

			if strings.Contains(strings.ToLower(eventMsg), "routing_key") {
				/* Format event message for PagerDuty
				 */
				payload := make(map[string]interface{})

				for key, val := range data {
					log.Infof("%s:%v", key, val)
					switch {
					case strings.Contains(strings.ToLower(key), "client_url"):
						payload["source"] = val
					case strings.Contains(strings.ToLower(key), "severity"):
						payload[key] = val
						delete(data, "severity")
					case strings.Contains(strings.ToLower(key), "description"):
						payload["summary"] = val
						delete(data, "description")
					case strings.Contains(strings.ToLower(key), "details"):
						details := ""
						for k, v := range val.(map[string]interface{}) {
							details = details + fmt.Sprintf("%s: %v\n", k, v)
						}
						payload["custom_details"] = details
						delete(data, "details")
					case strings.Contains(strings.ToLower(key), "routing_key"):
						headers["x-routing-key"] = val
						delete(data, "routing_key")
					}
				}

				if strings.Contains(strings.ToLower(eventSub), "resolve") {
					data["event_action"] = "resolve"
				} else {
					data["event_action"] = "trigger"
				}

				data["payload"] = payload

				byteArr, err := json.MarshalIndent(data, "", "  ")
				if err != nil {
					return nil, err
				}

				httpReq.Body = byteArr
				httpReq.URL = constants.PagerDutyURL
				httpReq.Method = "POST"
				headers["Content-Type"] = "application/json"
				httpReq.Headers = headers
				return &httpReq, nil
			}

			/* Format event message for Slack
			 */
			byteArr, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				return nil, err
			}

			httpReq.Body = byteArr
			httpReq.URL = constants.SlackURL
			httpReq.Method = "POST"
			headers["Content-Type"] = "application/json"
			httpReq.Headers = headers
			return &httpReq, nil
		}

		return nil, errors.New("no SNS event records found")
	}()

	if err != nil {
		log.Errorf("Error formatting SNS event message: %s", err)
	} else {
		sendNotification(*httpReq)
	}
}

func sendNotification(httpReq HttpRequest) {
	var (
		i       = 0
		err     error
		req     *http.Request
		resp    *http.Response
		client  = &http.Client{}
		byteArr []byte
	)
	log.Info("string(httpReq.Body): ", string(httpReq.Body))

	for ; i < constants.RequestRetries; i++ {
		if err = func() error {
			req, err = http.NewRequest(httpReq.Method, httpReq.URL, bytes.NewBuffer(httpReq.Body))
			if err != nil {
				return errors.New(fmt.Sprintf("Error sending notification to %s: %s. Retrying...", httpReq.URL, err))
			}
			for key, val := range httpReq.Headers {
				req.Header.Add(key, val.(string))
			}

			ctx, cancel := context.WithTimeout(req.Context(), constants.RequestTimeout*time.Second)
			defer cancel()
			req = req.WithContext(ctx)

			resp, err = client.Do(req)
			if err != nil {
				return errors.New(fmt.Sprintf("Error sending notification to %s: %s. Retrying...", httpReq.URL, err))
			}
			defer resp.Body.Close()

			byteArr, err = io.ReadAll(resp.Body)
			if err != nil {
				return errors.New(fmt.Sprintf("Error reading response: %v. Retrying...", err))
			}
			if statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300; !statusOK {
				return errors.New(fmt.Sprintf("Error sending notification to %s: non-OK HTTP status: %v. Error: %v. Retrying...", httpReq.URL, string(byteArr), resp.StatusCode))
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
