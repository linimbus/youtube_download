package main

import (
	"github.com/astaxie/beego/logs"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"time"
)

func HourSelectOptions() []string {
	return []string{
		"0", "1","2","4","6","8","10","12","24",
	}
}

func MinuteSelectOptions() []string {
	return []string{
		"0", "10","20","30","45",
	}
}

func ModeOptions() []string {
	return []string{
		LangValue("everyday"),LangValue("weekly"),LangValue("fixeddate"),
	}
}

func KeepSet() {
	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton

	var dataEdit *walk.DateEdit
	var weekBox [7]*walk.CheckBox

	now := time.Now()

	cnt, err := Dialog{
		AssignTo: &dlg,
		Title: LangValue("appointmenttimesetting"),
		Icon: ICON_TOOL_RESERVE,
		DefaultButton: &acceptPB,
		CancelButton: &cancelPB,
		Size: Size{200, 200},
		Layout: VBox{
			Alignment: AlignHNearVNear,
			MarginsZero: true,
		},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{
						Text: LangValue("mode") + ":",
					},
					ComboBox{
						Model: ModeOptions(),
						CurrentIndex: 0,
						OnCurrentIndexChanged: func() {

						},
					},

					Label{
						Text: LangValue("week") + ":",
					},

					Composite{
						Layout: Grid{
							Columns: 2,
							MarginsZero: true,
						},
						Children: []Widget{
							CheckBox{
								Enabled: false,
								AssignTo: &weekBox[0],
								Text: LangValue("sunday"),
							},
							CheckBox{
								Enabled: false,
								AssignTo: &weekBox[1],
								Text: LangValue("monday"),
							},
							CheckBox{
								Enabled: false,
								AssignTo: &weekBox[2],
								Text: LangValue("tuesday"),
							},
							CheckBox{
								Enabled: false,
								AssignTo: &weekBox[3],
								Text: LangValue("wednesday"),
							},
							CheckBox{
								Enabled: false,
								AssignTo: &weekBox[4],
								Text: LangValue("thursday"),
							},
							CheckBox{
								Enabled: false,
								AssignTo: &weekBox[5],
								Text: LangValue("friday"),
							},
							CheckBox{
								Enabled: false,
								AssignTo: &weekBox[6],
								Text: LangValue("saturday"),
							},
						},
					},

					Label{
						Text: LangValue("date") + ":",
					},
					DateEdit{
						AssignTo: &dataEdit,
						Enabled: false,
						MinDate: now,
						MaxDate: now.AddDate(10,0,0),
						OnDateChanged: func() {

						},
					},
					Label{
						Text: LangValue("time") + ":",
					},

					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
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

					Label{
						Text: LangValue("delaytime") + ":",
					},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							ComboBox{
								Model: HourSelectOptions(),
							},
							Label{
								Text: LangValue("hour"),
							},
							ComboBox{
								Model: MinuteSelectOptions(),
							},
							Label{
								Text: LangValue("minute"),
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
