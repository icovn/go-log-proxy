package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type LokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

type LokiPayload struct {
	Streams []LokiStream `json:"streams"`
}

// flattenJSON recursively flattens a nested JSON object into map[string]string
func flattenJSON(prefix string, in map[string]interface{}, out map[string]string) {
	for k, v := range in {
		key := k
		if prefix != "" {
			key = prefix + "_" + k
		}

		switch value := v.(type) {
		case map[string]interface{}:
			flattenJSON(key, value, out)
		case string:
			out[key] = value
		case float64:
			out[key] = fmt.Sprintf("%v", value)
		case bool:
			out[key] = fmt.Sprintf("%t", value)
		default:
			out[key] = fmt.Sprintf("%v", value)
		}
	}
}

func SendToLoki(msg json.RawMessage) {
	// Step 1: Flatten the JSON message
	var raw map[string]interface{}
	json.Unmarshal([]byte(msg), &raw)

	flatMap := make(map[string]string)
	flattenJSON("", raw, flatMap)
	// for k, v := range flatMap {
	// 	fmt.Printf("%s: %s\n", k, v)
	// }

	// Step 2: Prepare the Loki payload
	labels := make(map[string]string)
	message := flatMap[DotEnvVariable("LOG_MESSAGE_FIELD")]
	timestamp, _ := time.Parse(DotEnvVariable("LOG_TIMESTAMP_FORMAT"), flatMap[DotEnvVariable("LOG_TIMESTAMP_FIELD")])
	for key, value := range flatMap {
		if key == DotEnvVariable("LOG_TIMESTAMP_FIELD") || key == DotEnvVariable("LOG_MESSAGE_FIELD") {
			// Skip timestamp and message field
			continue
		}
		labels[strings.Replace(key, "-", "_", -1)] = fmt.Sprintf("%v", value)
	}

	payload := LokiPayload{
		Streams: []LokiStream{
			{
				Stream: labels,
				Values: [][]string{
					{
						fmt.Sprintf("%d", timestamp.UnixNano()),
						message,
					},
				},
			},
		},
	}
	// fmt.Println("- Labels: ", labels)
	// fmt.Println("- Message: ", message)
	// fmt.Println("- Timestamp: ", timestamp.UnixNano())

	buf := new(bytes.Buffer)
	_ = json.NewEncoder(buf).Encode(payload)

	req, err := http.NewRequest("POST", DotEnvVariable("LOKI_URL"), buf)
	if err != nil {
		fmt.Println("Failed to create request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// Optional: Basic Auth
	req.SetBasicAuth(DotEnvVariable("LOKI_USERNAME"), DotEnvVariable("LOKI_PASSWORD"))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending to Loki:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("Loki returned non-200:" + resp.Status + ", body: " + string(body))
	}
}
