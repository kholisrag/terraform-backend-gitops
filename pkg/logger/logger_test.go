package logger

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestInfo(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	logger.Info("info level test")

	if len(recorded.All()) != 1 {
		t.Errorf("Logger did not log the expected number of messages: 1")
	}

	if recorded.All()[0].Message != "info level test" {
		t.Errorf("Logger did not log the expected message: 'test'")
	}
}

func TestDebug(t *testing.T) {
	core, recorded := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	logger.Debug("debug level test")

	if len(recorded.All()) != 1 {
		t.Errorf("Logger did not log the expected number of messages: 1")
	}

	if recorded.All()[0].Message != "debug level test" {
		t.Errorf("Logger did not log the expected message: 'test'")
	}
}
