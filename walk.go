// FileLock - Read and write files with lock.
// Copyright (c) 2022-present, b3log.org
//
// FileLock is licensed under Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//         http://license.coscl.org.cn/MulanPSL2
//
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
// EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
// MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
//
// See the Mulan PSL v2 for more details.

package filelock

import (
	"io/fs"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/siyuan-note/logging"
)

var walkSequence atomic.Uint64

// Walk follows filepath.WalkDir semantics on every platform, including Android.
// FileInfo comes from the native filesystem, with its original time precision.
func Walk(root string, fn fs.WalkDirFunc) error {
	if runtime.GOOS != "android" {
		return filepath.WalkDir(root, fn)
	}
	id, start, visited := walkSequence.Add(1), time.Now(), 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		visited++
		if err != nil {
			logging.LogWarnf("walk [%d] native entry [%s] failed: %s", id, path, err)
		}
		return fn(path, entry, err)
	})
	logging.LogInfof("walk [%d] native dir [%s], visited [%d], elapsed [%s], error [%v]", id, root, visited, time.Since(start), err)
	return err
}
