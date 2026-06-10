package xenvutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
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

func TestListVersionDirsSupportsAnywordTemplate(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "flutter-3.27"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "flutter-oh-3.27"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := ListVersionDirs(filepath.Join(root, "flutter-{anyword}-{version}"))
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf("matched versions = %#v, want only 3.27", got)
	}
	if filepath.ToSlash(got["3.27"]) != filepath.ToSlash(filepath.Join(root, "flutter-oh-3.27")) {
		t.Fatalf("version 3.27 path = %q, want flutter-oh-3.27 path", got["3.27"])
	}
}

func TestListVersionDirsSupportsHyphenatedVersionWithAnywordTemplate(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{
		"azul-17.0.18",
		"azul-17.0.18-1",
		"graalvm-jdk-17.0.11",
	} {
		assert.Require(t, assert.NoErr(t, os.MkdirAll(filepath.Join(root, dir), 0o755)))
	}

	got, err := ListVersionDirs(filepath.Join(root, "{anyword}-{version}"))
	assert.Require(t, assert.NoErr(t, err))

	assert.Eq(t, filepath.ToSlash(filepath.Join(root, "azul-17.0.18")), filepath.ToSlash(got["17.0.18"]))
	assert.Eq(t, filepath.ToSlash(filepath.Join(root, "azul-17.0.18-1")), filepath.ToSlash(got["17.0.18-1"]))
	assert.Eq(t, filepath.ToSlash(filepath.Join(root, "graalvm-jdk-17.0.11")), filepath.ToSlash(got["17.0.11"]))
	if _, ok := got["1"]; ok {
		t.Fatalf("expected azul-17.0.18-1 not to be indexed as version 1, got %#v", got)
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
