package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

func main() {
	eventMsg := "routing_key: d24d67d821fd4904c0eb2a7399c15ed4\nseverity: error\ndescription: \"CPU Core count > 3\" # description of the alert\ndetails: {\"CPU Core count\": \"greater than 3\"}"

	data := make(map[string]interface{})

	_ = yaml.Unmarshal([]byte(eventMsg), &data)

	for k, v := range data {
		if k == "details" {
			details := ""
			for k, v := range v.(map[string]interface{}) {
				details = details + fmt.Sprintf("%s: %v\n", k, v)
			}
			fmt.Println(details)
		} else {
			fmt.Printf("%s -> %v\n", k, v)
		}
	}
}
