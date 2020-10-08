package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/lixiangyun/youtube_download/youtube"
	"golang.org/x/net/http/httpproxy"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
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

func run() error {
	flag.Usage = func() {
		fmt.Println(usageString)
		flag.PrintDefaults()
	}

	usr, _ := user.Current()
	flag.StringVar(&outputFile, "o", "", "The output file")
	flag.StringVar(&outputDir, "d", filepath.Join(usr.HomeDir, "Movies", "youtubedr"),
		"The output directory.")
	flag.StringVar(&outputQuality, "q", "", "The output file quality (hd720, medium)")
	flag.StringVar(&httpProxy, "p", "http://127.0.0.1:8080", "The http proxy, e.g. 127.0.0.1:8080")
	flag.IntVar(&itag, "i", -1, "Specify itag number, e.g. 13, 17")
	flag.BoolVar(&info, "info", false, "show info of video")

	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.PrintDefaults()
		return nil
	}

	proxycfg := &httpproxy.Config{HTTPProxy: httpProxy, HTTPSProxy: httpProxy}

	proxyfunc = proxycfg.ProxyFunc()

	httpTransport := &http.Transport{
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		Proxy: ProxyFunc,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig : &tls.Config{
			InsecureSkipVerify: true,
		},
	}


	dl := ytdl.Downloader{
		OutputDir: outputDir,
	}

	dl.HTTPClient = &http.Client{Transport: httpTransport}

	arg := flag.Arg(0)

	video, err := dl.GetVideo(arg)
	if err != nil {
		return err
	}

	if info {
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
		return nil
	}

	fmt.Println("download to directory", outputDir)

	if outputQuality == "hd1080" {
		fmt.Println("check ffmpeg is installed....")
		ffmpegVersionCmd := exec.Command("ffmpeg", "-version")
		if err := ffmpegVersionCmd.Run(); err != nil {
			return fmt.Errorf("please check ffmpeg is installed correctly, err: %w", err)
		}
		return dl.DownloadWithHighQuality(context.Background(), outputFile, video, outputQuality)
	}

	var downloadFormat *youtube.Format

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

	return dl.Download(context.Background(), video, downloadFormat, outputFile)
}
