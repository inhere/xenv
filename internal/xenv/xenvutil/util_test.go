package xenvutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListVersionDirsMatchesInstallTemplateStrictly(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "flutter-3.27"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "flutter-oh-3.27"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := ListVersionDirs(filepath.Join(root, "flutter-{version}"))
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf("matched versions = %#v, want only 3.27", got)
	}
	if _, ok := got["3.27"]; !ok {
		t.Fatalf("expected version 3.27 to be indexed, got %#v", got)
	}
	if filepath.ToSlash(got["3.27"]) != filepath.ToSlash(filepath.Join(root, "flutter-3.27")) {
		t.Fatalf("version 3.27 path = %q, want flutter-3.27 path", got["3.27"])
	}
}

func TestListVersionDirsWithoutVersionTemplateKeepsLegacyNumericScan(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "flutter-oh-3.27"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := ListVersionDirs(filepath.Join(root, "flutter"))
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := got["3.27"]; !ok {
		t.Fatalf("expected legacy numeric scan to index 3.27, got %#v", got)
	}
}
