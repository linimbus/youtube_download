package main

import (
	"github.com/astaxie/beego/logs"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"os/exec"
)

func OpenBrowserWeb(url string)  {
	cmd := exec.Command("rundll32","url.dll,FileProtocolHandler", url)
	err := cmd.Run()
	if err != nil {
		logs.Error("run cmd fail, %s", err.Error())
	}
}

var image1 walk.Image
var image2 walk.Image

func AboutAction( mw *walk.MainWindow ) {
	var ok    *walk.PushButton
	var about *walk.Dialog
	var err error

	if image1 == nil {
		image1 = IconLoadImageFromBox("sponsor1.jpg")
	}

	if image2 == nil {
		image2 = IconLoadImageFromBox("sponsor2.jpg")
	}

	_, err = Dialog{
		AssignTo:      &about,
		Title:         LangValue("about"),
		Icon:          walk.IconInformation(),
		MinSize:       Size{Width: 300, Height: 200},
		DefaultButton: &ok,
		Layout:  VBox{},
		Children: []Widget{
			TextLabel{
				Text: LangValue("aboutcontext"),
				MinSize:       Size{Width: 250, Height: 200},
				MaxSize:       Size{Width: 290, Height: 400},
			},
			Label{
				Text: LangValue("version") + ": "+ VersionGet(),
				TextAlignment: AlignCenter,
			},
			VSpacer{
				MinSize: Size{Height: 10},
			},
			Label{
				Text: LangValue("sponsor"),
				TextAlignment: AlignCenter,
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{
						MinSize: Size{Width: 10},
					},
					ImageView{
						ToolTipText: LangValue("alipay"),
						Image:    image1,
						MaxSize:  Size{80, 80},
					},
					HSpacer{
						MinSize: Size{Width: 10},
					},
					ImageView{
						ToolTipText: LangValue("wecartpay"),
						Image:    image2,
						MaxSize:  Size{80, 80},
					},
					HSpacer{
						MinSize: Size{Width: 10},
					},
				},
			},
			PushButton{
				Text:      "paypal.me",
				OnClicked: func() {
					OpenBrowserWeb("https://paypal.me/lixiangyun")
				},
			},
			PushButton{
				Text:      LangValue("officialweb"),
				OnClicked: func() {
					OpenBrowserWeb("https://github.com/lixiangyun/youtube_download")
				},
			},
			PushButton{
				Text:      LangValue("accpet"),
				OnClicked: func() { about.Cancel() },
			},
		},
	}.Run(mw)

	if err != nil {
		logs.Error(err.Error())
	}
}
