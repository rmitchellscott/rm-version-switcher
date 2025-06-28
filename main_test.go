package main

import (
	"os"
	"testing"
)

func TestPartitionInfo(t *testing.T) {
	p := PartitionInfo{
		Number:     2,
		Version:    "3.20.0.92",
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
	info := SystemInfo{
		Active: PartitionInfo{
			Number:     3,
			Version:    "3.20.0.92",
			IsActive:   true,
			IsNextBoot: true,
		},
		Fallback: PartitionInfo{
			Number:     2,
			Version:    "3.18.2.3",
			IsActive:   false,
			IsNextBoot: false,
		},
		NextBoot: 3,
	}

	if info.NextBoot != 3 {
		t.Errorf("Expected NextBoot to be 3, got %d", info.NextBoot)
	}
	if info.Active.Number != 3 {
		t.Errorf("Expected Active partition to be 3, got %d", info.Active.Number)
	}
	if info.Fallback.Number != 2 {
		t.Errorf("Expected Fallback partition to be 2, got %d", info.Fallback.Number)
	}
}

func TestGetDryRunSystemInfo(t *testing.T) {
	// Clean up any existing dry run file
	os.Remove("dry-run-boot.txt")

	info, err := getDryRunSystemInfo()
	if err != nil {
		t.Fatalf("getDryRunSystemInfo() failed: %v", err)
	}

	// Check default values
	if info.Active.Number != 3 {
		t.Errorf("Expected active partition to be 3, got %d", info.Active.Number)
	}
	if info.Fallback.Number != 2 {
		t.Errorf("Expected fallback partition to be 2, got %d", info.Fallback.Number)
	}
	if info.NextBoot != 3 {
		t.Errorf("Expected next boot to be 3, got %d", info.NextBoot)
	}
	if info.Active.Version != "3.20.0.92" {
		t.Errorf("Expected active version to be '3.20.0.92', got %s", info.Active.Version)
	}
	if info.Fallback.Version != "3.18.2.3" {
		t.Errorf("Expected fallback version to be '3.18.2.3', got %s", info.Fallback.Version)
	}

	// Test with dry run file
	err = os.WriteFile("dry-run-boot.txt", []byte("2"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test dry run file: %v", err)
	}
	defer os.Remove("dry-run-boot.txt")

	info, err = getDryRunSystemInfo()
	if err != nil {
		t.Fatalf("getDryRunSystemInfo() with file failed: %v", err)
	}

	if info.NextBoot != 2 {
		t.Errorf("Expected next boot to be 2 when reading from file, got %d", info.NextBoot)
	}
	if !info.Fallback.IsNextBoot {
		t.Errorf("Expected fallback partition to be next boot")
	}
	if info.Active.IsNextBoot {
		t.Errorf("Expected active partition to not be next boot")
	}
}

func TestSaveDryRunBootPartition(t *testing.T) {
	// Clean up any existing dry run file
	os.Remove("dry-run-boot.txt")
	defer os.Remove("dry-run-boot.txt")

	err := saveDryRunBootPartition(2)
	if err != nil {
		t.Fatalf("saveDryRunBootPartition() failed: %v", err)
	}

	// Verify file was created with correct content
	data, err := os.ReadFile("dry-run-boot.txt")
	if err != nil {
		t.Fatalf("Failed to read dry run file: %v", err)
	}

	if string(data) != "2" {
		t.Errorf("Expected file content to be '2', got %s", string(data))
	}

	// Test with partition 3
	err = saveDryRunBootPartition(3)
	if err != nil {
		t.Fatalf("saveDryRunBootPartition(3) failed: %v", err)
	}

	data, err = os.ReadFile("dry-run-boot.txt")
	if err != nil {
		t.Fatalf("Failed to read dry run file: %v", err)
	}

	if string(data) != "3" {
		t.Errorf("Expected file content to be '3', got %s", string(data))
	}
}

func TestBuildSystemInfoDisplay(t *testing.T) {
	info := &SystemInfo{
		Active: PartitionInfo{
			Number:     3,
			Version:    "3.20.0.92",
			IsActive:   true,
			IsNextBoot: true,
		},
		Fallback: PartitionInfo{
			Number:     2,
			Version:    "3.18.2.3",
			IsActive:   false,
			IsNextBoot: false,
		},
		NextBoot: 3,
	}

	display := buildSystemInfoDisplay(info)
	
	if display == "" {
		t.Error("buildSystemInfoDisplay() returned empty string")
	}

	// Check that it contains the title
	if !containsString(display, "reMarkable OS Version Switcher") {
		t.Error("Display should contain title")
	}

	// Check that it contains partition info
	if !containsString(display, "Partition  A:") {
		t.Error("Display should contain Partition A")
	}
	if !containsString(display, "Partition  B:") {
		t.Error("Display should contain Partition B")
	}

	// Check that it contains version numbers
	if !containsString(display, "3.20.0.92") {
		t.Error("Display should contain version 3.20.0.92")
	}
	if !containsString(display, "3.18.2.3") {
		t.Error("Display should contain version 3.18.2.3")
	}

	// Check that it contains status indicators
	if !containsString(display, "[ACTIVE]") {
		t.Error("Display should contain [ACTIVE] indicator")
	}
	if !containsString(display, "[NEXT BOOT]") {
		t.Error("Display should contain [NEXT BOOT] indicator")
	}
}

func TestLogToFile(t *testing.T) {
	// Clean up any existing debug file
	os.Remove("debug.log")
	defer os.Remove("debug.log")

	message := "Test log message"
	logToFile(message)

	// Check if file was created
	if _, err := os.Stat("debug.log"); os.IsNotExist(err) {
		t.Error("debug.log file was not created")
		return
	}

	// Read file content
	data, err := os.ReadFile("debug.log")
	if err != nil {
		t.Fatalf("Failed to read debug.log: %v", err)
	}

	content := string(data)
	if !containsString(content, message) {
		t.Errorf("Log file should contain message '%s', got: %s", message, content)
	}

	// Check timestamp format (should contain brackets and timestamp)
	if !containsString(content, "[") || !containsString(content, "]") {
		t.Error("Log should contain timestamp in brackets")
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDryRunBootPartitionPersistence(t *testing.T) {
	// Clean up
	os.Remove("dry-run-boot.txt")
	defer os.Remove("dry-run-boot.txt")

	// Save partition 2
	err := saveDryRunBootPartition(2)
	if err != nil {
		t.Fatalf("Failed to save partition 2: %v", err)
	}

	// Get system info should reflect the saved partition
	info, err := getDryRunSystemInfo()
	if err != nil {
		t.Fatalf("Failed to get dry run system info: %v", err)
	}

	if info.NextBoot != 2 {
		t.Errorf("Expected NextBoot to be 2, got %d", info.NextBoot)
	}
	if !info.Fallback.IsNextBoot {
		t.Error("Fallback partition should be marked as next boot")
	}
	if info.Active.IsNextBoot {
		t.Error("Active partition should not be marked as next boot")
	}

	// Save partition 3
	err = saveDryRunBootPartition(3)
	if err != nil {
		t.Fatalf("Failed to save partition 3: %v", err)
	}

	// Get system info should reflect the new saved partition
	info, err = getDryRunSystemInfo()
	if err != nil {
		t.Fatalf("Failed to get dry run system info after update: %v", err)
	}

	if info.NextBoot != 3 {
		t.Errorf("Expected NextBoot to be 3, got %d", info.NextBoot)
	}
	if info.Fallback.IsNextBoot {
		t.Error("Fallback partition should not be marked as next boot")
	}
	if !info.Active.IsNextBoot {
		t.Error("Active partition should be marked as next boot")
	}
}

func TestInvalidDryRunFile(t *testing.T) {
	// Clean up
	os.Remove("dry-run-boot.txt")
	defer os.Remove("dry-run-boot.txt")

	// Write invalid content
	err := os.WriteFile("dry-run-boot.txt", []byte("invalid"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid dry run file: %v", err)
	}

	// Should fall back to default (partition 3)
	info, err := getDryRunSystemInfo()
	if err != nil {
		t.Fatalf("getDryRunSystemInfo() with invalid file failed: %v", err)
	}

	if info.NextBoot != 3 {
		t.Errorf("Expected NextBoot to default to 3 with invalid file, got %d", info.NextBoot)
	}

	// Write out of range partition number
	err = os.WriteFile("dry-run-boot.txt", []byte("5"), 0644)
	if err != nil {
		t.Fatalf("Failed to write out of range dry run file: %v", err)
	}

	info, err = getDryRunSystemInfo()
	if err != nil {
		t.Fatalf("getDryRunSystemInfo() with out of range file failed: %v", err)
	}

	if info.NextBoot != 3 {
		t.Errorf("Expected NextBoot to default to 3 with out of range file, got %d", info.NextBoot)
	}
}