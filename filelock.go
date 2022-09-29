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
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/88250/gulu"
	"go.uber.org/multierr"
)

// TODO: 考虑改为每个文件一个锁以提高并发性能

var (
	ErrUnableLockFile = errors.New("unable to lock file")
	fileReadWriteLock = sync.Mutex{}
)

func ReleaseLock() (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return
}

func OpenFile(filePath string) (ret *os.File, err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	ret, err = os.OpenFile(filePath, os.O_RDWR, 0644)
	if isBusy(err) {
		err = multierr.Append(fmt.Errorf("open file [%s] failed: %s", filePath, err), ErrUnableLockFile)
	}
	return
}

func RemoveFile(filePath string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	err = os.Remove(filePath)
	if isBusy(err) {
		err = multierr.Append(fmt.Errorf("remove file [%s] failed: %s", filePath, err), ErrUnableLockFile)
	}
	return
}

func FileRead(filePath string) (data []byte, err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	data, err = os.ReadFile(filePath)
	if isBusy(err) {
		err = multierr.Append(fmt.Errorf("read file [%s] failed: %s", filePath, err), ErrUnableLockFile)
	}
	return
}

func FileWrite(filePath string, data []byte) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	err = gulu.File.WriteFileSafer(filePath, data, 0644)
	if isBusy(err) {
		err = multierr.Append(fmt.Errorf("write file [%s] failed: %s", filePath, err), ErrUnableLockFile)
	}
	return
}

func isBusy(err error) bool {
	if nil == err {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "access is denied") || strings.Contains(errMsg, "used by another process")
}
