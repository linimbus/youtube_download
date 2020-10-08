package main

import (
	"github.com/astaxie/beego/logs"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"time"
)

var mainWindow *walk.MainWindow

var mainWindowWidth = 430
var mainWindowHeight = 350

func waitWindows()  {
	for  {
		if mainWindow != nil && mainWindow.Visible() {
			break
		}
		time.Sleep(100*time.Millisecond)
	}
	NotifyInit()
}

func statusUpdate()  {

}

func init()  {
	go func() {
		waitWindows()
		for  {
			statusUpdate()
			time.Sleep(time.Second)
		}
	}()
}

func MainWindowsClose()  {
	if mainWindow != nil {
		mainWindow.Close()
		mainWindow = nil
	}
}

func CloseWindows()  {
	MainWindowsClose()
}


func MainWindows() error {
	cnt, err := MainWindow{
		Title:   "YouTube Downloader",
		Icon: ICON_Main,
		AssignTo: &mainWindow,
		MinSize: Size{mainWindowWidth, mainWindowHeight},
		Size: Size{mainWindowWidth, mainWindowHeight},
		Layout:  VBox{},
		MenuItems: MenuBarInit(),
		StatusBarItems: StatusBarInit(),
		ToolBar: ToolBarInit(),
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 3},
			},
			Composite{
				Layout: Grid{Columns: 2},
			},
		},
	}.Run()

	if err != nil {
		logs.Error(err.Error())
	} else {
		logs.Info("main windows exit %d", cnt)
	}

	if err:= recover();err != nil{
		logs.Error(err)
	}

	CloseWindows()
	return nil
}