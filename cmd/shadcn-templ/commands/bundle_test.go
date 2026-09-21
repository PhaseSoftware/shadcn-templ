package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchScripts(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builds := make(chan struct{}, 20)
	done := make(chan error, 1)
	go func() { done <- watchScripts(ctx, dir, func() error { builds <- struct{}{}; return nil }) }()
	await := func() {
		t.Helper()
		select {
		case <-builds:
		case err := <-done:
			t.Fatalf("watch stopped: %v", err)
		case <-time.After(3 * time.Second):
			t.Fatal("watch did not rebuild")
		}
	}
	await()
	child := filepath.Join(dir, "example")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(child, "example.js")
	if err := os.WriteFile(file, []byte("first"), 0644); err != nil {
		t.Fatal(err)
	}
	await()
	if err := os.WriteFile(file, []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	await()
	if err := os.Rename(file, filepath.Join(child, "renamed.js")); err != nil {
		t.Fatal(err)
	}
	await()
	if err := os.RemoveAll(child); err != nil {
		t.Fatal(err)
	}
	await()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("watch did not stop")
	}
}
