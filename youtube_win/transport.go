package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/logs"
	"golang.org/x/net/http/httpproxy"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
	"time"
)

const HTTP_CLIENT_TIME_OUT = 60

func NewTransport(timeout int, tlscfg *tls.Config) *http.Transport {
	tmout := time.Duration(timeout) * time.Second
	return &http.Transport{
		TLSClientConfig: tlscfg,
		DialContext: (&net.Dialer{
			Timeout:   tmout,
			KeepAlive: tmout,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          1000,
		IdleConnTimeout:       3*tmout,
		TLSHandshakeTimeout:   tmout,
		ExpectContinueTimeout: 5*time.Second }
}

type HttpClient struct {
	cli http.Client
	transport *http.Transport

	timeout int
	httpproxycfg  *httpproxy.Config
	httpproxyHandler func(reqURL *url.URL) (*url.URL, error)
}

func (http *HttpClient)ProxyFunc(r *http.Request) (*url.URL, error)  {
	return http.httpproxyHandler(r.URL)
}

func (http *HttpClient)HttpProxyInit(proxy *HttpProxyOption) error {
	var proxyurl string
	if proxy.Auth {
		proxyurl = fmt.Sprintf("%s://%s:%s@%s", proxy.Protocal,
			url.QueryEscape(proxy.User), url.QueryEscape(proxy.Passwd), proxy.Address)
	} else {
		proxyurl = fmt.Sprintf("%s://%s", proxy.Protocal, proxy.Address)
	}
	http.httpproxycfg = &httpproxy.Config{HTTPProxy: proxyurl, HTTPSProxy: proxyurl}
	http.httpproxyHandler = http.httpproxycfg.ProxyFunc()
	http.transport.Proxy = http.ProxyFunc
	return nil
}

func (http *HttpClient)Sock5Init(proxycfg *HttpProxyOption) error {
	var auth *proxy.Auth
	if proxycfg.Auth {
		auth = &proxy.Auth{
			User: proxycfg.User,
			Password: proxycfg.Passwd,
		}
	}
	dialer, err := proxy.SOCKS5("tcp", proxycfg.Address, auth, &net.Dialer{})
	if err != nil {
		logs.Error("dial sock5 fail, %s", err.Error)
		return err
	}
	http.transport.Dial = dialer.Dial
	return nil
}

func HttpClientGet() (*HttpClient, error) {
	timeout := DataIntValueGet("httpclienttimeout")
	if timeout == 0 {
		timeout = HTTP_CLIENT_TIME_OUT
	}

	tlscfg, err := TlsConfigClient("")
	if err != nil {
		logs.Error(err.Error())
		return nil, err
	}

	httpClient := new(HttpClient)
	httpClient.transport = NewTransport(int(timeout), tlscfg)

	proxy := HttpProxyGet()
	if proxy == nil {
		return httpClient, nil
	}

	if proxy.Protocal == "http" || proxy.Protocal == "https" {
		err = httpClient.HttpProxyInit(proxy)
	} else if proxy.Protocal == "sock5" {
		err = httpClient.Sock5Init(proxy)
	}

	if err != nil {
		logs.Error(err.Error())
		return nil, err
	}

	return httpClient, nil
}

type HttpProxyOption struct {
	Protocal string // http、https、sock5
	Address  string
	Auth     bool
	User     string
	Passwd   string
}

func HttpProxyGet() *HttpProxyOption {
	value := DataStringValueGet("httpproxy")
	if value == "" {
		return nil
	}
	var opts HttpProxyOption
	err := json.Unmarshal([]byte(value), &opts)
	if err != nil {
		logs.Error(err.Error())
		return nil
	}
	return &opts
}

func HttpProxySet(opts *HttpProxyOption) error {
	value, err := json.Marshal(opts)
	if err != nil {
		logs.Error(err.Error())
		return err
	}
	return DataStringValueSet("httpproxy", string(value))
}


