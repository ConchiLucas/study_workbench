package main

import (
	"errors"
	"testing"
)

func TestRunStopsBeforeServerWhenInitializationFails(t *testing.T) {
	want := errors.New("migration failed")
	serverStarted := false

	err := run(
		func() error { return want },
		func() { serverStarted = true },
	)

	if !errors.Is(err, want) {
		t.Fatalf("expected initialization error, got %v", err)
	}
	if serverStarted {
		t.Fatal("server must not start after migration failure")
	}
}
