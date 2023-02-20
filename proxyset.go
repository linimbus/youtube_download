package main

import (
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func ProtocalOptions() []string {
	return []string{
		"HTTP","HTTPS","SOCK5",
	}
}

func TestUrlGet() string {
	url := DataStringValueGet("proxytesturl")
	if url == "" {
		url = "https://www.youtube.com"
	}
	return url
}

func TestUrlSet(url string)  {
	DataStringValueSet("proxytesturl", url)
}

func TestEngin(testurl string, item *HttpProxyOption) (time.Duration, error) {
	before := time.Now()

	if !IsConnect(item.Address, 5) {
		return 0, fmt.Errorf("remote address connnect %s fail", item.Address)
	}

	_, err := url.Parse(testurl)
	if err != nil {
		logs.Error("%s test url parse fail, %s", testurl, err.Error())
		return 0, err
	}

	httpclient, err := HttpClientGet(item)
	if err != nil {
		logs.Error(err.Error())
		return 0, err
	}

	req, err := http.NewRequest(http.MethodGet, testurl, nil)
	if err != nil {
		logs.Error(err.Error())
		return 0, err
	}

	rsp, err := httpclient.cli.Do(req)
	if err != nil {
		logs.Error(err.Error())
		return 0, err
	}

	defer rsp.Body.Close()
	_, err = ioutil.ReadAll(rsp.Body)
	if err != nil {
		logs.Error(err.Error())
		return 0, err
	}

	return time.Now().Sub(before), nil
}

func ProxySetDialog()  {
	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton

	var protocal *walk.ComboBox
	var using, auth *walk.RadioButton
	var user, passwd, address, testurl *walk.LineEdit
	var testbut *walk.PushButton

	curProxyOpt := HttpProxyGet()
	if curProxyOpt == nil {
		curProxyOpt = new(HttpProxyOption)
	}

	cnt, err := Dialog{
		AssignTo: &dlg,
		Title: LangValue("proxysetting"),
		Icon: ICON_Network_Flow,
		DefaultButton: &acceptPB,
		CancelButton: &cancelPB,
		Size: Size{300, 300},
		MinSize: Size{300, 300},
		Layout:  VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{
						Text: LangValue("usingproxy") + ":",
					},
					RadioButton{
						AssignTo: &using,
						OnBoundsChanged: func() {
							using.SetChecked(curProxyOpt.Using)
						},
						OnClicked: func() {
							using.SetChecked(!curProxyOpt.Using)
							curProxyOpt.Using = !curProxyOpt.Using
						},
					},

					Label{
						Text: LangValue("remoteaddress") + ":",
					},
					LineEdit{
						AssignTo: &address,
						Text: curProxyOpt.Address,
						OnEditingFinished: func() {
							curProxyOpt.Address = address.Text()
						},
					},
					Label{
						Text: LangValue("protocal") + ":",
					},
					ComboBox{
						AssignTo: &protocal,
						Model: ProtocalOptions(),
						Value: strings.ToUpper(curProxyOpt.Protocal),
						OnCurrentIndexChanged: func() {
							curProxyOpt.Protocal = strings.ToLower(protocal.Text())
						},
					},

					Label{
						Text: LangValue("whetherauth") + ":",
					},
					RadioButton{
						AssignTo: &auth,
						OnBoundsChanged: func() {
							auth.SetChecked(curProxyOpt.Auth)
						},
						OnClicked: func() {
							auth.SetChecked(!curProxyOpt.Auth)
							curProxyOpt.Auth = !curProxyOpt.Auth
							user.SetEnabled(curProxyOpt.Auth)
							passwd.SetEnabled(curProxyOpt.Auth)
						},
					},

					Label{
						Text: LangValue("user") + ":",
					},

					LineEdit{
						AssignTo: &user,
						Text: curProxyOpt.User,
						Enabled: curProxyOpt.Auth,
						OnEditingFinished: func() {
							curProxyOpt.User = user.Text()
						},
					},

					Label{
						Text: LangValue("password") + ":",
					},

					LineEdit{
						AssignTo: &passwd,
						Text: curProxyOpt.Passwd,
						Enabled: curProxyOpt.Auth,
						OnEditingFinished: func() {
							curProxyOpt.Passwd = passwd.Text()
						},
					},

					PushButton{
						AssignTo: &testbut,
						Text: LangValue("test"),
						OnClicked: func() {
							go func() {
								testbut.SetEnabled(false)
								delay, err := TestEngin(testurl.Text(), curProxyOpt)
								if err != nil {
									ErrorBoxAction(dlg, err.Error())
								} else {
									info := fmt.Sprintf("%s, %s %dms",
										LangValue("testpass"),
										LangValue("delay"), delay/time.Millisecond )
									InfoBoxAction(dlg, info)
								}
								testbut.SetEnabled(true)
							}()
						},
					},

					LineEdit{
						AssignTo: &testurl,
						Text: TestUrlGet(),
						OnEditingFinished: func() {
							TestUrlSet(testurl.Text())
						},
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						AssignTo: &acceptPB,
						Text:     LangValue("save"),
						OnClicked: func() {
							if curProxyOpt.Auth {
								if curProxyOpt.User == "" || curProxyOpt.Passwd == "" {
									ErrorBoxAction(dlg, LangValue("inputuserandpasswd"))
									return
								}
							}
							if curProxyOpt.Address == "" {
								ErrorBoxAction(dlg, LangValue("inputnameandaddress"))
								return
							}
							err := HttpProxySet(curProxyOpt)
							if err != nil {
								ErrorBoxAction(dlg, err.Error())
							} else {
								dlg.Accept()
							}
						},
					},
					PushButton{
						AssignTo:  &cancelPB,
						Text:      LangValue("cancel"),
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