package main

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"time"
)

type KeepCfg struct {
	Mode int
	Week [7]bool
	Time time.Time
}

func (k *KeepCfg)WeekEnable() bool {
	if k.Mode == 1 {
		return true
	}
	return false
}

func (k *KeepCfg)DateEnable() bool {
	if k.Mode == 2 {
		return true
	}
	return false
}

func KeepCfgGet() *KeepCfg {
	now := time.Now().Add(time.Minute)

	var cfg KeepCfg
	value := DataStringValueGet("keepconfig")
	if value != "" {
		err := json.Unmarshal([]byte(value), &cfg)
		if err != nil {
			logs.Error(err.Error())
		} else {
			if cfg.Time.Before(now) {
				cfg.Time = time.Date(now.Year(), now.Month(), now.Day(),
					cfg.Time.Hour(), cfg.Time.Minute(), cfg.Time.Second(),
					0, now.Location())
			}
			return &cfg
		}
	}

	time := time.Date(now.Year(), now.Month(), now.Day(),
		23, 30, 0,
		0, now.Location())
	return &KeepCfg{Time: time}
}

func KeepCfgSet(cfg *KeepCfg) error {
	value, err := json.Marshal(cfg)
	if err != nil {
		logs.Error(err.Error())
		return err
	}
	err = DataStringValueSet("keepconfig", string(value))
	if err != nil {
		logs.Error(err.Error())
		return err
	}
	return nil
}

func HourSelectOptions() []string {
	return []string{
		"0","1","2","4","8","12","18",
	}
}

func MinuteSelectOptions() []string {
	return []string{
		"0", "5", "10", "15", "30", "45",
	}
}

func ModeOptions() []string {
	return []string{
		LangValue("everyday"),
		LangValue("weekly"),
		LangValue("fixeddate"),
	}
}

var WEEK_NAME_KEY = []string{
	"sunday", "monday", "tuesday", "wednesday",
	"thursday", "friday", "saturday",
}

func WeekName(week int) string {
	return LangValue(WEEK_NAME_KEY[week])
}

func weekCheckBoxGet(cfg *KeepCfg, boxs []*walk.CheckBox) []Widget {
	var output []Widget

	for i:=0 ; i < 7 ; i++ {
		handler := func(i int) func() {
			return func() {
				boxs[i].SetChecked(!cfg.Week[i])
				cfg.Week[i] = !cfg.Week[i]
			}
		}
		output = append(output, CheckBox{
			Enabled: cfg.WeekEnable(),
			Checked: cfg.Week[i],
			AssignTo: &boxs[i],
			Text: WeekName(i),
			OnClicked: handler(i),
		},)
	}
	return output
}

func timeNowShow(now time.Time) string {
	year, month, day := now.Date()
	return fmt.Sprintf(
		"%4d-%02d-%02d %02d:%02d %s",
		year, month, day, now.Hour(), now.Minute(), WeekName(int(now.Weekday())))
}



func KeepSet() {
	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton

	var dataEdit *walk.DateEdit
	var weekBoxs [7]*walk.CheckBox

	var mode *walk.ComboBox
	var hourEdit, minEdit *walk.NumberEdit
	var hourDelay, minDelay *walk.ComboBox

	cfg := KeepCfgGet()

	oldcfgTime := cfg.Time

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
						Text: LangValue("timenow") + ":",
						MinSize: Size{Height: 30},
					},
					Label{
						MinSize: Size{Height: 30},
						Text: timeNowShow(time.Now()),
						Font: Font{Bold: true},
					},
					Label{
						Text: LangValue("mode") + ":",
					},
					ComboBox{
						AssignTo: &mode,
						Model: ModeOptions(),
						CurrentIndex: cfg.Mode,
						OnCurrentIndexChanged: func() {
							cfg.Mode = mode.CurrentIndex()

							if cfg.Mode == 1 {
								for _,v := range weekBoxs {
									v.SetEnabled(true)
								}
							} else {
								for _,v := range weekBoxs {
									v.SetEnabled(false)
								}
							}

							if cfg.Mode == 2 {
								dataEdit.SetEnabled(true)
							} else {
								dataEdit.SetEnabled(false)
							}
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
						Children: weekCheckBoxGet(cfg, weekBoxs[:]),
					},

					Label{
						Text: LangValue("date") + ":",
					},
					DateEdit{
						AssignTo: &dataEdit,
						Enabled: cfg.DateEnable(),
						MinDate: cfg.Time,
						MaxDate: cfg.Time.AddDate(10,0,0),
						OnDateChanged: func() {
							oldTime := cfg.Time
							cfg.Time = time.Date(
								dataEdit.Date().Year(), dataEdit.Date().Month(), dataEdit.Date().Day(),
								oldTime.Hour(), oldTime.Minute(), oldTime.Second(),
								0, oldTime.Location())
						},
					},
					Label{
						Text: LangValue("time") + ":",
					},

					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							NumberEdit{
								AssignTo: &hourEdit,
								MaxValue: 23,
								MinValue: 0,
								Value: float64(cfg.Time.Hour()),
								SpinButtonsVisible: true,
								OnValueChanged: func() {
									oldTime := cfg.Time
									cfg.Time = time.Date(
										oldTime.Year(), oldTime.Month(), oldTime.Day(),
										int(hourEdit.Value()),
										oldTime.Minute(), oldTime.Second(),
										0, oldTime.Location())
								},
							},
							Label{
								Text: LangValue("hour"),
							},
							NumberEdit{
								AssignTo: &minEdit,
								MaxValue: 59,
								MinValue: 0,
								Value: float64(cfg.Time.Minute()),
								SpinButtonsVisible: true,
								OnValueChanged: func() {
									oldTime := cfg.Time
									cfg.Time = time.Date(
										oldTime.Year(), oldTime.Month(), oldTime.Day(),
										oldTime.Hour(), int(minEdit.Value()),
										oldTime.Second(), 0, oldTime.Location())
								},
							},
							Label{
								Text: LangValue("minute"),
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
								AssignTo: &hourDelay,
								Model: HourSelectOptions(),
								OnCurrentIndexChanged: func() {
									add := StringToInt(hourDelay.Text())
									temp := oldcfgTime.Add(time.Duration(add) * time.Hour)
									hourEdit.SetValue(float64(temp.Hour()))
									minEdit.SetValue(float64(temp.Minute()))
								},
							},
							Label{
								Text: LangValue("hour"),
							},
							ComboBox{
								AssignTo: &minDelay,
								Model: MinuteSelectOptions(),
								OnCurrentIndexChanged: func() {
									add := StringToInt(minDelay.Text())
									temp := oldcfgTime.Add(time.Duration(add) * time.Minute)
									hourEdit.SetValue(float64(temp.Hour()))
									minEdit.SetValue(float64(temp.Minute()))
								},
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
							now := time.Now()
							if cfg.Mode == 2 && now.After(cfg.Time) {
								errInfo := fmt.Sprintf("%s [%s] < %s [%s]",
									LangValue("settingtime"), timeNowShow(cfg.Time),
									LangValue("timenow"), timeNowShow(now))
								ErrorBoxAction(dlg, errInfo)
								return
							}
							err := KeepCfgSet(cfg)
							if err != nil {
								ErrorBoxAction(dlg, err.Error())
								return
							}
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
		logs.Info("keep config dialog return %d", cnt)
	}
}
