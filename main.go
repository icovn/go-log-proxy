package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	logFile            *os.File
	batchLock          sync.Mutex
	batch              []LokiStream
	logMessageField    = DotEnvVariable("LOG_MESSAGE_FIELD")
	logTimestampField  = DotEnvVariable("LOG_TIMESTAMP_FIELD")
	logTimestampFormat = DotEnvVariable("LOG_TIMESTAMP_FORMAT")
	lokiUrl            = DotEnvVariable("LOKI_URL")
	lokiUsername       = DotEnvVariable("LOKI_USERNAME")
	lokiPassword       = DotEnvVariable("LOKI_PASSWORD")
)

type LokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

type LokiPayload struct {
	Streams []LokiStream `json:"streams"`
}

func main() {
	initLogger()
	go startHTTP(":" + DotEnvVariable("PORT_HTTP"))
	go startTCP(":" + DotEnvVariable("PORT_TCP"))
	go batchSender()

	select {}
}

func initLogger() {
	os.Mkdir("logs", 0755)
	var err error
	logFile, err = os.OpenFile("logs/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
}

func startHTTP(addr string) {
	http.HandleFunc("/log", httpHandler)
	log.Printf("HTTP server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}
func httpHandler(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	processMessage(data)
	w.Write([]byte("received"))
}

func startTCP(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("TCP error: %v", err)
	}
	defer ln.Close()

	log.Printf("TCP server listening on %s\n", addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}
		go handleTCPConnection(conn)
	}
}
func handleTCPConnection(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var data map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &data); err != nil {
			log.Println("Invalid JSON:", err)
			continue
		}
		processMessage(data)
	}
}

func processMessage(data map[string]interface{}) {
	jsonStr, _ := json.Marshal(data)
	log.Println("Received: ", string(jsonStr))

	// Minimize/compact the JSON
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, jsonStr); err != nil {
		// log.Println("Failed to compact JSON:", err)
		return
	}
	logFile.Write(compacted.Bytes())
	logFile.Write([]byte("\n"))

	flatMap := make(map[string]string)
	flattenJSON("", data, flatMap)
	labels := make(map[string]string)
	message := flatMap[logMessageField]
	timestamp, _ := time.Parse(logTimestampFormat, flatMap[logTimestampField])
	for key, value := range flatMap {
		if key == logTimestampField || key == logMessageField {
			continue
		}
		formattedKey := strings.Replace(key, "-", "_", -1)
		formattedKey = strings.Replace(formattedKey, ".", "_", -1)
		labels[formattedKey] = fmt.Sprintf("%v", value)
	}
	logMsg := LokiStream{
		Stream: labels,
		Values: [][]string{
			{
				fmt.Sprintf("%d", timestamp.UnixNano()),
				message,
			},
		},
	}

	batchLock.Lock()
	batch = append(batch, logMsg)
	batchLock.Unlock()
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

func batchSender() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		batchLock.Lock()
		if len(batch) == 0 {
			batchLock.Unlock()
			continue
		}
		toSend := batch
		batch = nil
		batchLock.Unlock()

		sendToLoki(toSend)
	}
}

func sendToLoki(streams []LokiStream) {
	if len(streams) == 0 {
		return
	}

	body := map[string]interface{}{
		"streams": streams,
	}

	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", lokiUrl, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(lokiUsername, lokiPassword)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Failed to send to Loki:", err)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		log.Printf("Loki error status %d, detail: %s\n", resp.StatusCode, string(bodyBytes))
	} else {
		log.Println("Sent batch to Loki:", len(streams), "messages")
	}
}
