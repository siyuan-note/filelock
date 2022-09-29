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
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/88250/gulu"
	"github.com/siyuan-note/logging"
	"go.uber.org/multierr"
)

// TODO: 考虑改为每个文件一个锁以提高并发性能

var (
	ErrUnableAccessFile = errors.New("unable to access file")
	fileReadWriteLock   = sync.Mutex{}
)

func Move(src, dest string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	err = os.Rename(src, dest)
	return
}

func RoboCopy(src, dest string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()

	if gulu.OS.IsWindows() {
		robocopy := "robocopy"
		cmd := exec.Command(robocopy, src, dest, "/DCOPY:T", "/E", "/IS", "/R:0", "/NFL", "/NDL", "/NJH", "/NJS", "/NP", "/NS", "/NC")
		gulu.CmdAttr(cmd)
		var output []byte
		output, err = cmd.CombinedOutput()
		if strings.Contains(err.Error(), "exit status 16") {
			// 某些版本的 Windows 无法同步 https://github.com/siyuan-note/siyuan/issues/4197
			return gulu.File.Copy(src, dest)
		}

		if nil != err && strings.Contains(err.Error(), exec.ErrNotFound.Error()) {
			robocopy = os.Getenv("SystemRoot") + "\\System32\\" + "robocopy"
			cmd = exec.Command(robocopy, src, dest, "/DCOPY:T", "/E", "/IS", "/R:0", "/NFL", "/NDL", "/NJH", "/NJS", "/NP", "/NS", "/NC")
			gulu.CmdAttr(cmd)
			output, err = cmd.CombinedOutput()
		}
		if nil == err ||
			strings.Contains(err.Error(), "exit status 3") ||
			strings.Contains(err.Error(), "exit status 1") ||
			strings.Contains(err.Error(), "exit status 2") ||
			strings.Contains(err.Error(), "exit status 5") ||
			strings.Contains(err.Error(), "exit status 6") ||
			strings.Contains(err.Error(), "exit status 7") {
			return nil
		}
		logging.LogErrorf("robocopy data from [%s] to [%s] failed: %s %s", src, dest, string(output), err)
	}
	return gulu.File.Copy(src, dest)
}

func Copy(src, dest string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()

	err = gulu.File.Copy(src, dest)
	if isBusy(err) {
		err = multierr.Append(fmt.Errorf("copy [src=%s, dest=%s] failed: %s", src, dest, err), ErrUnableAccessFile)
	}
	return
}

func Remove(p string) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	err = os.RemoveAll(p)
	if isBusy(err) {
		err = multierr.Append(fmt.Errorf("remove file [%s] failed: %s", p, err), ErrUnableAccessFile)
	}
	return
}

func ReleaseLock() (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	return
}

func ReadFile(filePath string) (data []byte, err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	data, err = os.ReadFile(filePath)
	if isBusy(err) {
		err = multierr.Append(fmt.Errorf("read file [%s] failed: %s", filePath, err), ErrUnableAccessFile)
	}
	return
}

func WriteFile(filePath string, data []byte) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()
	err = gulu.File.WriteFileSafer(filePath, data, 0644)
	if isBusy(err) {
		err = multierr.Append(fmt.Errorf("write file [%s] failed: %s", filePath, err), ErrUnableAccessFile)
	}
	return
}

func WriteFileByReader(filePath string, reader io.Reader) (err error) {
	fileReadWriteLock.Lock()
	defer fileReadWriteLock.Unlock()

	err = gulu.File.WriteFileSaferByReader(filePath, reader, 0644)
	if isBusy(err) {
		err = multierr.Append(fmt.Errorf("write file [%s] failed: %s", filePath, err), ErrUnableAccessFile)
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
