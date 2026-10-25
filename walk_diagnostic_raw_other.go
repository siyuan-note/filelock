//go:build !linux && !android

package filelock

import "os"

const walkDiagnosticRawSupported = false

func rawReadDir(string, int) (map[string]rawEntry, error) { return nil, errRawWalkUnsupported }
func walkDiagnosticInode(os.FileInfo) uint64              { return 0 }
func walkDiagnosticPlatform(string) map[string]any        { return nil }
