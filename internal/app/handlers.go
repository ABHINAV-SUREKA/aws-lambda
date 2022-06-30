package app

import (
	"encoding/json"
)

type payload struct {
	Payload Message `json:"payload"`
}

type Message map[string]interface{}

func (config *config) Handler() (string, error) {
	alertChan := make(chan int, 1)

	payload := formatEvent(config.event)
	payloadByteArr, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}

	go notifySlack(payloadByteArr, alertChan)
	go notifyPagerDuty(payloadByteArr, alertChan)

	for i := 0; i < 2; i++ {
		<-alertChan
	}

	return string(payloadByteArr), nil
}
