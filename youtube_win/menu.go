package main

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var LangActions []*walk.Action

func LangOnTriggeredSet(idx int)  {
	LangOptionSet(idx)
	for i, v := range LangActions {
		if i == idx {
			v.SetChecked(true)
		} else {
			v.SetChecked(false)
		}
	}
	InfoBoxAction(MainWindowsCtrl(), LangValue("rebootsetting"))
}

func MenuBarInit() []MenuItem {
	langs := LangOptionGet()
	idx := LangOptionIdx()

	LangActions = make([]*walk.Action, len(langs))

	var langMenu []MenuItem
	for i, v := range langs {
		trigger := func(index int) func() {
			return func() {
				LangOnTriggeredSet(index)
			}
		}

		action := Action{
			AssignTo: &LangActions[i],
			Text: v,
			OnTriggered: trigger(i),
		}
		if idx == i {
			action.Checked = true
		}
		langMenu = append(langMenu, action)
	}

	return []MenuItem{
		Menu{
			Text: LangValue("langname"),
			Items: langMenu,
		},
		Action{
			Text: LangValue("miniwin"),
			OnTriggered: func() {
				MainWindowsVisible(false)
			},
		},
		Action{
			Text: LangValue("about"),
			OnTriggered: func() {
				AboutAction(mainWindowCtrl.ctrl)
			},
		},
	}
}
