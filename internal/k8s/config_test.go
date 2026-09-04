package k8s

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeKubeconfig writes a minimal kubeconfig holding a single context.
func writeKubeconfig(t *testing.T, path, context string) string {
	t.Helper()
	content := "apiVersion: v1\nkind: Config\nclusters:\n- name: " + context +
		"\n  cluster:\n    server: https://example.invalid\ncontexts:\n- name: " + context +
		"\n  context:\n    cluster: " + context + "\n    user: " + context +
		"\nusers:\n- name: " + context + "\n  user: {}\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDetectKubeconfig(t *testing.T) {
	dir := t.TempDir()
	first := writeKubeconfig(t, filepath.Join(dir, "config"), "first")
	second := writeKubeconfig(t, filepath.Join(dir, "staging-config"), "second")
	missing := filepath.Join(dir, "missing")

	empty := filepath.Join(dir, "empty-config")
	if err := os.WriteFile(empty, []byte("apiVersion: v1\nkind: Config\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	sep := string(filepath.ListSeparator)

	tests := []struct {
		name       string
		kubeconfig string
		want       []string
	}{
		{"single file", first, []string{first}},
		{"single missing file", missing, nil},
		{"file without context", empty, nil},
		{"multiple files", first + sep + second, []string{first, second}},
		{"multiple files with one missing", missing + sep + second, []string{missing, second}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("KUBECONFIG", tt.kubeconfig)
			got := DetectKubeconfig()
			if strings.Join(got, sep) != strings.Join(tt.want, sep) {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
