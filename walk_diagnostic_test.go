package filelock

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestWalkDiagnosticZeroInodeIsNotAssumedMissing(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "probe")
	if err := os.WriteFile(path, []byte("private-content"), 0644); err != nil {
		t.Fatal(err)
	}
	read := func(string, int) (map[string]rawEntry, error) {
		return map[string]rawEntry{"probe": {Name: "probe", Inode: 0, Offset: 128, Type: 8}}, nil
	}
	result := runRound(context.Background(), root, 8192, 1, 20, read)
	if result.ZeroInodes != 1 || result.MissingReadDir != 0 || result.MissingWalkDir != 0 || result.Errors != 0 || result.RawDigest != result.WalkDigest {
		t.Fatalf("zero inode incorrectly treated as absent: %+v", result)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "private-content" {
		t.Fatalf("diagnostic changed source: %v", err)
	}
}

func TestWalkDiagnosticSeparatesConcurrentDirectoryChanges(t *testing.T) {
	root := t.TempDir()
	calls := 0
	read := func(string, int) (map[string]rawEntry, error) {
		calls++
		if calls == 1 {
			return map[string]rawEntry{"removed": {Name: "removed", Inode: 1, Type: 8}}, nil
		}
		return map[string]rawEntry{}, nil
	}
	result := runRound(context.Background(), root, 8192, 1, 20, read)
	if result.UnstableDirectories != 1 || result.MissingReadDir != 0 || result.MissingReaddirnames != 0 || result.WalkCompared {
		t.Fatalf("concurrent removal classified as a standard-library omission: %+v", result)
	}
}

func TestWalkDiagnosticCancellationAndErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	read := func(string, int) (map[string]rawEntry, error) {
		called = true
		return nil, os.ErrPermission
	}
	result := runRound(ctx, t.TempDir(), 8192, 1, 20, read)
	if called || result.Errors != 1 {
		t.Fatalf("canceled diagnostic read directories: %+v", result)
	}
	result = runRound(context.Background(), t.TempDir(), 8192, 1, 20, read)
	if result.Errors != 1 || result.RawEntries != 0 {
		t.Fatalf("permission error hidden: %+v", result)
	}
}
