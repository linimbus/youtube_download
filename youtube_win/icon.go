package main

import (
	"github.com/astaxie/beego/logs"
	"github.com/lxn/walk"
	"os"
)

func IconLoadFromBox(filename string, size walk.Size) *walk.Icon {
	body, err := BoxFile().Bytes(filename)
	if err != nil {
		logs.Error(err.Error())
		return nil
	}
	dir := DEFAULT_HOME + "\\icon\\"
	_, err = os.Stat(dir)
	if err != nil {
		err = os.MkdirAll(dir, 644)
		if err != nil {
			logs.Error(err.Error())
			return nil
		}
	}
	filepath := dir + filename
	err = SaveToFile(filepath, body)
	if err != nil {
		logs.Error(err.Error())
		return nil
	}
	icon, err := walk.NewIconFromFileWithSize(filepath, size)
	if err != nil {
		logs.Error(err.Error())
		return nil
	}
	return icon
}

var ICON_Main          *walk.Icon
var ICON_Main_Mini     *walk.Icon
var ICON_Network_Flow  *walk.Icon

var ICON_Max_Size = walk.Size{
	Width: 128, Height: 128,
}

var ICON_Min_Size = walk.Size{
	Width: 32, Height: 32,
}

func IconInit() error {
	ICON_Main = IconLoadFromBox("main.ico", ICON_Max_Size)
	ICON_Main_Mini = IconLoadFromBox("mainmini.ico", ICON_Min_Size)
	ICON_Network_Flow = IconLoadFromBox("flow.ico", ICON_Min_Size)
	return nil
}

