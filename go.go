package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type Go struct {
	version string
	os      string
	arch    string
	root    string
	path    string
}

func NewGo() (*Go, error) {
	var (
		output string
		err    error
	)

	g := &Go{path: os.Getenv("GOPATH")}

	output, err = g.ExecuteCmd(os.Environ(), "", "env", "GOROOT")
	if err != nil {
		return nil, err
	} else {
		g.root = strings.TrimSpace(output)
	}

	output, err = g.ExecuteCmd(os.Environ(), "", "version")
	if err != nil {
		return nil, err
	}

	reg := regexp.MustCompile(`(go\d+\.\d+) (\w+)/(\w+)`)
	matches := reg.FindStringSubmatch(output)
	if matches == nil {
		return nil, fmt.Errorf("can not get version info from result: %s", output)
	}

	g.version = matches[1]
	g.os = matches[2]
	g.arch = matches[3]

	return g, nil
}

// get all environment variable
func (this *Go) GetEnviron() map[string]string {
	env := make(map[string]string)
	for _, p := range os.Environ() {
		arr := strings.SplitN(p, "=", 2)
		env[arr[0]] = arr[1]
	}
	return env
}

// prepare environment variable for building
func (this *Go) PrepareEnviron(goos, goarch string) []string {
	m := this.GetEnviron()
	m["GOOS"] = goos
	m["GOARCH"] = goarch

	env := make([]string, len(m))
	i := 0
	for k, v := range m {
		env[i] = k + "=" + v
		i++
	}
	return env
}

// get go version
func (this *Go) GetVersion() string {
	return this.version
}

// get host os
func (this *Go) GetOS() string {
	return this.os
}

// get host architecture
func (this *Go) GetArch() string {
	return this.arch
}

// get GOPATH environment variable
func (this *Go) GetPath() string {
	return this.path
}

// get import path of main package
func (this *Go) GetImportPath(pkg string) (path string, err error) {
	var (
		dir    string
		args   []string
		output string
	)

	if filepath.IsAbs(pkg) {
		dir = pkg
		args = []string{"list", "-f", "{{.Name}},{{.ImportPath}}"}
	} else {
		args = []string{"list", "-f", "{{.Name}},{{.ImportPath}}", pkg}
	}

	output, err = this.ExecuteCmd(nil, dir, args...)
	if err != nil {
		err = fmt.Errorf("%v, output:\n%s", err, output)
		return
	}

	for _, line := range strings.Split(output, "\n") {
		arr := strings.SplitN(line, ",", 2)
		if len(arr) == 2 && arr[0] == "main" {
			path = arr[1]
			return
		}
	}

	err = fmt.Errorf("no main package in '%s'", pkg)
	return
}

// build project
func (this *Go) Build(pkg, os, arch, output string) (result string, err error) {
	env := this.PrepareEnviron(os, arch)

	if !filepath.IsAbs(output) {
		output = filepath.Join(pkg, output)
	}

	if os == "windows" && filepath.Ext(output) != ".exe" {
		output += ".exe"
	}

	// if package path is outside the GOPATH, we move to that directory to build.
	dir := ""
	if filepath.IsAbs(pkg) {
		dir = pkg
		pkg = ""
	}

	// inject BUILD_TIME variable
	ldflags := fmt.Sprintf("-X main.BUILD_TIME '%s'", time.Now().Format("2006-01-02 15:04:05"))
	result, err = this.ExecuteCmd(env, dir, "build", "-i", "-ldflags", ldflags, "-o", output, pkg)
	return
}

// build packages and tools for cross-compilation
func (this *Go) BuildTools(os, arch string) (result string, err error) {
	dir := filepath.Join(this.root, "src")
	cmdName := filepath.Join(dir, IfString(runtime.GOOS == "windows", "make.bat", "make.bash"))
	cmd := exec.Command(cmdName, "--no-clean")
	cmd.Dir = dir
	cmd.Env = this.PrepareEnviron(os, arch)

	var output []byte
	output, err = cmd.CombinedOutput()
	result = string(output)
	return
}

// execute go command, the first arg of args must be a go command like 'build'
func (this *Go) ExecuteCmd(env []string, dir string, args ...string) (result string, err error) {
	var output []byte

	cmd := exec.Command("go", args...)
	if env != nil {
		cmd.Env = env
	}
	if dir != "" {
		cmd.Dir = dir
	}

	output, err = cmd.CombinedOutput()
	result = string(output)
	if err != nil {
		err = fmt.Errorf("go %s > %v", args[0], err)
	}
	return
}
