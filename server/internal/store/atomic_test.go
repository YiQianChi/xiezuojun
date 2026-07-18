package store

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicAppendDoesNotRewrite(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "log.jsonl")

	for i := 0; i < 3; i++ {
		if err := AtomicAppend(p, []byte("line\n")); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}

	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	lines := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines++
		if scanner.Text() != "line" {
			t.Fatalf("line %d content = %q, want %q", lines, scanner.Text(), "line")
		}
	}
	if lines != 3 {
		t.Fatalf("lines = %d, want 3 (read-modify-write regression)", lines)
	}

	info, _ := os.Stat(p)
	if info.Size() != 15 {
		t.Fatalf("size = %d, want 15", info.Size())
	}
}
