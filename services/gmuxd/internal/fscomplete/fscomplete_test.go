package fscomplete

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCompleteRequiresPathLikeInput(t *testing.T) {
	_, err := Complete("repo", 10)
	if !errors.Is(err, ErrUnsupportedPath) {
		t.Fatalf("err = %v, want ErrUnsupportedPath", err)
	}
}

func TestCompleteReturnsDirectoriesOnly(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "apple"))
	mustMkdir(t, filepath.Join(dir, "apricot"))
	if err := os.WriteFile(filepath.Join(dir, "app.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Complete(filepath.Join(dir, "ap"), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %+v", len(got), got)
	}
	if got[0].Name != "apple" || got[1].Name != "apricot" {
		t.Fatalf("unexpected suggestions: %+v", got)
	}
}

func TestCompleteHidesDotDirsUnlessPrefixIsDot(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, ".secret"))
	mustMkdir(t, filepath.Join(dir, "src"))

	got, err := Complete(dir+string(os.PathSeparator), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "src" {
		t.Fatalf("hidden dirs should be suppressed by default: %+v", got)
	}

	got, err = Complete(dir+string(os.PathSeparator)+".", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != ".secret" {
		t.Fatalf("dot prefix should show dot dirs: %+v", got)
	}
}

func TestCompleteLimitsResults(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a1", "a2", "a3"} {
		mustMkdir(t, filepath.Join(dir, name))
	}

	got, err := Complete(filepath.Join(dir, "a"), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
}
