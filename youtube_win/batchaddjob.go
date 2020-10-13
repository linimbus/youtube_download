package main

import (
	"github.com/astaxie/beego/logs"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func AddJobBatch()  {
	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton

	var now, keep *walk.RadioButton

	cnt, err := Dialog{
		AssignTo: &dlg,
		Title: LangValue("batchadd"),
		Icon: ICON_TOOL_DOWNLOAD,
		DefaultButton: &acceptPB,
		CancelButton: &cancelPB,
		Size: Size{400, 500},
		MinSize: Size{400, 500},
		Layout:  VBox{
			Alignment: AlignHNearVNear,
		},
		Children: []Widget{
			Label{
				Text: LangValue("downloadlist"),
			},
			TextEdit{
				VScroll: true,
				Text: "",
			},
			Label{
				Text: LangValue("downloadoptions"),
			},
			Composite{
				Layout: Grid{Columns: 4},
				Children: []Widget{
					CheckBox{
						Text: LangValue("all"),
					},
					CheckBox{
						Text: "webm",
						Checked: true,
					},
					CheckBox{
						Text: "mp4",
						Checked: true,
					},
					CheckBox{
						Text: "Audio",
						Checked: true,
					},
					CheckBox{
						Text: "hd2160(4k)",
					},
					CheckBox{
						Text: "hd1440(2k)",
					},
					CheckBox{
						Text: "hd1080",
					},
					CheckBox{
						Text: "hd720",
					},
					CheckBox{
						Text: "480p",
					},
					CheckBox{
						Text: "360p",
					},
					CheckBox{
						Text: "240p",
					},
					CheckBox{
						Text: "144p",
					},
				},
			},
			Composite{
				Layout: HBox{
					Alignment: AlignHNearVNear,
				},
				Children: []Widget{
					RadioButton{
						AssignTo: &now,
						Text:     LangValue("downloadnow"),
						OnBoundsChanged: func() {
							//now.SetChecked(!video.Keep)
						},
						OnClicked: func() {
							//video.Keep = false
							//now.SetChecked(!video.Keep)
							//keep.SetChecked(video.Keep)
						},
					},
					RadioButton{
						AssignTo: &keep,
						Text:     LangValue("appointmentdownload"),
						OnClicked: func() {
							//video.Keep = true
							//keep.SetChecked(video.Keep)
							//now.SetChecked(!video.Keep)
						},
					},
				},
			},
			Composite{
				Layout: HBox{
					Alignment: AlignHNearVNear,
				},
				Children: []Widget{
					PushButton{
						Text: LangValue("add"),
						OnClicked: func() {
							dlg.Accept()
						},
					},
					PushButton{
						Text: LangValue("cancel"),
						OnClicked: func() {
							dlg.Cancel()
						},
					},
					HSpacer{

					},
				},
			},
		},
	}.Run(MainWindowsCtrl())
	if err != nil {
		logs.Error(err.Error())
	} else {
		logs.Info("add job dialog return %d", cnt)
	}
}
