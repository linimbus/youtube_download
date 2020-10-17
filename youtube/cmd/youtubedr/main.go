package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/lixiangyun/youtube_download/youtube"
	"golang.org/x/net/http/httpproxy"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	ytdl "github.com/lixiangyun/youtube_download/youtube/downloader"
	"github.com/olekukonko/tablewriter"
)

const usageString string = `Usage: youtubedr [OPTION] [URL]
Download a video from youtube.
Example: youtubedr -o "Campaign Diary".mp4 https://www.youtube.com/watch\?v\=XbNghLqsVwU
`

var (
	outputFile         string
	outputDir          string
	outputQuality      string
	httpProxy          string
	itag               int
	info               bool
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err)
		os.Exit(1)
	}
}

var proxyfunc func(reqURL *url.URL) (*url.URL, error)

func ProxyFunc(r *http.Request) (*url.URL, error)  {
	return proxyfunc(r.URL)
}

func newTransport(timeout int, tlscfg *tls.Config) *http.Transport {
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

func init()  {
	usr, _ := user.Current()

	flag.StringVar(&outputFile, "o", "", "The output file")
	flag.StringVar(&outputDir, "d", filepath.Join(usr.HomeDir, "Movies", "youtubedr"), "The output directory.")
	flag.StringVar(&httpProxy, "p", "http://127.0.0.1:8080", "The http proxy, e.g. 127.0.0.1:8080")
	flag.IntVar(&itag, "i", -1, "Specify itag number, e.g. 13, 17")
	flag.BoolVar(&info, "info", false, "show info of video")
}

const SLICE_SIZE = 128*1024

func httpClientGet(client *http.Client, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	value, err := ioutil.ReadAll(resp.Body)
	if err != nil && err != io.EOF {
		return value, err
	}
	return value, nil
}

func WriteFull(w io.Writer, body []byte) error {
	begin := 0
	for  {
		cnt, err := w.Write(body[begin:])
		if cnt > 0 {
			begin += cnt
		}
		if begin >= len(body) {
			return err
		}
		if err != nil {
			return err
		}
	}
}

func buildSliceUrl(URLs string, begin, size int) string {
	URLs = fmt.Sprintf("%s?range=%d-%d",URLs,begin, begin + size)
	return URLs
}

func downloadslice(wait *sync.WaitGroup,queue chan struct{}, client *http.Client, url string, call func([]byte) )  {
	for i:=0; i<1000; i++ {
		body, err := httpClientGet(client, url)
		if err == nil {
			fmt.Printf("url:%s, %d\n", url, len(body))

			call(body)
			break
		}
		fmt.Println(err.Error())
	}
	queue <- struct{}{}
	wait.Done()
}

var mlock sync.Mutex
var wg sync.WaitGroup
var queue chan struct{}

func downLoadWask(client *http.Client,URL string, offset int, size int, file *os.File ) {
	url := buildSliceUrl(URL, offset, size-1 )
	downloadslice(&wg, queue, client, url, func(bytes []byte) {
		mlock.Lock()
		file.Seek(int64(offset),0)
		err := WriteFull(file, bytes)
		if err != nil {
			fmt.Println(err.Error())
		}
		mlock.Unlock()
	})
}

func videoDownLoad(cli youtube.Client, format *youtube.Format) {
	length, err := strconv.Atoi(format.ContentLength)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	sliceCnt := length / SLICE_SIZE
	sliceEnd := length % SLICE_SIZE

	fmt.Printf("totalSize: %d, sliceCnt: %d", length, sliceCnt)

	file, err := os.Create("video.mp4")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	queue = make( chan struct{}, 100 )
	for i:=0; i < 20; i++ {
		queue <- struct{}{}
	}

	for i:= 0 ; i < sliceCnt ;i++  {
		<- queue
		offset := i*SLICE_SIZE
		wg.Add(1)
		go downLoadWask(cli.HTTPClient, format.URL, offset, SLICE_SIZE, file)
	}

	wg.Add(1)
	go downLoadWask(cli.HTTPClient, format.URL, sliceCnt*SLICE_SIZE, sliceEnd, file)

	wg.Wait()
	return
}

func run() error {
	flag.Parse()

	flag.Usage = func() {
		fmt.Println(usageString)
		flag.PrintDefaults()
	}

	proxycfg := &httpproxy.Config{HTTPProxy: httpProxy, HTTPSProxy: httpProxy}
	proxyfunc = proxycfg.ProxyFunc()

	httpTransport := newTransport(60, nil)
	httpTransport.Proxy = ProxyFunc

	dl := ytdl.Downloader{
		OutputDir: outputDir,
	}

	dl.HTTPClient = &http.Client{Transport: httpTransport}

	arg := "https://www.youtube.com/watch?v=g-LlyjdnjSM"

	video, err := dl.GetVideo(arg)
	if err != nil {
		return err
	}

	fmt.Printf("Title:    %s\n", video.Title)
	fmt.Printf("Author:   %s\n", video.Author)
	fmt.Printf("Duration: %v\n", video.Duration)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"itag", "quality", "MimeType"})

	for _, itag := range video.Formats {
		table.Append([]string{strconv.Itoa(itag.ItagNo), itag.Quality, itag.MimeType})
	}
	table.Render()

	fmt.Println("download to directory", outputDir)

	var downloadFormat *youtube.Format

	itag = 160
	if itag != -1 {
		for _, v := range video.Formats {
			if itag == v.ItagNo {
				fmt.Println("%v", v)
				downloadFormat = &v
				break
			}
		}
	}

	if downloadFormat == nil {
		downloadFormat = &video.Formats[0]
	}

	videoDownLoad(dl.Client, downloadFormat)

	return nil
}
