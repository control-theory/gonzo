package filereader

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFollowStopWhileOutputIsFull(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	if err := os.WriteFile(path, []byte("initial\n"), 0600); err != nil {
		t.Fatal(err)
	}

	reader, err := New([]string{path}, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Stop)
	lines := reader.Start()
	select {
	case line := <-lines:
		if line != "initial" {
			t.Fatalf("initial line = %q, want initial", line)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("initial line was not delivered")
	}

	var watchedFile *os.File
	waitFor(t, "file watch to be registered", func() bool {
		reader.mu.Lock()
		defer reader.mu.Unlock()
		watcher := reader.watchers[path]
		if watcher == nil || len(watcher.WatchList()) == 0 {
			return false
		}
		watchedFile = reader.fileStates[path].file
		return true
	})

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.WriteString(strings.Repeat("appended\n", 2*cap(lines)))
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "appended lines to fill the output channel", func() bool {
		return len(lines) == cap(lines)
	})

	reader.Stop()
	done := make(chan struct{})
	go func() {
		reader.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("follow mode did not stop with a full output channel")
	}

	for range cap(lines) {
		if line := <-lines; line != "appended" {
			t.Fatalf("buffered line = %q, want appended", line)
		}
	}
	select {
	case _, open := <-lines:
		if open {
			t.Fatal("reader delivered a line after shutdown")
		}
	default:
		t.Fatal("reader did not close the output channel")
	}
	if _, err := watchedFile.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("followed file stat error = %v, want closed file", err)
	}
}

// Helpers

func waitFor(t *testing.T, description string, ready func() bool) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for !ready() {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %s", description)
		case <-tick.C:
		}
	}
}

// End helpers
