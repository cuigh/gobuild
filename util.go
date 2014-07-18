package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

const (
	FileType_File      = 1
	FileType_Directory = 2
)

func IfString(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}
	return falseVal
}

func IfInt(condition bool, trueVal, falseVal int) int {
	if condition {
		return trueVal
	}
	return falseVal
}

// replaces ${var} or $var in the string based on map data
func ExpandString(s string, data map[string]string) string {
	return os.Expand(s, func(p string) string {
		v, _ := data[p]
		return v
	})
}

// create dir if not exist
func CreateDir(dir string, mode os.FileMode) error {
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dir, mode)
		}
	}
	return err
}

func CopyFile(src, dest string, mode os.FileMode) error {
	data, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dest, data, mode)
}

// get file type, 0-file, 1-directory
func GetFileType(path string) (fileType int, err error) {
	var fi os.FileInfo

	fi, err = os.Stat(path)
	if err == nil {
		fileType = IfInt(fi.IsDir(), FileType_Directory, FileType_File)
	} else if os.IsNotExist(err) {
		err = fmt.Errorf("can not find directory or file: %s", path)
	}
	return
}
