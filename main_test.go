package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetupCacheDir(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := filepath.Join(os.TempDir(), "test-cache")
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Test setup cache directory
	err := setupCacheDir(tmpDir)
	if err != nil {
		t.Fatalf("setupCacheDir failed: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("Cache directory was not created: %v", err)
	}

	if !info.IsDir() {
		t.Fatal("Cache path is not a directory")
	}
}
