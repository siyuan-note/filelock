//go:build linux || android

package filelock

import (
	"fmt"
	"os"
	"syscall"
)

const walkDiagnosticRawSupported = true

func rawReadDir(path string, size int) (map[string]rawEntry, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.Close(fd)
	result := make(map[string]rawEntry)
	buffer := make([]byte, size)
	for batch := 0; ; batch++ {
		n, err := syscall.Getdents(fd, buffer)
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return result, nil
		}
		if n < 0 || n > len(buffer) {
			return nil, fmt.Errorf("invalid getdents length %d", n)
		}
		entries, err := decodeDirents(buffer[:n], batch)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if len(result) >= 100000 {
				return nil, fmt.Errorf("raw directory exceeds limit 100000")
			}
			if _, exists := result[entry.Name]; exists {
				return nil, fmt.Errorf("duplicate dirent %q", entry.Name)
			}
			result[entry.Name] = entry
		}
	}
}

func walkDiagnosticInode(info os.FileInfo) uint64 {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return stat.Ino
	}
	return 0
}

func walkDiagnosticPlatform(root string) map[string]any {
	result := map[string]any{}
	var info syscall.Statfs_t
	if err := syscall.Statfs(root, &info); err == nil {
		result["fsType"] = fmt.Sprintf("0x%x", info.Type)
	}
	if namespace, err := os.Readlink("/proc/self/ns/mnt"); err == nil {
		result["mountNamespace"] = namespace
	}
	return result
}
