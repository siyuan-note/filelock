package filelock

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func wireEntry(name string, inode uint64, offset int64, kind byte) []byte {
	size := (19 + len(name) + 1 + 7) &^ 7
	data := make([]byte, size)
	binary.NativeEndian.PutUint64(data[:8], inode)
	binary.NativeEndian.PutUint64(data[8:16], uint64(offset))
	binary.NativeEndian.PutUint16(data[16:18], uint16(size))
	data[18] = kind
	copy(data[19:], name)
	return data
}

func TestDecodePreservesZeroInodesAndBatchBoundaries(t *testing.T) {
	first := append(wireEntry(".", 1, 1, 4), wireEntry("..", 2, 2, 4)...)
	first = append(first, wireEntry("zero.txt", 0, 17, 8)...)
	a, err := decodeDirents(first, 0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := decodeDirents(wireEntry("directory", 33, 9000000001, 4), 1)
	if err != nil {
		t.Fatal(err)
	}
	got := append(a, b...)
	want := []rawEntry{{Name: "zero.txt", Inode: 0, Offset: 17, Type: 8, Batch: 0}, {Name: "directory", Inode: 33, Offset: 9000000001, Type: 4, Batch: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decoded %v, want %v", got, want)
	}
}

func TestDecodeRejectsMalformedRecords(t *testing.T) {
	zeroLength := wireEntry("probe", 1, 1, 8)
	binary.NativeEndian.PutUint16(zeroLength[16:18], 0)
	oversized := wireEntry("probe", 1, 1, 8)
	binary.NativeEndian.PutUint16(oversized[16:18], uint16(len(oversized)+8))
	unterminated := wireEntry("probe", 1, 1, 8)
	for i := 19; i < len(unterminated); i++ {
		unterminated[i] = 'x'
	}
	for name, data := range map[string][]byte{
		"short-header": {0, 1}, "zero-length": zeroLength, "oversized": oversized,
		"unterminated": unterminated, "empty-name": wireEntry("", 1, 1, 8), "slash": wireEntry("a/b", 1, 1, 8),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeDirents(data, 0); err == nil {
				t.Fatal("accepted malformed dirent")
			}
		})
	}
}
