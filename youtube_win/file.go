package main

import (
	"fmt"
	"os"
)

var DEFAULT_HOME string

const DEFAULT_DIR_HOME = "youtube_downloader"

func LogDirGet() string {
	dir := fmt.Sprintf("%s\\runlog", DEFAULT_HOME)
	_, err := os.Stat(dir)
	if err != nil {
		os.MkdirAll(dir, 644)
	}
	return dir
}

func appDataDir() string {
	datadir := os.Getenv("APPDATA")
	if datadir == "" {
		datadir = os.Getenv("CD")
	}
	if datadir == "" {
		datadir = ".\\"
	} else {
		datadir = fmt.Sprintf("%s\\%s", datadir, DEFAULT_DIR_HOME)
	}
	return datadir
}

func userVideoDir() string {
	userDir := fmt.Sprintf("%s%s\\Videos", os.Getenv("HomeDrive"), os.Getenv("HOMEPATH"))
	_, err := os.Stat(userDir)
	if err != nil {
		userDir = os.Getenv("CD")
	}
	if userDir == "" {
		userDir = ".\\"
	} else {
		userDir = fmt.Sprintf("%s\\%s", userDir, DEFAULT_DIR_HOME)
	}
	_, err = os.Stat(userDir)
	if err != nil {
		os.MkdirAll(userDir, 664)
	}
	return userDir
}

func appDataDirInit() error {
	dir := appDataDir()
	_, err := os.Stat(dir)
	if err != nil {
		err = os.MkdirAll(dir, 644)
		if err != nil {
			return err
		}
	}
	DEFAULT_HOME = dir
	return nil
}

func FileInit() error {
	err := appDataDirInit()
	if err != nil {
		return err
	}
	return nil
}
