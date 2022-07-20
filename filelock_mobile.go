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

package filelock

import (
	"errors"
	"os"
	"sync"

	"github.com/88250/gulu"
)

var (
	ErrUnableLockFile = errors.New("unable to lock file")

	fileReadWriteLock = sync.Mutex{}
)

func ReleaseFileLocks(localAbsPath string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return
}

func ReleaseAllFileLocks() (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return
}

func OpenFile(filePath string) (ret *os.File, err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return os.OpenFile(filePath, os.O_RDWR, 0644)
}

func CloseFile(file *os.File) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return file.Close()
}

func RemoveFile(filePath string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return os.Remove(filePath)
}

func NoLockFileRead(filePath string) (data []byte, err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return os.ReadFile(filePath)
}

func LockFileRead(filePath string) (data []byte, err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return os.ReadFile(filePath)
}

func NoLockFileWrite(filePath string, data []byte) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return gulu.File.WriteFileSafer(filePath, data, 0644)
}

func LockFileWrite(filePath string, data []byte) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return gulu.File.WriteFileSafer(filePath, data, 0644)
}

func LockFile(filePath string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return
}

func UnlockFile(filePath string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return
}

func IsLocked(filePath string) bool {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return false
}
