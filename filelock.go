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
	"io"
	"os"
	"strings"
	"sync"

	"github.com/88250/gulu"
	"github.com/siyuan-note/logging"
)

// TODO: 考虑改为每个文件一个锁以提高并发性能

var (
	fileReadWriteLock = sync.Mutex{}
)

func Move(src, dest string) (err error) {
	if src == dest {
		return nil
	}

	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	err = os.Rename(src, dest)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "move [src=%s, dest=%s] failed: %s", src, dest, err)
		return
	}
	return
}

func Copy(src, dest string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()

	err = gulu.File.Copy(src, dest)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "copy [src=%s, dest=%s] failed: %s", src, dest, err)
		return
	}
	return
}

func CopyNewtimes(src, dest string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()

	err = gulu.File.CopyNewtimes(src, dest)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "copy [src=%s, dest=%s] failed: %s", src, dest, err)
		return
	}
	return
}

func Rename(p, newP string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	err = os.Rename(p, newP)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "rename [p=%s, newP=%s] failed: %s", p, newP, err)
		return
	}
	return
}

func Remove(p string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	err = os.RemoveAll(p)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "remove file [%s] failed: %s", p, err)
		return
	}
	return
}

func ReadFile(filePath string) (data []byte, err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	data, err = os.ReadFile(filePath)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "read file [%s] failed: %s", filePath, err)
		return
	}
	return
}

func WriteFileWithoutChangeTime(filePath string, data []byte) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	err = gulu.File.WriteFileSaferWithoutChangeTime(filePath, data, 0644)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "write file [%s] failed: %s", filePath, err)
		return
	}
	return
}

func WriteFile(filePath string, data []byte) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	err = gulu.File.WriteFileSafer(filePath, data, 0644)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "write file [%s] failed: %s", filePath, err)
		return
	}
	return
}

func WriteFileByReader(filePath string, reader io.Reader) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()

	err = gulu.File.WriteFileSaferByReader(filePath, reader, 0644)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "write file [%s] failed: %s", filePath, err)
	}
	return
}

func isDenied(err error) bool {
	if nil == err {
		return false
	}

	if errors.Is(err, os.ErrPermission) {
		return true
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "access is denied") || strings.Contains(errMsg, "used by another process")
}
