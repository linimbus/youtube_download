package main

import (
	"fmt"
	"github.com/astaxie/beego/logs"
	ytdl "github.com/lixiangyun/youtube_download/youtube/downloader"
	"github.com/lixiangyun/youtube_download/youtube"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

func Clipboard() (string, error) {
	text, err := walk.Clipboard().Text()
	if err != nil {
		logs.Error(err.Error())
		return "", fmt.Errorf("no any clipboard")
	}
	return text, nil
}

var mlock sync.Mutex

func VideoInfoGet(link string) (*youtube.Video, error) {
	httpclient, err := HttpClientGet(HttpProxyGet())
	if err != nil {
		logs.Error(err.Error())
		return nil, err
	}

	dl := ytdl.Downloader{
		OutputDir: DataDownLoadDirGet(),
	}
	dl.HTTPClient = httpclient.cli

	videoInfo, err := dl.GetVideo(link)
	if err != nil {
		logs.Error(err.Error())
		return nil, err
	}

	return videoInfo, nil
}

func StringCat(s string, flag string) string {
	idx := strings.Index(s, flag)
	if idx == -1 {
		return s
	}
	return s[:idx]
}

func StringToInt(s string) int {
	num, err := strconv.Atoi(s)
	if err != nil {
		logs.Error("string[%s] to int fail, %s",s , err.Error())
		return 0
	}
	return num
}

func videoToMode(info *youtube.Video) *VideoModel {
	video := NewVideoMode()
	video.Timestamp = GetTimeStamp()
	video.Duration = info.Duration
	video.Title = info.Title
	video.Author = info.Author

	for _, v := range info.Formats {
		video.items = append(video.items, &VideoFormat{
			ItagNo:   v.ItagNo,
			Quality:  v.Quality,
			Format:   StringCat(v.MimeType, ";"),
			MimeType: v.MimeType,
			FPS:      v.FPS,
			Width:    v.Width,
			Height:   v.Height,
			Length:   StringToInt(v.ContentLength),
		})
	}

	return video
}

func UpdateAction(link string, update *walk.PushButton, video *VideoModel) error {
	mlock.Lock()
	defer mlock.Unlock()

	_, err := url.Parse(link)
	if err != nil {
		logs.Error(err.Error())
		return err
	}

	update.SetText(LangValue("updating"))
	update.SetEnabled(false)

	defer update.SetText(LangValue("update"))
	defer update.SetEnabled(true)

	info, err := VideoInfoGet(link)
	if err != nil {
		logs.Error(err.Error())
		return err
	}

	video.Update(videoToMode(info))

	return nil
}

func DownLinkFromClipboard() string {
	link, err := Clipboard()
	if err != nil {
		logs.Error(err.Error())
		return ""
	}
	urls, err := url.Parse(link)
	if err != nil {
		logs.Error(err.Error())
		return ""
	}
	if -1 == strings.Index(urls.Hostname(),"youtube.com") {
		return ""
	}
	return link
}

func WebUrlInput(dlg *walk.Dialog, video *VideoModel) []Widget {
	var input *walk.LineEdit
	var update *walk.PushButton

	return []Widget{
		Label{
			Text: LangValue("downloadlink"),
		},
		LineEdit{
			AssignTo: &input,
			Text: DownLinkFromClipboard(),
		},
		PushButton{
			Text: LangValue("pastelink"),
			OnClicked: func() {
				input.SetText(DownLinkFromClipboard())
			},
		},
		PushButton{
			AssignTo: &update,
			Text: LangValue("update"),
			OnClicked: func() {
				err := UpdateAction(input.Text(), update, video)
				if err != nil {
					ErrorBoxAction(dlg, err.Error())
				}
			},
		},
	}
}

func AddJobOnce()  {
	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton

	//var protocal *walk.ComboBox
	//var using, auth *walk.RadioButton
	//var user, passwd, address, testurl *walk.LineEdit
	//var testbut *walk.PushButton

	video := NewVideoMode()

	cnt, err := Dialog{
		AssignTo: &dlg,
		Title: LangValue("adddownloadjob"),
		Icon: walk.IconInformation(),
		DefaultButton: &acceptPB,
		CancelButton: &cancelPB,
		Size: Size{mainWindowWidth, mainWindowHeight},
		MinSize: Size{mainWindowWidth, mainWindowHeight},
		Layout:  VBox{
			Alignment: AlignHNearVNear,
			MarginsZero: true,
			Margins: Margins{Left: 10, Top: 5, Bottom: 10, Right: 10},
		},
		Children: []Widget{
			Composite{
				Layout: HBox{
					Alignment: AlignHNearVNear,
				},
				Children: WebUrlInput(dlg, video),
			},
			Composite{
				Layout: VBox{
					Alignment: AlignHNearVNear,
				},
				Children: VideoWight(video),
 			},
		},
	}.Run(MainWindowsCtrl())
	if err != nil {
		logs.Error(err.Error())
	} else {
		logs.Info("add job dialog return %d", cnt)
	}
}
