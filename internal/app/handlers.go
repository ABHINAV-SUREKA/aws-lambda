package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type payload struct {
	Payload Message `json:"payload"`
}

type Message map[string]interface{}

type httpRequest struct {
	url     string
	headers map[string]string
	method  string
	reqBody []byte
}

func (config *config) Handler() (string, error) {
	alertChan := make(chan struct{}, 1)

	payload := formatEvent(config.event)
	payloadByteArr, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}

	for i := 0; i < 2; i++ {
		<-alertChan
	}

	return string(payloadByteArr), nil
}

func (config *config) SendNotification(httpReq httpRequest, alertChan chan struct{}) {
	for i := 0; i < requestRetries; i++ {
		if err := func() error {
			req, err := http.NewRequest(httpReq.method, httpReq.url, bytes.NewBuffer(httpReq.reqBody))
			if err != nil {
				return errors.New(fmt.Sprintf("error notifying Slack: %s. Retrying...", err))
			}
			for key, val := range httpReq.headers {
				req.Header.Add(key, val)
			}
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return errors.New(fmt.Sprintf("error sending notification: %s. Retrying...", err))
			}
			defer resp.Body.Close()
			if statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300; !statusOK {
				return errors.New(fmt.Sprintf("error sending notification: non-OK HTTP status: %v. Retrying...", resp.StatusCode))
			}
			return nil
		}(); err != nil {
			log.Error(err)
			time.Sleep(requestSleep * time.Second)
			continue
		}
		break
	}
	alertChan <- struct{}{}
}
