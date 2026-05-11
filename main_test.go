package main

import (
	"os"
	"strings"
	"testing"

	"github.com/rmitchellscott/remarkable-go/partition"
)

func TestPartitionInfo(t *testing.T) {
	p := partition.Info{
		Number:     2,
		Version:    "3.20.0.92",
		Label:      "A",
		IsActive:   true,
		IsNextBoot: false,
	}

	if p.Number != 2 {
		t.Errorf("Expected Number to be 2, got %d", p.Number)
	}
	if p.Version != "3.20.0.92" {
		t.Errorf("Expected Version to be '3.20.0.92', got %s", p.Version)
	}
	if !p.IsActive {
		t.Errorf("Expected IsActive to be true")
	}
	if p.IsNextBoot {
		t.Errorf("Expected IsNextBoot to be false")
	}
}

func TestSystemInfo(t *testing.T) {
	info := partition.SystemInfo{
		Active: partition.Info{
			Number:     3,
			Version:    "3.20.0.92",
			IsActive:   true,
			IsNextBoot: true,
		},
		Fallback: partition.Info{
			Number:     2,
			Version:    "3.18.2.3",
			IsActive:   false,
			IsNextBoot: false,
		},
	}

	if info.Active.Number != 3 {
		t.Errorf("Expected Active partition to be 3, got %d", info.Active.Number)
	}
	if info.Fallback.Number != 2 {
		t.Errorf("Expected Fallback partition to be 2, got %d", info.Fallback.Number)
	}
	if !info.Active.IsNextBoot {
		t.Errorf("Expected Active partition to be next boot")
	}
}

func TestLogToFile(t *testing.T) {
	os.Remove("debug.log")
	defer os.Remove("debug.log")

	message := "Test log message"
	logToFile(message)

	if _, err := os.Stat("debug.log"); os.IsNotExist(err) {
		t.Error("debug.log file was not created")
		return
	}

	data, err := os.ReadFile("debug.log")
	if err != nil {
		t.Fatalf("Failed to read debug.log: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, message) {
		t.Errorf("Log file should contain message '%s', got: %s", message, content)
	}

	if !strings.Contains(content, "[") || !strings.Contains(content, "]") {
		t.Error("Log should contain timestamp in brackets")
	}
}
