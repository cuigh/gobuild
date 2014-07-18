# gobuild

gobuild is a config based tool for building go projects. gobuild supports
parallelize building and cross-compilation.

## Install

To install gobuild, just use `go get` command. 

```
$ go get github.com/cuigh/gobuild
```

## Usage

First, if you want to use cross-compilation, you must initialize tools and packages:

```
$ gobuild -i darwin/amd64,windows/386,linux/amd64
```

gobuild is very similar to `go build` command. For example, if you want to build the current package, just call `gobuild`:

```
$ gobuild
build projects with 8 routines...
>> github.com/cuigh/gobuild(darwin/amd64) -> success
```

Or if you want to specify routine count for parallel building(default is number of CPU):

```
$ $ gobuild -p=4 -v github.com/cuigh/gobuild
```

You also can build projects from configuration:

```
$ gobuild github.com/cuigh/gobuild/build.xml
```

Run `gobuild -h` for help and additional informations.

## Configure

If you want to build projects with configuration, here is a full example:

```
<?xml version="1.0" encoding="UTF-8"?>
<projects version="1.0">
	<project path="mtime.com/tools/gobuild">
		<platform os="windows" arch="" on="windows" output="${GOPATH}/bin/${PKGNAME}1/${PKGNAME}">
			<actions>
				<action name="copy" args="config ${OUTPUT}" on="after"/>
			</actions>
		</platform>
		<platform os="darwin" arch="" on="darwin" output="${GOPATH}/bin/${PKGNAME}1/${PKGNAME}">
			<actions>
				<action name="copy" args="config/*.conf ${OUTPUTDIR}/config" on="after"/>
				<!-- <action name="replace" args="${OUTPUTDIR}/config/url.conf test.com xxx.com" on="after" mode="publish"/> -->
			</actions>
		</platform>
		<platform os="linux" arch="" on="linux" output="${GOPATH}/bin/${PKGNAME}1/${PKGNAME}">
			<actions>
				<action name="copy" args="config/*.conf ${OUTPUTDIR}/config" on="after"/>
			</actions>
		</platform>	
	</project>
</projects>
```
