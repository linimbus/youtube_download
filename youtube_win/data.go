package main

import (
	"github.com/astaxie/beego/logs"
	"golang.org/x/sys/windows/registry"
	"syscall"
)

const DATA_KEY = "SOFTWARE\\" + DEFAULT_DIR_HOME

var keyhandle registry.Key

func init()  {
	keyhandle = registry.Key(syscall.InvalidHandle)
}

func keyGet() (registry.Key, error) {
	if syscall.Handle(keyhandle) == syscall.InvalidHandle {
		key, err := registry.OpenKey(registry.CURRENT_USER, DATA_KEY, registry.ALL_ACCESS)
		if err != nil {
			if err != registry.ErrNotExist {
				return 0, err
			}
			key, _, err = registry.CreateKey(registry.CURRENT_USER, DATA_KEY, registry.ALL_ACCESS)
			if err != nil {
				return 0, err
			}
		}
		keyhandle = key
	}
	return keyhandle, nil
}

func DataStringValueGet(name string) string {
	key, err := keyGet()
	if err != nil {
		return ""
	}
	value, _, err := key.GetStringValue(name)
	if err != nil {
		return ""
	}
	return value
}

func DataStringValueSet(name string, value string) error {
	key, err := keyGet()
	if err != nil {
		return err
	}
	return key.SetStringValue(name, value)
}

func DataIntValueGet(name string ) uint32 {
	key, err := keyGet()
	if err != nil {
		return 0
	}
	value, _, err := key.GetIntegerValue(name)
	if err != nil {
		return 0
	}
	return uint32(value)
}

func DataIntValueSet(name string, value uint32) error {
	key, err := keyGet()
	if err != nil {
		return err
	}
	return key.SetDWordValue(name, value)
}

func DataLongValueGet(name string ) uint64 {
	key, err := keyGet()
	if err != nil {
		return 0
	}
	value, _, err := key.GetIntegerValue(name)
	if err != nil {
		return 0
	}
	return value
}

func DataLongValueSet(name string, value uint64) error {
	key, err := keyGet()
	if err != nil {
		return err
	}
	return key.SetQWordValue(name, value)
}

func DataInit() error {
	dir := DataStringValueGet("downloaddir")
	if dir == "" {
		return DataStringValueSet("downloaddir", userVideoDir())
	}
	return nil
}

func DataDownLoadDirGet() string {
	return DataStringValueGet("downloaddir")
}

func DataDownLoadDirSet(dir string) error {
	err := DataStringValueSet("downloaddir", dir)
	if err != nil {
		logs.Error(err.Error())
	}
	return nil
}