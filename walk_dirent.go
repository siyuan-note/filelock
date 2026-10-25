package filelock

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type rawEntry struct {
	Name   string `json:"name"`
	Inode  uint64 `json:"inode"`
	Offset int64  `json:"offset"`
	Type   uint8  `json:"type"`
	Batch  int    `json:"batch"`
}

// Linux/Android getdents64 returns native-endian linux_dirent64 records.
// Preserve zero inodes so that the probe does not repeat the stdlib filter.
func decodeDirents(data []byte, batch int) ([]rawEntry, error) {
	var entries []rawEntry
	for len(data) != 0 {
		if len(data) < 20 {
			return nil, fmt.Errorf("truncated dirent header: %d bytes", len(data))
		}
		length := int(binary.NativeEndian.Uint16(data[16:18]))
		if length < 20 || length > len(data) {
			return nil, fmt.Errorf("invalid dirent record length %d in %d bytes", length, len(data))
		}
		record := data[:length]
		end := bytes.IndexByte(record[19:], 0)
		if end < 0 {
			return nil, fmt.Errorf("dirent name has no terminator")
		}
		name := string(record[19 : 19+end])
		if name == "" || bytes.IndexByte([]byte(name), '/') >= 0 {
			return nil, fmt.Errorf("invalid dirent name %q", name)
		}
		if name != "." && name != ".." {
			entries = append(entries, rawEntry{
				Name: name, Inode: binary.NativeEndian.Uint64(record[:8]),
				Offset: int64(binary.NativeEndian.Uint64(record[8:16])), Type: record[18], Batch: batch,
			})
		}
		data = data[length:]
	}
	return entries, nil
}
