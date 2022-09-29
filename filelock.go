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
	"os"
	"sync"

	"github.com/88250/gulu"
)

// TODO: 考虑改为每个文件一个锁以提高并发性能

var fileReadWriteLock = sync.Mutex{}

func ReleaseLock() (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return
}

func OpenFile(filePath string) (ret *os.File, err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return os.OpenFile(filePath, os.O_RDWR, 0644)
}

func RemoveFile(filePath string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return os.Remove(filePath)
}

func FileRead(filePath string) (data []byte, err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return os.ReadFile(filePath)
}

func FileWrite(filePath string, data []byte) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return gulu.File.WriteFileSafer(filePath, data, 0644)
}
