package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"
)

var (
	logFile *os.File
)

func init() {
	os.Mkdir("logs", 0755)
	var err error
	logFile, err = os.OpenFile("logs/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
}

func logAndWrite(msg json.RawMessage) {
	// Minimize/compact the JSON
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, msg); err != nil {
		// log.Println("Failed to compact JSON:", err)
		return
	}

	log.Println(compacted.String())
	logFile.Write(compacted.Bytes())
	logFile.Write([]byte("\n"))

	// Send to Loki
	SendToLoki(msg)
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var msg json.RawMessage
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	logAndWrite(msg)
	w.WriteHeader(http.StatusOK)
}

func tcpListener() {
	ln, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("TCP accept error:", err)
			continue
		}
		go handleTCP(conn)
	}
}

func handleTCP(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				log.Println("TCP read error:", err)
			}
			break
		}
		var msg json.RawMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			log.Println("Invalid JSON:", err)
			continue
		}
		logAndWrite(msg)
	}
}

func main() {
	defer logFile.Close()
	go tcpListener()

	http.HandleFunc("/log", httpHandler)
	log.Println("HTTP listening on :8080 and TCP on :9000")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
