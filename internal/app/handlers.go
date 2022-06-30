package app

import (
	"encoding/json"
)

type payload struct {
	Payload Message `json:"payload"`
}

type Message map[string]interface{}

func (config *config) Handler() (string, error) {
	payload := formatEvent(config.event)
	eventJSON, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}

	return string(eventJSON), nil
}
