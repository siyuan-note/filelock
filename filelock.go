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

var (
	lockMutex      = sync.Mutex{}
	operatingFiles = map[string]*sync.Mutex{}
)

func Lock(filePath string) {
	lock(filePath)
}

func Unlock(filePath string) {
	unlock(filePath)
}

func IsLocked(filePath string) bool {
	lockMutex.Lock()
	defer lockMutex.Unlock()

	mutex := operatingFiles[filePath]
	if nil == mutex {
		return false
	}
	return gulu.IsMutexLocked(mutex)
}

func OpenFile(filePath string, flag int, perm os.FileMode) (file *os.File, err error) {
	lock(filePath)

	file, err = os.OpenFile(filePath, flag, perm)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "open file [%s] failed: %s", filePath, err)
		return
	}
	return
}

func CloseFile(file *os.File) (err error) {
	if nil == file {
		return
	}

	err = file.Close()
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "close file [%s] failed: %s", file.Name(), err)
		return
	}

	unlock(file.Name())
	return
}

func IsExist(filePath string) (ret bool) {
	lock(filePath)

	ret = gulu.File.IsExist(filePath)

	unlock(filePath)
	return
}

func Copy(src, dest string) (err error) {
	lock(src)

	err = gulu.File.Copy(src, dest)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "copy [src=%s, dest=%s] failed: %s", src, dest, err)
		return
	}

	unlock(src)
	return
}

func CopyNewtimes(src, dest string) (err error) {
	lock(src)

	err = gulu.File.CopyNewtimes(src, dest)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "copy [src=%s, dest=%s] failed: %s", src, dest, err)
		return
	}

	unlock(src)
	return
}

func Rename(p, newP string) (err error) {
	if p == newP {
		return nil
	}

	lock(p)

	if gulu.File.IsExist(newP) && gulu.File.IsDir(p) && gulu.File.IsDir(newP) {
		err = gulu.File.Copy(p, newP)
		if isDenied(err) {
			logging.LogFatalf(logging.ExitCodeFileSysErr, "copy [p=%s, newP=%s] failed: %s", p, newP, err)
			return
		}
		err = os.RemoveAll(p)
		if isDenied(err) {
			logging.LogFatalf(logging.ExitCodeFileSysErr, "remove [%s] failed: %s", p, err)
			return
		}
		return
	}

	err = os.Rename(p, newP)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "rename [p=%s, newP=%s] failed: %s", p, newP, err)
		return
	}

	unlock(p)
	return
}

func Remove(p string) (err error) {
	lock(p)

	err = os.RemoveAll(p)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "remove file [%s] failed: %s", p, err)
		return
	}

	unlock(p)
	return
}

func ReadFile(filePath string) (data []byte, err error) {
	lock(filePath)

	data, err = os.ReadFile(filePath)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "read file [%s] failed: %s", filePath, err)
		return
	}

	unlock(filePath)
	return
}

func WriteFileWithoutChangeTime(filePath string, data []byte) (err error) {
	lock(filePath)

	err = gulu.File.WriteFileSaferWithoutChangeTime(filePath, data, 0644)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "write file [%s] failed: %s", filePath, err)
		return
	}

	unlock(filePath)
	return
}

func WriteFile(filePath string, data []byte) (err error) {
	lock(filePath)

	err = gulu.File.WriteFileSafer(filePath, data, 0644)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "write file [%s] failed: %s", filePath, err)
		return
	}

	unlock(filePath)
	return
}

func WriteFileByReader(filePath string, reader io.Reader) (err error) {
	lock(filePath)

	err = gulu.File.WriteFileSaferByReader(filePath, reader, 0644)
	if isDenied(err) {
		logging.LogFatalf(logging.ExitCodeFileSysErr, "write file [%s] failed: %s", filePath, err)
	}

	unlock(filePath)
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

func lock(filePath string) {
	lockMutex.Lock()
	defer lockMutex.Unlock()

	mutex := operatingFiles[filePath]
	if nil == mutex {
		mutex = &sync.Mutex{}
		operatingFiles[filePath] = mutex
	}
	mutex.Lock()
}

func unlock(filePath string) {
	lockMutex.Lock()
	defer lockMutex.Unlock()

	mutex := operatingFiles[filePath]
	if nil != mutex {
		mutex.Unlock()
	}
}
