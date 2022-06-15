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

//go:build android || ios
// +build android ios

package filelock

import (
	"errors"
	"os"
	"sync"

	"github.com/88250/gulu"
)

var ErrUnableLockFile = errors.New("unable to lock file")

func ReleaseFileLocks(boxLocalPath string) {}

func ReleaseAllFileLocks() {}

func OpenFile(filePath string) (*os.File, error) {
	return os.OpenFile(filePath, os.O_RDWR, 0644)
}

func CloseFile(file *os.File) (err error) {
	return file.Close()
}

func RemoveFile(filePath string) (err error) {
	return os.Remove(filePath)
}

func NoLockFileRead(filePath string) (data []byte, err error) {
	return os.ReadFile(filePath)
}

func LockFileRead(filePath string) (data []byte, err error) {
	return os.ReadFile(filePath)
}

func NoLockFileWrite(filePath string, data []byte) (err error) {
	return gulu.File.WriteFileSafer(filePath, data, 0644)
}

func LockFileWrite(filePath string, data []byte) (err error) {
	return gulu.File.WriteFileSafer(filePath, data, 0644)
}

func LockFile(filePath string) (err error) {
	return
}

func UnlockFile(filePath string) (err error) {
	return
}

var fileLocks = sync.Map{}

func IsLocked(filePath string) bool {
	return false
}

func LockFileReadWrite() {
}

func UnlockFileReadWriteLock() {
}
