package main

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var toolBars *walk.ToolBar

func ToolBarInit() ToolBar {
	return ToolBar{
		AssignTo: &toolBars,
		ButtonStyle: ToolBarButtonImageOnly,
		MinSize: Size{Width: 64, Height: 64},
		Items: []MenuItem{
			Action{
				Text: LangValue("add"),
				Image: ICON_TOOL_ADD,
				OnTriggered: func() {
					AddJobOnce()
				},
			},
			Action{
				Text: LangValue("batchadd"),
				Image: ICON_TOOL_DOWNLOAD,
				OnTriggered: func() {
					AddJobBatch()
				},
			},
			Action{
				Text: LangValue("scheddownloadtime"),
				Image: ICON_TOOL_RESERVE,
				OnTriggered: func() {
					KeepSet()
				},
			},
			Action{
				Text: LangValue("delete"),
				Image: ICON_TOOL_DEL,
				OnTriggered: func() {
					list := JobTableSelectList()
					if len(list) == 0 {
						ErrorBoxAction(MainWindowsCtrl(), LangValue("nochoiceobject"))
						return
					}
					DeleteDiaglog(list)
				},
			},
			//Action{
			//	Text: LangValue("start"),
			//	Image: ICON_TOOL_PLAY,
			//},
			//Action{
			//	Text: LangValue("stop"),
			//	Image: ICON_TOOL_STOP,
			//},
			Action{
				Text: LangValue("setting"),
				Image: ICON_TOOL_SETTING,
				OnTriggered: func() {
					Setting()
				},
			},
		},
	}
}
