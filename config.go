package main

import (
	"encoding/xml"
	"os"
	// "path/filepath"
)

type BuildConfig struct {
	XMLName  xml.Name   `xml:"projects"`
	Projects []*Project `xml:"project"`
}

type Project struct {
	Path      string `xml:"path,attr"`
	FullPath  string
	Platforms []*Platform `xml:"platform"`
}

type Platform struct {
	OS      string    `xml:"os,attr"`
	Arch    string    `xml:"arch,attr"`
	Output  string    `xml:"output,attr"`
	On      string    `xml:"on,attr"`
	Actions []*Action `xml:"actions>action"`
}

type Action struct {
	Name string `xml:"name,attr"`
	Args string `xml:"args,attr"`
	Mode string `xml:"mode,attr"`
	On   string `xml:"on,attr"`
}

func NewBuildConfig(configPath string) (*BuildConfig, error) {
	var cfg *BuildConfig

	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = DefaultBuildConfig()
		} else {
			return nil, err
		}
	} else {
		cfg = &BuildConfig{}
		decoder := xml.NewDecoder(file)
		err = decoder.Decode(cfg)
		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func DefaultBuildConfig() *BuildConfig {
	cfg := &BuildConfig{}

	project := new(Project)
	cfg.Projects = []*Project{project}

	platform := new(Platform)
	project.Platforms = []*Platform{platform}
	platform.Output = "${GOPATH}/bin/${PKGNAME}/${PKGNAME}"

	action := new(Action)
	action.Name = "copy"
	action.Args = "config/*.conf ${OUTPUTDIR}/config"
	action.On = "after"
	platform.Actions = []*Action{action}

	return cfg
}
