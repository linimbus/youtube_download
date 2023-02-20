package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/astaxie/beego/logs"
	"github.com/kkdai/youtube"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type BatchCfg struct {
	Keep    bool
	All     bool
	Highest bool
	Hd1080p bool
	Hd720   bool
	Medium  bool
	Small   bool
}

func ParseWebUrlToList(ctx string) []string {
	list := strings.Split(ctx, "\n")
	var output []string
	for _, v := range list {
		v = strings.ReplaceAll(v, "\r", "")
		_, err := url.Parse(v)
		if err != nil {
			logs.Error(err.Error())
			continue
		}
		if v == "" {
			continue
		}
		output = append(output, v)
	}
	logs.Info("parse web url list : %v", output)
	return output
}

func QualtiyItagNoGet(formats youtube.FormatList, qualtiy string) int {
	var indexList []int
	for i, format := range formats {
		if strings.ToLower(format.Quality) == qualtiy {
			indexList = append(indexList, i)
		}
	}
	if len(indexList) == 0 {
		return -1
	}
	for _, v := range indexList {
		if -1 != strings.Index(formats[v].MimeType, "mp4a") {
			return formats[v].ItagNo
		}
		if -1 != strings.Index(formats[v].MimeType, "mp4") {
			return formats[v].ItagNo
		}
	}
	return formats[indexList[0]].ItagNo
}

func AutioItagNoGet(formats youtube.FormatList) int {
	var itagNoList []int
	for _, format := range formats {
		mimeType := strings.ToLower(format.MimeType)
		if -1 != strings.Index(mimeType, "audio") {
			if -1 != strings.Index(mimeType, "mp4a") {
				return format.ItagNo
			}
			itagNoList = append(itagNoList, format.ItagNo)
		}
	}
	if len(itagNoList) == 0 {
		return -1
	}
	return itagNoList[0]
}

func HighestItagNoGet(formats youtube.FormatList) int {
	var height int
	var itagNo int
	for _, format := range formats {
		if format.Height > height {
			itagNo = format.ItagNo
			height = format.Height
		}
	}
	return itagNo
}

func ParseItagnoList(video *youtube.Video, cfg *BatchCfg) []int {
	var output []int

	if cfg.Small {
		temp := QualtiyItagNoGet(video.Formats, "small")
		if temp != -1 {
			output = append(output, temp)
		}
	}

	if cfg.Medium {
		temp := QualtiyItagNoGet(video.Formats, "medium")
		if temp != -1 {
			output = append(output, temp)
		}
	}

	if cfg.Hd720 {
		temp := QualtiyItagNoGet(video.Formats, "hd720")
		if temp != -1 {
			output = append(output, temp)
		}
	}

	if cfg.Hd1080p {
		temp := QualtiyItagNoGet(video.Formats, "hd1080")
		if temp != -1 {
			output = append(output, temp)
		}
	}

	if cfg.Highest || len(output) == 0 {
		temp := HighestItagNoGet(video.Formats)
		if temp != -1 {
			output = append(output, temp)
		}
	}

	temp := AutioItagNoGet(video.Formats)
	if temp != -1 {
		output = append(output, temp)
	}

	return output
}

func BatchAddJob(weburl []string, cfg *BatchCfg) error {
	if cfg.All {
		cfg.Highest = true
		cfg.Hd1080p = true
		cfg.Hd720 = true
		cfg.Medium = true
		cfg.Small = true
	}

	httpclient, err := HttpClientGet(HttpProxyGet())
	if err != nil {
		logs.Error(err.Error())
		return err
	}
	client := &youtube.Client{HTTPClient: httpclient.cli}

	var errInfo string

	for _, url := range weburl {
		video, err := WebVideoGet(client, url)
		if err != nil {
			logs.Error(err.Error())
			return err
		}
		itagNoList := ParseItagnoList(video, cfg)
		err = JobAdd(video, itagNoList, url, cfg.Keep)
		if err != nil {
			logs.Error(err.Error())
			errInfo += fmt.Sprintln(url)
		}
	}

	if errInfo != "" {
		return fmt.Errorf("%s", errInfo)
	}

	return nil
}

func AddJobBatch() {
	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton
	var addBut, cancelBut *walk.PushButton

	var now, keep *walk.RadioButton
	var allCbx, highCbx, hd1080pCbx, hd720pCbx, mediumCbx, smallCbx *walk.CheckBox
	var addWebUrl *walk.TextEdit

	cfg := BatchCfg{Highest: true}

	cnt, err := Dialog{
		AssignTo:      &dlg,
		Title:         LangValue("batchadd"),
		Icon:          ICON_TOOL_DOWNLOAD,
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		Size:          Size{400, 500},
		MinSize:       Size{400, 500},
		Layout: VBox{
			Alignment: AlignHNearVNear,
		},
		Children: []Widget{
			Label{
				Text: LangValue("downloadlist"),
			},
			TextEdit{
				VScroll:  true,
				Text:     "",
				AssignTo: &addWebUrl,
			},
			Label{
				Text: LangValue("downloadoptions"),
			},
			Composite{
				ToolTipText: LangValue("quality"),
				Layout:      HBox{},
				Children: []Widget{
					CheckBox{
						AssignTo: &allCbx,
						Text:     LangValue("all"),
						OnClicked: func() {
							cfg.All = allCbx.Checked()
						},
					},
					CheckBox{
						AssignTo: &highCbx,
						Text:     "highest",
						Checked:  true,
						OnClicked: func() {
							cfg.Highest = highCbx.Checked()
						},
					},
					CheckBox{
						AssignTo: &hd1080pCbx,
						Text:     "hd1080p",
						OnClicked: func() {
							cfg.Hd1080p = hd1080pCbx.Checked()
						},
					},
					CheckBox{
						AssignTo: &hd720pCbx,
						Text:     "hd720p",
						OnClicked: func() {
							cfg.Hd720 = hd720pCbx.Checked()
						},
					},
					CheckBox{
						AssignTo: &mediumCbx,
						Text:     "medium(360p)",
						OnClicked: func() {
							cfg.Medium = mediumCbx.Checked()
						},
					},
					CheckBox{
						AssignTo: &smallCbx,
						Text:     "small(240p)",
						OnClicked: func() {
							cfg.Small = smallCbx.Checked()
						},
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
							now.SetChecked(!cfg.Keep)
						},
						OnClicked: func() {
							cfg.Keep = false
							now.SetChecked(!cfg.Keep)
							keep.SetChecked(cfg.Keep)
						},
					},
					RadioButton{
						AssignTo: &keep,
						Text:     LangValue("appointmentdownload"),
						OnClicked: func() {
							cfg.Keep = true
							now.SetChecked(!cfg.Keep)
							keep.SetChecked(cfg.Keep)
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
						AssignTo: &addBut,
						Text:     LangValue("add"),
						OnClicked: func() {
							webUrlList := ParseWebUrlToList(addWebUrl.Text())
							if len(webUrlList) == 0 {
								ErrorBoxAction(dlg, LangValue("batchaddurl"))
								return
							}
							addBut.SetEnabled(false)
							cancelBut.SetEnabled(false)
							go func() {
								err := BatchAddJob(webUrlList, &cfg)

								addBut.SetEnabled(true)
								cancelBut.SetEnabled(true)

								if err != nil {
									ErrorBoxAction(dlg, fmt.Sprintf("%s %s",
										err.Error(),
										LangValue("batchaddfail")))
									return
								}
								dlg.Accept()
							}()
						},
					},
					PushButton{
						AssignTo: &cancelBut,
						Text:     LangValue("cancel"),
						OnClicked: func() {
							dlg.Cancel()
						},
					},
					HSpacer{},
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
