package logger

import (
	"bytes"
	"encoding/json"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestJSONLogger(t *testing.T) {
	var output bytes.Buffer
	log, err := newWithSink(Config{JSON: true, Level: "info"}, zapcore.AddSync(&output), zapcore.AddSync(&bytes.Buffer{}))
	if err != nil {
		t.Fatalf("newWithSink() error = %v", err)
	}
	log.Info("ready", zap.String("component", "test"))
	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("decode log: %v", err)
	}
	if entry["message"] != "ready" || entry["component"] != "test" {
		t.Fatalf("unexpected entry: %#v", entry)
	}
}

func TestLoggerRejectsInvalidLevel(t *testing.T) {
	if _, err := newWithSink(Config{Level: "verbose"}, zapcore.AddSync(&bytes.Buffer{}), zapcore.AddSync(&bytes.Buffer{})); err == nil {
		t.Fatal("expected an error")
	}
}
