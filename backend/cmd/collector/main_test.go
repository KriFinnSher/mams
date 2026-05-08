package main

import "testing"

func TestParseLine_JSON(t *testing.T) {
	line := `{"level":"warn","message":"boom","environment":"staging","service_id":"svc-2"}`
	got := parseLine(line, "svc-1", "dev")
	if got.Level != "warn" || got.Message != "boom" {
		t.Fatalf("unexpected parsed log: %+v", got)
	}
	if got.Environment != "staging" || got.ServiceID != "svc-2" {
		t.Fatalf("unexpected mapping: %+v", got)
	}
}

func TestParseLine_JSONFallbacks(t *testing.T) {
	line := `{"msg":"ok"}`
	got := parseLine(line, "svc-1", "dev")
	if got.Level != "info" || got.Message != "ok" {
		t.Fatalf("unexpected parsed log: %+v", got)
	}
	if got.Environment != "dev" || got.ServiceID != "svc-1" {
		t.Fatalf("unexpected mapping: %+v", got)
	}
}

func TestParseLine_Raw(t *testing.T) {
	got := parseLine("plain text", "svc-1", "dev")
	if got.Level != "info" || got.Message != "plain text" {
		t.Fatalf("unexpected parsed log: %+v", got)
	}
	if got.Environment != "dev" || got.ServiceID != "svc-1" {
		t.Fatalf("unexpected mapping: %+v", got)
	}
}
