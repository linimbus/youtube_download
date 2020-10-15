package main

import (
	. "github.com/lxn/walk/declarative"
	"github.com/astaxie/beego/logs"
	"github.com/lxn/walk"
)


func DeleteDiaglog(list []string)  {
	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton
	var deleteFile bool
	var deleteBut *walk.RadioButton

	var show string
	for _, v := range list {
		show += v + "\n"
	}

	cnt, err := Dialog{
		AssignTo: &dlg,
		Title: LangValue("delete"),
		Icon: ICON_TOOL_DEL,
		DefaultButton: &acceptPB,
		CancelButton: &cancelPB,
		MinSize: Size{200, 200},
		Size: Size{200, 200},
		Layout: VBox{},
		Children: []Widget{
			Composite{
				Layout: VBox{MarginsZero: true},
				Children: []Widget{
					Label{
						Text: LangValue("deleteconfirm") + ":",
						Alignment: AlignHNearVCenter,
					},
					TextLabel{
						Text: show,
						Alignment: AlignHNearVCenter,
					},
					RadioButton{
						AssignTo: &deleteBut,
						Alignment: AlignHNearVCenter,
						Text: LangValue("deletefile"),
						OnClicked: func() {
							deleteBut.SetChecked(!deleteFile)
						},
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						Text: LangValue("accpet"),
						OnClicked: func() {
							go func() {
								JobDelete(list, deleteFile)
								dlg.Accept()
							}()
						},
					},
					PushButton{
						Text: LangValue("cancel"),
						OnClicked: func() {
							dlg.Cancel()
						},
					},
				},
			},
		},
	}.Run(MainWindowsCtrl())
	if err != nil {
		logs.Error(err.Error())
	} else {
		logs.Info("delete dialog return %d", cnt)
	}
}