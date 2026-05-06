package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	MAMSURL   string
	ServiceID string
	Env       string
	BatchSize int
}

type LogEntry struct {
	Environment string `json:"environment"`
	Level       string `json:"level"`
	Message     string `json:"message"`
}

func main() {
	cfg := Config{
		BatchSize: 50,
	}
	flag.StringVar(&cfg.MAMSURL, "mams-url", os.Getenv("MAMS_URL"), "MAMS backend URL")
	flag.StringVar(&cfg.ServiceID, "service-id", os.Getenv("SERVICE_ID"), "Service ID")
	flag.StringVar(&cfg.Env, "env", "dev", "Environment")
	flag.Parse()

	if cfg.MAMSURL == "" || cfg.ServiceID == "" {
		fmt.Fprintf(os.Stderr, "MAMS_URL and SERVICE_ID are required\n")
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	batch := make([]LogEntry, 0, cfg.BatchSize)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		entry := parseLine(line, cfg.Env)
		batch = append(batch, entry)

		if len(batch) >= cfg.BatchSize {
			if err := sendBatch(cfg.MAMSURL, cfg.ServiceID, batch); err != nil {
				log.Printf("send batch failed: %v", err)
			}
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := sendBatch(cfg.MAMSURL, cfg.ServiceID, batch); err != nil {
			log.Printf("send final batch failed: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("stdin error: %v", err)
	}
}

func parseLine(line, env string) LogEntry {
	var msg struct {
		Level   string `json:"level"`
		Message string `json:"message"`
		Msg     string `json:"msg"`
	}

	if err := json.Unmarshal([]byte(line), &msg); err == nil {
		level := msg.Level
		if level == "" {
			level = "info"
		}
		message := msg.Message
		if message == "" {
			message = msg.Msg
		}
		if message == "" {
			message = line
		}
		return LogEntry{Environment: env, Level: level, Message: message}
	}

	return LogEntry{
		Environment: env,
		Level:       "info",
		Message:     line,
	}
}

func sendBatch(url, serviceID string, batch []LogEntry) error {
	payload, err := json.Marshal(batch)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/internal/services/%s/logs", url, serviceID), strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}