package confs

import (
	"fmt"
	"os"
	"path"
	"bytes"
	"strings"
	"errors"
	"io/ioutil"
	"crypto/md5"
	"golang.org/x/sys/unix"
)

const (
	HashAttr string = "user.hash"
	OriginAttr string = "user.origin"

	checkOnly = false
	createIfNeeded = true
)

var (
	nameError   = errors.New("path is not abolute or contains invalid characters")
	treeError   = errors.New("invalid subtree")
	pathError   = errors.New("invalid path")
	dirError    = errors.New("not a directory")
	fileError   = errors.New("invalid file")
	attrError   = errors.New("extended file attribute error")
	originError = errors.New("mismatching origins")
	defError    = errors.New("definition file is missing or invalid")
	lengthError = errors.New("only part could be written")
	hashError   = errors.New("invalid file hash")
)


func IsValidPath(path string) bool {
	if !strings.HasPrefix(path, "/") {
		return false
	}
	for i := 0;  i < len(path);  i++ {
		c := byte(path[i])
		if c != 0x2f && c != 0x5f && (c < 0x30 || (c > 0x39 && c < 0x41) || (c > 0x5a && c < 0x61) || c > 0x7a) {
			return false
		}
	}
	return true
}

func splitPath(confsPath string) (string, []string, *Error) {
	var dirs []string

	if !IsValidPath(confsPath) {
		return "", []string{}, newError(nameError, confsPath)
	}

	if dirs = strings.Split(confsPath, "/"); len(dirs) < 2 {
		return "", []string{}, newError(pathError, confsPath)
	}

	if !subTrees[dirs[1]] {
		return "", []string{}, newError(treeError, "/" + dirs[1])
	}

	return dirs[1], dirs[2:], nil
}

func checkDirPath(subtree string, dirs []string, create bool) (bool, *Error) {
	dir := ""
	exists := true
	jsFile := ""

	for i := 0;  i < len(dirs);  i++ {
		dir += ("/" + dirs[i])
		dropPath := fmt.Sprintf("%s/%s%s", dropZone, subtree, dir)
		if dir == "/" {
			jsFile = fmt.Sprintf("%s/root.js")
		} else {
			jsFile = fmt.Sprintf("%s%s.js", defRoot, dir)
		}

		if info, err := os.Stat(jsFile); err != nil || !info.Mode().IsRegular() {
			return false, newError(defError, jsFile)
		}

		if exists {
			if info, err := os.Stat(dropPath); err == nil {
				if !info.IsDir() {
					return false, newError(dirError, dropPath)
				}
			} else {
				if !os.IsNotExist(err) {
					return false, newError(err, dropPath)
				}
				if !create {
					exists = false
				} else {
					if err := os.MkdirAll(dropPath, 0755); err != nil {
						return false, newError(err, dropPath)
					}
				}
			}
		}
	}

	return exists, nil
}

func CheckDirPath(confsPath string, create bool) (bool, *Error) {
	subtree, entries, err := splitPath(confsPath)

	if err != nil {
		return false, err
	}

	return checkDirPath(subtree, entries, create)
}

func CheckFilePath(confsPath string, origin string, checkDir bool) (string, *Error) {
	subtree, entries, err := splitPath(confsPath)

	if err != nil {
		return "", err
	}
	if len(entries) < 1 {
		return "", newError(pathError, confsPath)
	}

	dirs := entries[:len(entries)-1]
	dir := "/" + strings.Join(dirs, "/")
	file := entries[len(entries)-1]
	jsFile := fmt.Sprintf("%s%s/%s.js", defRoot, dir, file)
	dropPath := fmt.Sprintf("%s/%s%s/%s", dropZone, subtree, dir, file)

	if info, err := os.Stat(jsFile); err != nil || !info.Mode().IsRegular() {
		return "", newError(defError, jsFile)
	}

	if info, err := os.Stat(dropPath); err == nil {
		if !info.Mode().IsRegular() {
			return "", newError(fileError, dropPath)
		}
		if len(origin) > 0 {
			if o, err := getFileOrigin(dropPath); err == nil && o != origin {
				return "", newError(originError, dropPath)
			}
		}
	} else {
		if checkDir {
			if _, err := checkDirPath(subtree, dirs, createIfNeeded); err != nil {
				return "", err
			}
		}
	}
	
	return dropPath, nil
}


func getFileOrigin(dropPath string) (string, *Error) {
	buf := make([]byte, 32)
	size, err := unix.Getxattr(dropPath, OriginAttr, buf);

	if err != nil {
		return "", newError(err, dropPath)
	}

	return string(buf[:size]), nil
}

func setFileOrigin(dropPath, origin string, overwrite bool) *Error {
	var flags int

	if overwrite {
		flags = 2
	} else {
		flags = 1
	}

	if err := unix.Setxattr(dropPath, OriginAttr, []byte(origin), flags); err != nil {
		return newError(err, dropPath)
	}

	return nil
}

func getFileHash(dropPath string) ([]byte, *Error) {
	hash := make([]byte, md5.Size)
	size, err := unix.Getxattr(dropPath, HashAttr, hash)

	if err != nil {
		return nil, newError(err, dropPath)
	} else if size != md5.Size {
		return nil, newError(hashError, dropPath)
	}

	return []byte(hash), nil
}

func setFileHash(dropPath string, hash []byte, overwrite bool) *Error {
	var flags int

	if overwrite {
		flags = 2
	} else {
		flags = 1
	}

	if err := unix.Setxattr(dropPath, HashAttr, hash[:], flags); err != nil {
		return newError(err, dropPath)
	}

	return nil
}

func writeFile(dropPath string, content []byte, origin string) *Error {
	var (
		file *os.File = nil
		err error = nil
		tmpName string = dropPath
		info os.FileInfo
		size int
	)

	hash := md5.Sum(content)

	if !force {
		if h, err := getFileHash(dropPath); err == nil {
			if bytes.Compare(h, hash[:]) == 0 {
				Infof("'%s' does not overwrite '%s' with identical content", origin, dropPath)
				return nil
			}
		}
	}
	
	if file, err = ioutil.TempFile(tmpDir, path.Base(dropPath)); err == nil {
		if info, err = file.Stat(); err == nil {
			tmpName = fmt.Sprintf("%s/%s", tmpDir, info.Name())
			if size, err = file.Write(content); err == nil && size != len(content) {
				err = lengthError
			}
		}
		file.Close()
	}
	if err != nil {
		return newError(err, tmpName)
	}

	if err := setFileOrigin(tmpName, origin, false); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := setFileHash(tmpName, hash[:], false); err != nil {
		os.Remove(tmpName)
		return err
	}
	

	if err = os.Rename(tmpName, dropPath); err != nil {
		os.Remove(tmpName)
		return newError(err, tmpName)
	}

	Infof("'%s' wrote '%s'", origin, dropPath)

	return nil
}
