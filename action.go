package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Executor struct {
	dir  string
	data map[string]string
}

func NewExecutor(dir string, data map[string]string) *Executor {
	return &Executor{dir, data}
}

func (this *Executor) Execute(actions ...*Action) error {
	for _, action := range actions {
		if action.Mode == "" || action.Mode == _Mode {
			var (
				err  error
				args = strings.Split(action.Args, " ")
			)

			switch action.Name {
			case "exec":
				err = this.Exec(args)
			case "copy":
				err = this.Copy(args)
			case "replace":
				err = this.Replace(args)
			default:
				return fmt.Errorf("action [%s] is not supported", action.Name)
			}

			if err != nil {
				return fmt.Errorf("execute action error > %v", err)
			}
		}
	}

	return nil
}

func (this *Executor) Exec(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command is specified")
	}

	for i := 1; i < len(args); i++ {
		args[i] = ExpandString(args[i], this.data)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = this.dir

	if result, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v, output:\n%s", err, result)
	}

	return nil
}

func (this *Executor) Copy(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("copy action take two arguments")
	}

	src := ExpandString(args[0], this.data)
	dest := ExpandString(args[1], this.data)

	if !filepath.IsAbs(src) {
		src = filepath.Join(this.dir, src)
	}
	if !filepath.IsAbs(dest) {
		dest = filepath.Join(this.dir, dest)
	}

	if strings.ContainsAny(src, "*?") {
		return this.CopyFile(src, dest)
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		return this.CopyDir(src, dest)
	} else {
		return this.CopyFile(src, dest)
	}
}

func (this *Executor) CopyDir(src, dest string) error {
	srcInfo, err := os.Stat(src)

	dest = filepath.Join(dest, filepath.Base(src))
	_, err = os.Stat(dest)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dest, srcInfo.Mode())
		if err != nil {
			return err
		}
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relatedPath := path[len(src):]
		dest := filepath.Join(dest, relatedPath)
		if info.IsDir() {
			return CreateDir(dest, info.Mode())
		} else {
			return CopyFile(path, dest, info.Mode())
		}
	})
}

func (this *Executor) CopyFile(src, dest string) error {
	// ioutil.ReadDir(dirname)
	files, err := filepath.Glob(src)
	if err != nil {
		return err
	}

	if files == nil {
		return nil
	}

	err = CreateDir(dest, os.ModePerm)
	if err != nil {
		return err
	}

	for _, f := range files {
		fi, err := os.Stat(f)
		if err != nil {
			return err
		}

		err = CopyFile(f, filepath.Join(dest, filepath.Base(f)), fi.Mode())
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *Executor) Replace(args []string) error {
	var useRegexp bool
	flags := flag.NewFlagSet("action", flag.ContinueOnError)
	flags.BoolVar(&useRegexp, "r", false, "use regular expression")
	err := flags.Parse(args)
	if err != nil {
		return err
	}

	args = flags.Args()
	if len(args) != 3 {
		return fmt.Errorf("replace action should take 3 arguments")
	}

	path := ExpandString(args[0], this.data)
	if !filepath.IsAbs(path) {
		path = filepath.Join(this.dir, path)
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	if useRegexp {
		reg, err := regexp.Compile(args[1])
		if err != nil {
			return err
		}
		data = reg.ReplaceAll(data, []byte(args[2]))
	} else {
		data = bytes.Replace(data, []byte(args[1]), []byte(args[2]), -1)
	}

	fi, err := file.Stat()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, fi.Mode())
}

func (this *Executor) GZip(args []string) error {
	return nil
}
