package main

import (
	"encoding/json"
	"fmt"
	. "github.com/lxn/walk/declarative"
	"github.com/astaxie/beego/logs"
	"github.com/lxn/walk"
	"os"
)

type BaseSetting struct {
	HomeDir       string
	Speed         int
	MultiThreaded int
}

var baseSettingCache *BaseSetting

func BaseSettingGet() BaseSetting {
	if baseSettingCache == nil {
		value := DataStringValueGet("basesetting")
		if value != "" {
			var out BaseSetting
			err := json.Unmarshal([]byte(value), &out)
			if err != nil {
				logs.Error(err.Error())
			} else {
				baseSettingCache = &out
				return out
			}
		}
		return BaseSetting{
			HomeDir: defaultVideoDir(),
			Speed: 0, MultiThreaded: 10,
		}
	}
	return *baseSettingCache
}

func BaseSettingSet(cfg BaseSetting) error  {
	value, err := json.Marshal(cfg)
	if err != nil {
		logs.Error(err.Error())
		return err
	}

	err = DataStringValueSet("basesetting", string(value))
	if err != nil {
		logs.Error(err.Error())
		return err
	}

	baseSettingCache = &cfg

	return nil
}

func DirDialog(dlg *walk.Dialog, oldPath string) string {
	dlgDir := new(walk.FileDialog)

	dlgDir.FilePath = oldPath
	dlgDir.Title = LangValue("selectdownloaddir")

	bool, err := dlgDir.ShowBrowseFolder(dlg)
	if err != nil {
		logs.Error(err.Error())
		return ""
	}

	if bool {
		return dlgDir.FilePath
	}
	return ""
}

func Setting()  {
	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton
	var downloadDir *walk.TextEdit
	var speed, multi *walk.NumberEdit

	cfg := BaseSettingGet()
	oldDir := cfg.HomeDir

	cnt, err := Dialog{
		AssignTo: &dlg,
		Title: LangValue("basesetting"),
		Icon: ICON_TOOL_SETTING,
		DefaultButton: &acceptPB,
		CancelButton: &cancelPB,
		MinSize: Size{400, 100},
		Size: Size{400, 200},
		Layout: VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2, MarginsZero: true},
				Children: []Widget{
					Label{
						Text: LangValue("downloaddir") + ":",
					},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							TextEdit{
								VScroll: true,
								CompactHeight: true,
								AssignTo: &downloadDir,
								Text: cfg.HomeDir,
								OnTextChanged: func() {
									dir := downloadDir.Text()
									if dir == "" {
										ErrorBoxAction(dlg, LangValue("inputdirectory"))
										downloadDir.SetText(oldDir)

										return
									}
									_, err := os.Stat(dir)
									if err != nil {
										errInfo := fmt.Sprintf("\"%s\"%s",
											dir, LangValue("directorynotexist") )
										ErrorBoxAction(dlg, errInfo)

										downloadDir.SetText(oldDir)
										return
									}
									cfg.HomeDir = dir
								},
							},
							PushButton{
								MaxSize: Size{Width: 20},
								Text: "...",
								OnClicked: func() {
									dir := DirDialog(dlg, oldDir)
									if dir != "" {
										downloadDir.SetText(dir)
										cfg.HomeDir = dir
									}
								},
							},
						},
					},
					Label{
						Text: LangValue("downloadsetting") + ":",
					},
					Composite{
						Layout: HBox{MarginsZero: true, Alignment: AlignHNearVNear},
						Children: []Widget{
							NumberEdit{
								AssignTo: &speed,
								SpinButtonsVisible: true,
								Value: float64(cfg.Speed),
								MinValue: float64(0),
								MaxValue: float64(1000),
								Suffix: " MB/s",
								OnValueChanged: func() {
									cfg.Speed = int(speed.Value())
								},
							},
							Label{
								Text: LangValue("speedlimit") + "[0-1000]",
							},
							HSpacer{
								MaxSize: Size{Width: 10},
							},
							NumberEdit{
								AssignTo: &multi,
								SpinButtonsVisible: true,
								Value: float64(cfg.MultiThreaded),
								MinValue: float64(1),
								MaxValue: float64(100),
								OnValueChanged: func() {
									cfg.MultiThreaded = int(multi.Value())
								},
							},
							Label{
								Text: LangValue("multithreaded") + "[1-100]",
							},
						},
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						Text: LangValue("save"),
						OnClicked: func() {
							err := BaseSettingSet(cfg)
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
		logs.Info("base setting dialog return %d", cnt)
	}
}
