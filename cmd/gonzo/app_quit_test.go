package main

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// blockingReader simulates a live pipe (e.g. `kubectl logs -f ... | gonzo`)
// whose writer never closes, so Read blocks indefinitely until the reader is
// closed. This is the exact condition behind issue #58: the stdin-reading
// goroutine stays blocked on read(2), holding the terminal so control is not
// returned to the parent (K9s) until the user hits Ctrl-C.
type blockingReader struct {
	closed chan struct{}
	once   sync.Once
}

func newBlockingReader() *blockingReader {
	return &blockingReader{closed: make(chan struct{})}
}

func (b *blockingReader) Read(p []byte) (int, error) {
	// Block until Close is called, then report EOF like a closed pipe.
	<-b.closed
	return 0, io.EOF
}

func (b *blockingReader) Close() error {
	b.once.Do(func() { close(b.closed) })
	return nil
}

// TestQuitKeyTriggersInputTeardown is the regression test for issue #58.
// Pressing "q" must (1) cancel the app context and (2) stop the stdin reader
// so that a goroutine blocked on a live pipe unblocks immediately, returning
// control to the parent process without requiring Ctrl-C.
func TestQuitKeyTriggersInputTeardown(t *testing.T) {
	for _, key := range []string{"q", "ctrl+c"} {
		t.Run(key, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			reader := newBlockingReader()
			m := &simpleTuiModel{
				dashboard:    nil, // not exercised: quit handling happens before dashboard dispatch
				ctx:          ctx,
				cancelFunc:   cancel,
				stdin:        reader,
				hasStdinData: true,
				inputChan:    make(chan string, 1),
			}

			// Launch the stdin reader; it will block on reader.Read.
			done := make(chan struct{})
			go func() {
				m.readStdinAsync()
				close(done)
			}()

			// The reader must currently be blocked (no EOF yet).
			select {
			case <-done:
				t.Fatal("readStdinAsync returned before quit was requested")
			case <-time.After(50 * time.Millisecond):
			}

			// Simulate the quit key. handleQuitKey is the cancellation seam:
			// it must cancel the context and stop the input reader.
			if !m.handleQuitKey(tea.KeyMsg{Type: keyType(key), Runes: keyRunes(key)}) {
				t.Fatalf("handleQuitKey(%q) = false, want true (key should be recognized as quit)", key)
			}

			// Context must be cancelled.
			select {
			case <-ctx.Done():
			default:
				t.Errorf("quit key %q did not cancel app context", key)
			}

			// The blocked stdin goroutine must unblock and return promptly.
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatalf("quit key %q did not stop the stdin reader; goroutine still blocked", key)
			}
		})
	}
}

// keyType/keyRunes build a tea.KeyMsg matching what bubbletea produces for the
// given logical key, so the test exercises the same msg.String() path as the app.
func keyType(key string) tea.KeyType {
	switch key {
	case "ctrl+c":
		return tea.KeyCtrlC
	default:
		return tea.KeyRunes
	}
}

func keyRunes(key string) []rune {
	switch key {
	case "ctrl+c":
		return nil
	default:
		return []rune(key)
	}
}
