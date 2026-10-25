package filelock

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestWalkNativeSemantics(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"skip/hidden", "skip-extra/keep", "nested/leaf", "z.txt"} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, mode := range []string{"all", "skip-dir", "skip-all", "error"} {
		t.Run(mode, func(t *testing.T) {
			refused := errors.New("refused")
			collect := func(walk func(string, fs.WalkDirFunc) error) ([]string, error) {
				var paths []string
				err := walk(root, func(path string, entry fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					rel, _ := filepath.Rel(root, path)
					paths = append(paths, rel)
					if entry.Name() == "skip" {
						switch mode {
						case "skip-dir":
							return fs.SkipDir
						case "skip-all":
							return fs.SkipAll
						case "error":
							return refused
						}
					}
					return nil
				})
				return paths, err
			}
			got, gotErr := collect(Walk)
			want, wantErr := collect(filepath.WalkDir)
			if !reflect.DeepEqual(got, want) || !errors.Is(gotErr, wantErr) {
				t.Fatalf("walk differs: %v %v; want %v %v", got, gotErr, want, wantErr)
			}
		})
	}
}

func TestWalkPreservesNativeMetadataAndErrors(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "probe")
	if err := os.WriteFile(path, []byte("probe"), 0644); err != nil {
		t.Fatal(err)
	}
	stamp := time.Unix(1700000100, 123456700)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	want, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	if err = Walk(root, func(current string, entry fs.DirEntry, err error) error {
		if err != nil || current != path {
			return err
		}
		found = true
		info, err := entry.Info()
		if err != nil || info.Size() != want.Size() || !info.ModTime().Equal(want.ModTime()) {
			t.Fatalf("native metadata changed: %v %v", info, err)
		}
		return nil
	}); err != nil || !found {
		t.Fatalf("native file not visited: %v", err)
	}
	if err = Walk(filepath.Join(root, "missing"), func(_ string, _ fs.DirEntry, err error) error { return err }); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing-root error hidden: %v", err)
	}
}
