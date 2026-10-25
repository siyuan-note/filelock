package filelock

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type observation struct {
	Kind       string    `json:"kind"`
	Path       string    `json:"path"`
	Raw        *rawEntry `json:"raw,omitempty"`
	RawAfter   *rawEntry `json:"rawAfter,omitempty"`
	Error      string    `json:"error,omitempty"`
	StatInode  uint64    `json:"statInode,omitempty"`
	Size       int64     `json:"size,omitempty"`
	ModifiedNS int64     `json:"modifiedNS,omitempty"`
	Mode       string    `json:"mode,omitempty"`
}

type roundReport struct {
	Round               int           `json:"round"`
	Buffer              int           `json:"buffer"`
	Directories         int           `json:"directories"`
	RawEntries          int           `json:"rawEntries"`
	ZeroInodes          int           `json:"zeroInodes"`
	MissingReadDir      int           `json:"missingReadDir"`
	UnexpectedReadDir   int           `json:"unexpectedReadDir"`
	MissingReaddirnames int           `json:"missingReaddirnames"`
	MissingWalkDir      int           `json:"missingWalkDir"`
	UnexpectedWalkDir   int           `json:"unexpectedWalkDir"`
	UnstableDirectories int           `json:"unstableDirectories"`
	Errors              int           `json:"errors"`
	RawDigest           string        `json:"rawDigest"`
	WalkDigest          string        `json:"walkDigest"`
	WalkCompared        bool          `json:"walkCompared"`
	ElapsedMS           int64         `json:"elapsedMS"`
	Details             []observation `json:"details,omitempty"`
}

func (r *roundReport) add(detail observation) {
	if len(r.Details) < 100 {
		r.Details = append(r.Details, detail)
	}
}

func (r *roundReport) fail(kind, path string, err error) {
	r.Errors++
	r.add(observation{Kind: kind, Path: path, Error: err.Error()})
}

func namesStable(a, b map[string]rawEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for name := range a {
		if _, exists := b[name]; !exists {
			return false
		}
	}
	return true
}

func digest(paths map[string]bool) string {
	names := make([]string, 0, len(paths))
	for name := range paths {
		names = append(names, name)
	}
	sort.Strings(names)
	hash := sha256.New()
	for _, name := range names {
		fmt.Fprintf(hash, "%s\x00", name)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func describe(path string, entry, after rawEntry, kind string) observation {
	result := observation{Kind: kind, Path: path, Raw: &entry, RawAfter: &after}
	info, err := os.Lstat(path)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Size, result.ModifiedNS, result.Mode = info.Size(), info.ModTime().UnixNano(), info.Mode().String()
	result.StatInode = walkDiagnosticInode(info)
	return result
}

func runRound(ctx context.Context, root string, buffer, number, maxDirs int, readRaw func(string, int) (map[string]rawEntry, error)) (r roundReport) {
	start := time.Now()
	r = roundReport{Round: number, Buffer: buffer}
	defer func() { r.ElapsedMS = time.Since(start).Milliseconds() }()
	queue := []string{root}
	rawPaths := map[string]bool{}
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			r.fail("canceled", ".", err)
			return r
		}
		dir := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if r.Directories >= maxDirs {
			r.fail("directory-limit", dir, fmt.Errorf("limit %d exceeded", maxDirs))
			break
		}
		r.Directories++
		before, err := readRaw(dir, buffer)
		if err != nil {
			r.fail("raw-before", dir, err)
			continue
		}
		standard, stdErr := os.ReadDir(dir)
		if stdErr != nil {
			r.fail("ReadDir", dir, stdErr)
		}
		standardNames := map[string]bool{}
		for _, item := range standard {
			standardNames[item.Name()] = true
		}
		names := map[string]bool{}
		namesOK := false
		file, openErr := os.Open(dir)
		if openErr != nil {
			r.fail("Readdirnames-open", dir, openErr)
		} else {
			readNames, namesErr := file.Readdirnames(-1)
			file.Close()
			if namesErr != nil {
				r.fail("Readdirnames", dir, namesErr)
			}
			namesOK = namesErr == nil
			for _, name := range readNames {
				names[name] = true
			}
		}
		after, err := readRaw(dir, buffer)
		if err != nil {
			r.fail("raw-after", dir, err)
			continue
		}
		stable := namesStable(before, after)
		if !stable {
			r.UnstableDirectories++
			r.add(observation{Kind: "directory-changed-during-comparison", Path: dir})
		}
		for name, entry := range before {
			if err := ctx.Err(); err != nil {
				r.fail("canceled", ".", err)
				return r
			}
			if r.RawEntries >= 100000 {
				r.fail("entry-limit", ".", fmt.Errorf("limit 100000 exceeded"))
				return r
			}
			abs := filepath.Join(dir, name)
			rawPaths[abs] = true
			r.RawEntries++
			if entry.Inode == 0 {
				r.ZeroInodes++
				r.add(describe(abs, entry, after[name], "zero-inode"))
			}
			if stable && stdErr == nil && !standardNames[name] {
				r.MissingReadDir++
				r.add(describe(abs, entry, after[name], "missing-ReadDir"))
			}
			if stable && namesOK && !names[name] {
				r.MissingReaddirnames++
				r.add(describe(abs, entry, after[name], "missing-Readdirnames"))
			}
			info, statErr := os.Lstat(abs)
			if statErr != nil {
				r.fail("Lstat", abs, statErr)
			} else if info.IsDir() {
				queue = append(queue, abs)
			}
		}
		if stable && stdErr == nil {
			for name := range standardNames {
				if _, exists := before[name]; !exists {
					r.UnexpectedReadDir++
					r.add(observation{Kind: "ReadDir-only", Path: filepath.Join(dir, name)})
				}
			}
		}
	}
	r.RawDigest = digest(rawPaths)
	if r.Errors != 0 || r.UnstableDirectories != 0 {
		return r
	}
	walkPaths := map[string]bool{}
	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if canceled := ctx.Err(); canceled != nil {
			return canceled
		}
		if len(walkPaths) >= 100000 {
			return fmt.Errorf("entry limit 100000 exceeded")
		}
		if err != nil {
			r.fail("WalkDir-entry", path, err)
			return nil
		}
		if path != root {
			walkPaths[path] = true
		}
		return nil
	})
	if walkErr != nil {
		r.fail("WalkDir", root, walkErr)
	}
	if r.Errors != 0 {
		return r
	}
	r.WalkCompared = true
	for path := range rawPaths {
		if !walkPaths[path] {
			r.MissingWalkDir++
			r.add(observation{Kind: "missing-WalkDir", Path: path})
		}
	}
	for path := range walkPaths {
		if !rawPaths[path] {
			r.UnexpectedWalkDir++
			r.add(observation{Kind: "WalkDir-only", Path: path})
		}
	}
	r.RawDigest, r.WalkDigest = digest(rawPaths), digest(walkPaths)
	r.ElapsedMS = time.Since(start).Milliseconds()
	return r
}

// DiagnoseWalk compares native traversal with raw directory entries in the calling
// process. It reads only metadata, never writes the source tree, and bounds each
// of three rounds to 20000 directories, 100000 entries, and 100 detail records.
// The JSON-lines report contains relative paths and no document content.
// Raw comparison is available on Android/Linux; other platforms report that it
// is unsupported rather than claiming a successful comparison.
func DiagnoseWalk(ctx context.Context, root string) (string, error) {
	root = filepath.Clean(root)
	info, err := os.Lstat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || !filepath.IsAbs(root) {
		return "", fmt.Errorf("diagnostic root must be an absolute directory")
	}
	var output strings.Builder
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	environment := map[string]any{
		"type": "environment", "scanID": walkSequence.Add(1), "go": runtime.Version(),
		"goos": runtime.GOOS, "goarch": runtime.GOARCH, "uid": os.Getuid(), "pid": os.Getpid(),
		"time": time.Now().UTC().Format(time.RFC3339), "contentRead": false,
		"rawSupported": walkDiagnosticRawSupported,
	}
	for key, value := range walkDiagnosticPlatform(root) {
		environment[key] = value
	}
	if err = encoder.Encode(environment); err != nil {
		return "", err
	}
	if !walkDiagnosticRawSupported {
		return output.String(), nil
	}
	for i, buffer := range []int{4096, 8192, 32768} {
		if err = ctx.Err(); err != nil {
			return output.String(), err
		}
		report := runRound(ctx, root, buffer, i+1, 20000, rawReadDir)
		for j := range report.Details {
			detail := &report.Details[j]
			if filepath.IsAbs(detail.Path) {
				if relative, relErr := filepath.Rel(root, detail.Path); relErr == nil {
					detail.Path = filepath.ToSlash(relative)
				}
			}
			detail.Error = strings.ReplaceAll(detail.Error, root, "<root>")
		}
		if err = encoder.Encode(report); err != nil {
			return output.String(), err
		}
	}
	return output.String(), ctx.Err()
}

var errRawWalkUnsupported = errors.New("raw directory diagnostics unsupported on this platform")
