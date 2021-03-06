// Copyright 2013 bee authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	path "path/filepath"
	"runtime"
	"strings"
)

var cmdRun = &Command{
	UsageLine: "run [appname] [watchall] [-main=*.go]",
	Short:     "run the app and start a Web server for development",
	Long: `
Run command will supervise the file system of the lessgo project using inotify,
it will recompile and restart the app after any modifications.

`,
}

type ListOpts []string

func (opts *ListOpts) String() string {
	return fmt.Sprint(*opts)
}

func (opts *ListOpts) Set(value string) error {
	*opts = append(*opts, value)
	return nil
}

type docValue string

func (d *docValue) String() string {
	return fmt.Sprint(*d)
}

func (d *docValue) Set(value string) error {
	*d = docValue(value)
	return nil
}

var mainFiles ListOpts

// The flags list of the paths excluded from watching
var excludedPaths strFlags

func init() {
	cmdRun.Run = runApp
	cmdRun.Flag.Var(&mainFiles, "main", "specify main go files")
	cmdRun.Flag.Var(&excludedPaths, "e", "Excluded paths[].")
}

var appname string

func runApp(cmd *Command, args []string) int {
	exit := make(chan bool)
	crupath, _ := os.Getwd()

	if len(args) == 0 || args[0] == "watchall" {
		appname = path.Base(crupath)
		ColorLog("[INFO] Uses '%s' as 'appname'\n", appname)
	} else {
		appname = args[0]
		ColorLog("[INFO] Uses '%s' as 'appname'\n", appname)
		if strings.HasSuffix(appname, ".go") && isExist(path.Join(crupath, appname)) {
			ColorLog("[WARN] The appname has conflic with crupath's file, do you want to build appname as %s\n", appname)
			ColorLog("[INFO] Do you want to overwrite it? [yes|no]]  ")
			if !askForConfirmation() {
				return 0
			}
		}
	}
	Debugf("current path:%s\n", crupath)

	var paths []string

	readAppDirectories(crupath, &paths)

	// Because monitor files has some issues, we watch current directory
	// and ignore non-go files.
	gps := GetGOPATHs()
	if len(gps) == 0 {
		ColorLog("[ERRO] Fail to start[ %s ]\n", "$GOPATH is not set or empty")
		os.Exit(2)
	}
	gopath := gps[0]
	for _, p := range conf.DirStruct.Others {
		paths = append(paths, strings.Replace(p, "$GOPATH", gopath, -1))
	}

	files := []string{}
	for _, arg := range mainFiles {
		if len(arg) > 0 {
			files = append(files, arg)
		}
	}

	NewWatcher(paths, files, false)
	Autobuild(files, false)

	for {
		select {
		case <-exit:
			runtime.Goexit()
		}
	}
	return 0
}

func readAppDirectories(directory string, paths *[]string) {
	fileInfos, err := ioutil.ReadDir(directory)
	if err != nil {
		return
	}

	useDirectory := false
	for _, fileInfo := range fileInfos {
		if strings.HasSuffix(fileInfo.Name(), "docs") {
			continue
		}

		if isExcluded(fileInfo) {
			continue
		}

		if fileInfo.IsDir() == true && fileInfo.Name()[0] != '.' {
			readAppDirectories(directory+"/"+fileInfo.Name(), paths)
			continue
		}

		if useDirectory == true {
			continue
		}

		if path.Ext(fileInfo.Name()) == ".go" {
			*paths = append(*paths, directory)
			useDirectory = true
		}
	}

	return
}

// If a file is excluded
func isExcluded(fileInfo os.FileInfo) bool {
	for _, p := range excludedPaths {
		if strings.HasSuffix(fileInfo.Name(), p) {
			ColorLog("[INFO] Excluding from watching [ %s ]\n", fileInfo.Name())
			return true
		}
	}
	return false
}
