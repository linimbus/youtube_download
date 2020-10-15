package main

import (
	"context"
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
	"time"
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

func VideoInfoGet(link string, dir string) (*youtube.Video, error) {
	httpclient, err := HttpClientGet(HttpProxyGet())
	if err != nil {
		logs.Error(err.Error())
		return nil, err
	}

	dl := ytdl.Downloader{
		OutputDir: dir,
	}
	dl.HTTPClient = httpclient.cli

	var video *youtube.Video

	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
		video, err = dl.GetVideoContext(ctx, link)
		cancel()
		if err != nil {
			logs.Error(err.Error())
			continue
		} else {
			break
		}
	}

	if err != nil {
		return nil, err
	}

	return video, nil
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

func UpdateAction(link string, update *walk.PushButton, video *VideoModel) error {
	mlock.Lock()
	defer mlock.Unlock()

	_, err := url.Parse(link)
	if err != nil {
		logs.Error(err.Error())
		return err
	}

	oldText := update.Text()

	update.SetText(LangValue("updating"))
	update.SetEnabled(false)

	defer update.SetText(oldText)
	defer update.SetEnabled(true)

	dir := BaseSettingGet().HomeDir
	info, err := VideoInfoGet(link, dir)
	if err != nil {
		logs.Error(err.Error())
		return err
	}

	video.DownloadDir = dir
	video.WebUrl = link
	video.Update(info)

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

func WebUrlInput(dlg **walk.Dialog, video *VideoModel) []Widget {
	var input *walk.TextEdit
	var update *walk.PushButton

	link := DownLinkFromClipboard()
	if link != "" {
		go func() {
			for  {
				if input != nil && input.Visible() {
					break
				}
				time.Sleep(100*time.Millisecond)
			}
			for  {
				if update != nil && update.Visible() {
					break
				}
				time.Sleep(100*time.Millisecond)
			}

			err := UpdateAction(input.Text(), update, video)
			if err != nil {
				ErrorBoxAction(*dlg, err.Error())
			}
		}()
	}

	return []Widget{
		Label{
			Text: LangValue("downloadlink"),
		},
		TextEdit{
			CompactHeight: true,
			AssignTo: &input,
			VScroll: true,
			Text: link,
		},
		PushButton{
			AssignTo: &update,
			Text: LangValue("pastelink"),
			OnClicked: func() {
				input.SetText(DownLinkFromClipboard())
				go func() {
					err := UpdateAction(input.Text(), update, video)
					if err != nil {
						ErrorBoxAction(*dlg, err.Error())
					}
				}()
			},
		},
	}
}

func AddJobOptionGet(video *VideoModel) []Widget {
	var all, hd4k, hd2k, hd1080p, hd720p, audio *walk.PushButton
	var allFlag, hd4kFlag, hd2kFlag, hd1080Flag, hd720Flag, audioFlag bool

	return []Widget{
		PushButton{
			AssignTo: &all,
			Text: LangValue("all"),
			OnClicked: func() {
				all.SetChecked(!allFlag)
				allFlag = !allFlag

				for _, v := range video.items {
					v.checked = allFlag
				}
				video.Flash()
			},
		},
		PushButton{
			AssignTo: &hd4k,
			Text: "hd2160(4K)",
			OnClicked: func() {
				hd4k.SetChecked(!hd4kFlag)
				hd4kFlag = !hd4kFlag

				for _, v := range video.items {
					if v.Quality == "hd2160" {
						v.checked = hd4kFlag
					}
				}
				video.Flash()
			},
		},
		PushButton{
			AssignTo: &hd2k,
			Text: "hd1440(2K)",
			OnClicked: func() {
				hd2k.SetChecked(!hd2kFlag)
				hd2kFlag = !hd2kFlag

				for _, v := range video.items {
					if v.Quality == "hd1440" {
						v.checked = hd2kFlag
					}
				}
				video.Flash()
			},
		},
		PushButton{
			AssignTo: &hd1080p,
			Text: "hd1080(1080p)",
			OnClicked: func() {
				hd1080p.SetChecked(!hd1080Flag)
				hd1080Flag = !hd1080Flag

				for _, v := range video.items {
					if v.Quality == "hd1080" {
						v.checked = hd1080Flag
					}
				}
				video.Flash()
			},
		},
		PushButton{
			AssignTo: &hd720p,
			Text: "hd720(720p)",
			OnClicked: func() {
				hd720p.SetChecked(!hd720Flag)
				hd720Flag = !hd720Flag

				for _, v := range video.items {
					if v.Quality == "hd720" {
						v.checked = hd720Flag
					}
				}
				video.Flash()
			},
		},
		PushButton{
			AssignTo: &audio,
			Text: "Audio",
			OnClicked: func() {
				audio.SetChecked(!audioFlag)
				audioFlag = !audioFlag

				for _, v := range video.items {
					if strings.Index(v.Format, "audio") == -1 {
						continue
					}
					if  strings.Index(v.MimeType, "mp4a") == -1 {
						continue
					}
					v.checked = audioFlag
				}
				video.Flash()
			},
		},
		HSpacer{

		},
	}
}

func DownloadOptionGet(video *VideoModel) []Widget {
	var now, keep *walk.RadioButton

	return []Widget{
		RadioButton{
			AssignTo: &now,
			Text: LangValue("downloadnow"),
			OnBoundsChanged: func() {
				now.SetChecked(!video.Keep)
			},
			OnClicked: func() {
				video.Keep = false
				now.SetChecked(!video.Keep)
				keep.SetChecked(video.Keep)
			},
		},
		RadioButton{
			AssignTo: &keep,
			Text: LangValue("appointmentdownload"),
			OnClicked: func() {
				video.Keep = true
				keep.SetChecked(video.Keep)
				now.SetChecked(!video.Keep)
			},
		},
	}
}

func addJobToTask(v *VideoModel) error {
	var itagno []int
	for _, v := range v.items {
		if v.checked {
			itagno = append(itagno, v.ItagNo)
		}
	}
	if len(itagno) == 0 {
		return fmt.Errorf("no select!")
	}

	return JobAdd(v.info, itagno, v.WebUrl, v.Keep)
}

func AddJobOnce()  {
	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton
	var addBut, cancelBut *walk.PushButton

	video := NewVideoMode()

	cnt, err := Dialog{
		AssignTo: &dlg,
		Title: LangValue("adddownloadjob"),
		Icon: ICON_TOOL_ADD,
		DefaultButton: &acceptPB,
		CancelButton: &cancelPB,
		Size: Size{700, 500},
		MinSize: Size{700, 500},
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
				Children: WebUrlInput(&dlg, video),
			},
			Composite{
				Layout: VBox{
					Alignment: AlignHNearVNear,
				},
				Children: VideoWight(video),
 			},
 			Composite{
				Layout: HBox{
					Alignment: AlignHNearVNear,
				},
				Children: AddJobOptionGet(video),
			},
			Composite{
				Layout: HBox{
					Alignment: AlignHNearVNear,
				},
				Children: DownloadOptionGet(video),
			},
			Composite{
				Layout: HBox{
					Alignment: AlignHNearVNear,
				},
				Children: []Widget{
					PushButton{
						AssignTo: &addBut,
						Text: LangValue("add"),
						OnClicked: func() {
							addBut.SetEnabled(false)
							cancelBut.SetEnabled(false)

							go func() {
								err := addJobToTask(video)

								addBut.SetEnabled(true)
								cancelBut.SetEnabled(true)

								if err != nil {
									ErrorBoxAction(dlg, err.Error())
									return
								}
								dlg.Accept()
							}()
						},
					},
					PushButton{
						AssignTo: &cancelBut,
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
