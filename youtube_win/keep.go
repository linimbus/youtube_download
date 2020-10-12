package main

import (
	"github.com/astaxie/beego/logs"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"time"
)

func TimestampSelectOptions() []string {
	return []string{
		"1h","2h","4h","6h","8h","10h","12h","24h",
	}
}

func KeepSet() {
	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton

	now := time.Now()

	cnt, err := Dialog{
		AssignTo: &dlg,
		Title: LangValue(""),
		Icon: walk.IconInformation(),
		DefaultButton: &acceptPB,
		CancelButton: &cancelPB,
		Size: Size{300, 200},
		MinSize: Size{300, 200},
		Layout: VBox{
			Alignment: AlignHNearVNear,
			MarginsZero: true,
		},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{
						Text: "Date" + ":",
					},
					DateEdit{
						MinDate: now,
						MaxDate: now.AddDate(10,0,0),
						OnDateChanged: func() {

						},
					},
					NumberEdit{
						MaxValue: 23,
						MinValue: 0,
						Value: float64(now.Hour()),
						SpinButtonsVisible: true,
					},
					NumberEdit{
						MaxValue: 59,
						MinValue: 0,
						Value: float64(now.Minute()),
						SpinButtonsVisible: true,
					},
					NumberEdit{
						MaxValue: 59,
						MinValue: 0,
						Value: float64(now.Second()),
						SpinButtonsVisible: true,
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{
						Text: "Setting" + ":",
					},
					ComboBox{
						Model: TimestampSelectOptions(),
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					RadioButtonGroup{
						Optional: true,
						Buttons: []RadioButton{
							{
								Text: "每天",
							},
							{
								Text: "每周",
							},
							{
								Text: "一次性",
							},
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
						Text: LangValue("save"),
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
