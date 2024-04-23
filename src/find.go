package main

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/triole/logseal"
)

var (
	fi  os.FileInfo
	err error
)

func mkdir(foldername string) {
	os.MkdirAll(foldername, os.ModePerm)
}

func find(basedir string, rxFilter string) []string {
	inf, err := os.Stat(basedir)
	if err != nil {
		lg.IfErrFatal(
			"unable to access md folder", logseal.F{
				"path": basedir, "error": err,
			},
		)
	}
	if inf.IsDir() == false {
		lg.Fatal(
			"not a folder, please provide a directory to look for md files.",
			logseal.F{"path": basedir},
		)
	}

	filelist := []string{}
	rxf, _ := regexp.Compile(rxFilter)

	err = filepath.Walk(basedir, func(path string, f os.FileInfo, err error) error {
		if rxf.MatchString(path) {
			inf, err := os.Stat(path)
			if err == nil {
				if inf.IsDir() == false {
					filelist = append(filelist, path)
				}
			} else {
				lg.IfErrInfo("stat file failed", logseal.F{"path": path})
			}
		}
		return nil
	})
	lg.IfErrFatal("find files failed", logseal.F{"path": basedir})
	return filelist
}
