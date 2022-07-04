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
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/88250/flock"
	"github.com/88250/gulu"
	"go.uber.org/multierr"
)

var (
	ErrUnableLockFile = errors.New("unable to lock file")
)

var (
	fileLocks         = sync.Map{}
	expiration        = 5 * time.Minute
	fileReadWriteLock = sync.Mutex{}
)

type LockItem struct {
	fl      *flock.Flock
	expired int64
}

func init() {
	go func() {
		// 锁定超时自动解锁
		for range time.Tick(10 * time.Second) {
			fileReadWriteLock.Lock()

			now := time.Now().UnixNano()
			var expiredKeys []string
			fileLocks.Range(func(k, v interface{}) bool {
				lockItem := v.(*LockItem)
				if now > lockItem.expired {
					expiredKeys = append(expiredKeys, k.(string))
				}
				return true
			})

			for _, k := range expiredKeys {
				unlockFile0(k)
			}

			fileReadWriteLock.Unlock()
		}
	}()
}

func ReleaseFileLocks(localAbsPath string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()

	fileLocks.Range(func(k, v interface{}) bool {
		if strings.HasPrefix(k.(string), localAbsPath) {
			if unlockErr := unlockFile0(k.(string)); nil != unlockErr {
				err = multierr.Append(err, unlockErr)
			}
		}
		return true
	})
	return
}

func ReleaseAllFileLocks() (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()

	fileLocks.Range(func(k, v interface{}) bool {
		if unlockErr := unlockFile0(k.(string)); nil != unlockErr {
			err = multierr.Append(err, unlockErr)
		}
		return true
	})
	return
}

func OpenFile(filePath string) (ret *os.File, err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()

	v, ok := fileLocks.Load(filePath)
	if !ok {
		lock, lockErr := lockFile0(filePath)
		if nil != lockErr {
			return nil, lockErr
		}
		return lock.Fh(), nil
	}
	ret = v.(*LockItem).fl.Fh()
	if _, err = ret.Seek(0, io.SeekStart); nil != err {
		return
	}
	return
}

func CloseFile(file *os.File) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()

	if nil == file {
		return
	}
	filePath := file.Name()
	v, _ := fileLocks.Load(filePath)
	if nil == v {
		return file.Close()
	}
	lockItem := v.(*LockItem)
	err = lockItem.fl.Unlock()
	fileLocks.Delete(filePath)
	return
}

func RemoveFile(filePath string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()

	_, ok := fileLocks.Load(filePath)
	if ok {
		if err = unlockFile0(filePath); nil != err {
			return
		}
	}
	return os.Remove(filePath)
}

func NoLockFileRead(filePath string) (data []byte, err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()

	v, ok := fileLocks.Load(filePath)
	if !ok {
		return os.ReadFile(filePath)
	}
	lockItem := v.(*LockItem)
	handle := lockItem.fl.Fh()
	if _, err = handle.Seek(0, io.SeekStart); nil != err {
		return
	}
	return io.ReadAll(handle)
}

func LockFileRead(filePath string) (data []byte, err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()

	if !gulu.File.IsExist(filePath) {
		err = os.ErrNotExist
		return
	}

	lock, lockErr := lockFile0(filePath)
	if nil != lockErr {
		err = lockErr
		return
	}

	handle := lock.Fh()
	if _, err = handle.Seek(0, io.SeekStart); nil != err {
		return
	}
	return io.ReadAll(handle)
}

func NoLockFileWrite(filePath string, data []byte) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()

	v, ok := fileLocks.Load(filePath)
	if !ok {
		return os.WriteFile(filePath, data, 0644)
	}

	lockItem := v.(*LockItem)
	handle := lockItem.fl.Fh()
	err = gulu.File.WriteFileSaferByHandle(handle, data)
	return
}

func LockFileWrite(filePath string, data []byte) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()

	lock, lockErr := lockFile0(filePath)
	if nil != lockErr {
		err = lockErr
		return
	}

	handle := lock.Fh()
	err = gulu.File.WriteFileSaferByHandle(handle, data)
	return
}

func IsLocked(filePath string) bool {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()

	v, _ := fileLocks.Load(filePath)
	if nil == v {
		return false
	}
	return true
}

func UnlockFile(filePath string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return unlockFile0(filePath)
}

func unlockFile0(filePath string) (err error) {
	v, _ := fileLocks.Load(filePath)
	if nil == v {
		return
	}
	lockItem := v.(*LockItem)
	err = lockItem.fl.Unlock()
	fileLocks.Delete(filePath)
	return
}

func LockFile(filePath string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	_, err = lockFile0(filePath)
	return
}

func lockFile0(filePath string) (lock *flock.Flock, err error) {
	lockItemVal, _ := fileLocks.Load(filePath)
	var lockItem *LockItem
	if nil == lockItemVal {
		lock = flock.New(filePath)
		var locked bool
		var lockErr error
		for i := 0; i < 7; i++ {
			locked, lockErr = lock.TryLock()
			if nil != lockErr || !locked {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			break
		}

		if nil != lockErr {
			err = multierr.Append(fmt.Errorf("lock file [%s] failed: %s", filePath, lockErr), ErrUnableLockFile)
			return
		}

		if !locked {
			err = multierr.Append(fmt.Errorf("unable to lock file [%s]", filePath), ErrUnableLockFile)
			return
		}
		lockItem = &LockItem{fl: lock}
	} else {
		lockItem = lockItemVal.(*LockItem)
		lock = lockItem.fl
	}
	lockItem.expired = time.Now().Add(expiration).UnixNano()
	fileLocks.Store(filePath, lockItem)
	return
}
