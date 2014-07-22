// gobuild is a config based tool for building go projects. It supports
// parallelize building and cross-compilation.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	_Verbose   bool
	_Parallel  int
	_Mode      string
	_WG        sync.WaitGroup
	_Semaphore chan bool
	_Results   chan *BuildResult
)

type BuildResult struct {
	Package string
	OS      string
	Arch    string
	Error   error
	Output  string
}

func main() {
	os.Exit(Main())
}

func Main() int {
	var (
		err  error
		g    *Go
		cfg  *BuildConfig
		ok   bool
		init string
	)

	// parse args
	flags := flag.NewFlagSet("gobuild", flag.ExitOnError)
	flags.Usage = printUsage
	flags.StringVar(&init, "i", "", "initialize cross-compilation tools")
	flags.IntVar(&_Parallel, "p", 0, "number of go routine for parallel building")
	flags.BoolVar(&_Verbose, "v", false, "verbose mode")
	flags.StringVar(&_Mode, "m", "", "build mode, develop|test|publish")
	if err := flags.Parse(os.Args[1:]); err != nil {
		flags.Usage()
		return 1
	}
	if _Parallel <= 0 {
		_Parallel = runtime.NumCPU()
	}

	// initialize go components
	g, err = NewGo()
	if err != nil {
		fmt.Println("initialize failed:", err)
		return 1
	}

	// init packages and tools for cross-compilation
	if init != "" {
		fmt.Println("initialize packages and tools...")
		platforms := strings.Split(init, ",")
		ok = true
		for _, platform := range platforms {
			output, err := buildTools(g, platform)
			fmt.Printf(">> %15s -> %s", platform, IfString(err == nil, "success", "failed"))
			if err != nil {
				ok = false
				fmt.Printf(", error: %v", err)
			}
			if _Verbose && output != "" {
				fmt.Printf(", output: \n%s", output)
			}
			fmt.Println()
		}
		return IfInt(ok, 0, 1)
	}

	// load config for building project
	cfg, err = loadConfig(g, flags)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	// build projects parallel
	count := 0
	for _, p := range cfg.Projects {
		count += len(p.Platforms)
	}
	if count == 0 {
		fmt.Println("no projects need to build")
		return 1
	}
	_Results = make(chan *BuildResult, count)
	_Semaphore = make(chan bool, _Parallel)

	// print output informations
	fmt.Printf("build projects with %d routines...\n", _Parallel)
	for _, p := range cfg.Projects {
		buildProject(g, p)
	}

	go func() {
		ok = true
		for r := range _Results {
			fmt.Printf(">> %s(%s/%s) -> %s", r.Package, r.OS, r.Arch, IfString(r.Error == nil, "success", "failed"))
			if r.Error != nil {
				ok = false
				fmt.Printf(", error: %v", r.Error)
			}
			if _Verbose && r.Output != "" {
				fmt.Printf(", output: \n%s", r.Output)
			}
			fmt.Println()
		}
	}()

	_WG.Wait()
	close(_Results)

	return IfInt(ok, 0, 1)
}

// load config for building projects
func loadConfig(g *Go, flags *flag.FlagSet) (*BuildConfig, error) {
	var (
		err       error
		fileType  int
		path      string
		cfgPath   string
		goSrcPath = filepath.Join(g.GetPath(), "src")
	)

	args := flags.Args()
	if len(args) == 0 {
		path = "."
	} else {
		path = args[0]
	}
	if !filepath.IsAbs(path) {
		if strings.HasPrefix(path, ".") {
			wd, err := os.Getwd()
			if err != nil {
				return nil, err
			}
			path = filepath.Join(wd, path)
		} else {
			path = filepath.Join(goSrcPath, path)
		}
	}

	fileType, err = GetFileType(path)
	if err != nil {
		return nil, err
	}

	cfgPath = IfString(fileType == FileType_Directory, filepath.Join(path, "build.xml"), path)
	cfg, err := NewBuildConfig(cfgPath)
	if err != nil {
		return nil, err
	}

	// process package path
	for _, p := range cfg.Projects {
		if filepath.IsAbs(p.Path) {
			p.FullPath = p.Path
		} else {
			if p.Path == "" {
				p.FullPath = filepath.Dir(cfgPath)
			} else if strings.HasPrefix(path, ".") {
				p.FullPath = filepath.Join(filepath.Dir(cfgPath), p.Path)
			} else {
				p.FullPath = filepath.Join(g.GetPath(), "src", p.Path)
			}
		}
	}

	return cfg, nil
}

func buildProject(g *Go, project *Project) {
	for i := 0; i < len(project.Platforms); i++ {
		platform := project.Platforms[i]

		if platform.On != "" && platform.On != g.GetOS() {
			continue
		}

		_WG.Add(1)
		go func() {
			defer func() {
				_WG.Done()
				<-_Semaphore
			}()

			_Semaphore <- true
			_Results <- buildPlatform(g, project.Path, project.FullPath, platform)
		}()
	}
}

func buildPlatform(g *Go, pkgPath, pkgFullPath string, platform *Platform) (result *BuildResult) {
	result = &BuildResult{}

	os := IfString(platform.OS == "", g.GetOS(), platform.OS)
	arch := IfString(platform.Arch == "", g.GetArch(), platform.Arch)
	result.OS, result.Arch = os, arch

	importPath, err := g.GetImportPath(pkgFullPath)
	if err != nil {
		result.Package, result.Error = pkgPath, err
		return
	} else {
		result.Package = importPath
	}

	data := map[string]string{
		"GOPATH":  g.GetPath(),
		"GOOS":    os,
		"GOARCH":  arch,
		"PKGDIR":  filepath.Dir(pkgFullPath),
		"PKGNAME": filepath.Base(pkgFullPath),
	}
	output := ExpandString(platform.Output, data)

	data["OUTPUTDIR"] = filepath.Dir(output)
	data["OUTPUTNAME"] = filepath.Dir(output)
	data["BUILDTIME"] = time.Now().Format("20060102150405")
	executor := NewExecutor(pkgFullPath, data)

	for _, action := range platform.Actions {
		if action.On == "before" {
			result.Error = executor.Execute(action)
			if result.Error != nil {
				return
			}
		}
	}

	result.Output, result.Error = g.Build(importPath, os, arch, output)

	if result.Error == nil {
		for _, action := range platform.Actions {
			if action.On == "after" {
				result.Error = executor.Execute(action)
				if result.Error != nil {
					return
				}
			}
		}
	}

	return
}

func buildTools(g *Go, platform string) (output string, err error) {
	pair := strings.Split(platform, "/")
	if len(pair) != 2 {
		return "", fmt.Errorf("platform invalid")
	}

	os, arch := pair[0], pair[1]
	return g.BuildTools(os, arch)
}

func printUsage() {
	const UsageText = `Usage: gobuild [options] [directory|config]

  gobuild is a tool for building golang project.

Options:

  -i           initialize cross-compilation tools
  -p=-1        number of go routine for parallel building, defaults to number of CPUs
  -v           verbose mode
  -m           build mode: develop|test|publish

`
	fmt.Fprintf(os.Stderr, UsageText)
}
