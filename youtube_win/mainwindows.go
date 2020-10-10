package main

import (
	"github.com/astaxie/beego/logs"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"os"
	"time"
)

var mainWindowWidth = 430
var mainWindowHeight = 250

func ButtonWight() []Widget {
	var start *walk.PushButton
	var stop *walk.PushButton

	return []Widget{
		PushButton{
			AssignTo:  &start,
			Text:      LangValue("start"),
			OnClicked: func() {
				copy, err := walk.Clipboard().Text()
				if err != nil {
					logs.Error("no ")
				} else {
					lineEdit.SetText(copy)
				}
			},
		},
		PushButton{
			AssignTo:  &stop,
			Enabled:   false,
			Text:      LangValue("stop"),
			OnClicked: func() {

			},
		},
	}
}

var lineEdit *walk.LineEdit

func TableWight() []Widget {
	return []Widget{
		Label{
			Text: "Url",
		},
		LineEdit{
			ReadOnly: true,
			AssignTo: &lineEdit,
		},
	}
}

type MainWindowCtrl struct {
	instance *MainWindow
	ctrl     *walk.MainWindow
	exitInt int
	exit    chan struct{}
}

var mainWindowCtrl *MainWindowCtrl

func init() {
	mainWindowCtrl = new(MainWindowCtrl)
	mainWindowCtrl.exit = make(chan struct{}, 10)
}

func MainWindowsVisible(flag bool)  {
	if mainWindowCtrl.ctrl != nil {
		mainWindowCtrl.ctrl.SetVisible(flag)
	}
}

func AppExitPreFunc()  {
	if mainWindowCtrl.ctrl != nil {
		mainWindowCtrl.ctrl.Close()
		mainWindowCtrl.ctrl = nil
	}
	NotifyExit()
	if err:= recover();err != nil{
		logs.Error(err)
	}
}

func MainWindowsCtrl() *walk.MainWindow {
	return mainWindowCtrl.ctrl
}

func MainWindowsExit()  {
	CapSignal(AppExitPreFunc)
	<- mainWindowCtrl.exit
	AppExitPreFunc()
	os.Exit(mainWindowCtrl.exitInt)
}

func MainWindowStart() error {
	logs.Info("main windows start")
	mainWindowCtrl.instance = mainWindowBuilder(&mainWindowCtrl.ctrl)

	go func() {
		cnt, err := mainWindowCtrl.instance.Run()
		if err != nil {
			logs.Error(err.Error())
		}
		mainWindowCtrl.exitInt = cnt
		mainWindowCtrl.exit <- struct{}{}
		logs.Info("main windows close")
	}()

	for  {
		if mainWindowCtrl.ctrl != nil && mainWindowCtrl.ctrl.Visible() {
			mainWindowCtrl.ctrl.SetSize(walk.Size{
				mainWindowWidth,
				mainWindowHeight})
			break
		}
		time.Sleep(200*time.Millisecond)
	}

	NotifyInit(mainWindowCtrl.ctrl)

	logs.Info("main windows start success")
	return nil
}

func mainWindowBuilder(mw **walk.MainWindow) *MainWindow {
	return &MainWindow{
		Title:   "YouTube Downloader",
		Icon: ICON_Main,
		AssignTo: mw,
		MinSize: Size{mainWindowWidth, mainWindowHeight-1},
		Size: Size{mainWindowWidth, mainWindowHeight-1},
		Layout:  VBox{
			Alignment: AlignHNearVNear,
			MarginsZero: true,
			Margins: Margins{Left: 5, Top: 5},
		},
		MenuItems: MenuBarInit(),
		StatusBarItems: StatusBarInit(),
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 8},
				Children: []Widget{
					ToolBarInit(),
				},
			},
			Composite{
				Layout: Grid{Columns: 2},
				Children: TableWight(),
			},
			Composite{
				Layout: Grid{Columns: 2},
				Children: ButtonWight(),
			},
			Composite{
				Layout: Grid{Columns: 2},
				Children: ButtonWight(),
			},
			Composite{
				Layout: Grid{Columns: 2},
				Children: ButtonWight(),
			},
		},
	}
}
