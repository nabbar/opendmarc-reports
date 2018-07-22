package version

import (
	"fmt"
	"path"
	"reflect"
	"runtime"

	"strings"

	"github.com/hashicorp/go-version"
	. "github.com/nabbar/opendmarc-reports/logger"
)

/*
Copyright 2017 Nicolas JUHEL

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

const (
	// requires at least Go v1.9.1
	minGoVersion        = "1.10.0"
	goVersionConstraint = ">= " + minGoVersion
)

var (
	// -ldflags "-X version.Release=$(git describe --tags HEAD || git describe --all HEAD) -X version.Build=$(git rev-parse --short HEAD) -X version.Date=$(date +%FT%T%z) -X version.Package=$(basename $(pwd))"

	// Release the git tag of the current build, used with -X version.Release=$(git describe --tags HEAD || git describe --all HEAD)
	Release = "0.0"

	// Build the git commit of the current build, used with -X version.Build=$(git rev-parse --short HEAD)
	Build = "00000"

	// Date the current datetime RFC like for the build, used with -X version.Date=$(date +%FT%T%z)
	Date = "2017-10-21T00:00:00+0200"

	// Package the current package name of the build directory, used with -X version.Package=$(basename $(pwd))
	Package = "noname"

	author = "Nicolas JUHEL"

	prefix = "DMARC"
)

type empty struct{}

// Check if this binary is compiled with at least minimum Go version.
func init() {
	if Package == "" || Package == "noname" {
		Package = path.Base(path.Dir(reflect.TypeOf(empty{}).PkgPath()))
	}

	curVer := runtime.Version()[2:]

	constraint, err := version.NewConstraint(goVersionConstraint)
	if err != nil {
		FatalLevel.Logf("Cannot check GoVersion contraint : %v", err)
	}

	goVersion, err := version.NewVersion(curVer)
	if err != nil {
		FatalLevel.Logf("Cannot extract GoVersion runtime : %v", err)
	}

	if !constraint.Check(goVersion) {
		FatalLevel.Logf("%s is not compiled with Go %s ! Please use Go %s to recompile !", Package, goVersion, minGoVersion)
	}

	//config.InfoLevel.Logf("Runtime Go Version %s is compliance to package '%s'", curVer, Package)
}

// Info print all information about current build and version
func Info() {
	println(fmt.Sprintf("Running %s", GetHeader()))
}

// GetInfo return string about current build and version
func GetInfo() string {
	return fmt.Sprintf("Release: %s, Build: %s, Date: %s", Release, Build, Date)
}

// GetAppId return string about package name, release and runtime info
func GetAppId() string {
	return fmt.Sprintf("%s (OS: %s; Arch: %s)", Release, runtime.GOOS, runtime.GOARCH)
}

// GetAuthor return string about author name and repository info
func GetAuthor() string {
	return fmt.Sprintf("by %s (source : %s)", author, path.Dir(reflect.TypeOf(empty{}).PkgPath()))
}

// GetAuthor return string about author name and repository info
func GetHeader() string {
	return fmt.Sprintf("%s (%s)", Package, GetInfo())
}

func GetPrefix() string {
	return strings.ToUpper(prefix)
}
