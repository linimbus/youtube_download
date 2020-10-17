package youtube

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

// Client offers methods to download video metadata and video streams.
type Client struct {
	// Debug enables debugging output through log package
	Debug bool

	// HTTPClient can be used to set a custom HTTP client.
	// If not set, http.DefaultClient will be used
	HTTPClient *http.Client
}

// GetVideo fetches video metadata
func (c *Client) GetVideo(url string) (*Video, error) {
	return c.GetVideoContext(context.Background(), url)
}

// GetVideoContext fetches video metadata with a context
func (c *Client) GetVideoContext(ctx context.Context, url string) (*Video, error) {
	id, err := extractVideoID(url)
	if err != nil {
		return nil, fmt.Errorf("extractVideoID failed: %w", err)
	}

	// Circumvent age restriction to pretend access through googleapis.com
	eurl := "https://youtube.googleapis.com/v/" + id
	body, err := c.httpGetBodyBytes(ctx, "https://youtube.com/get_video_info?video_id="+id+"&eurl="+eurl)
	if err != nil {
		return nil, err
	}

	v := &Video{
		ID: id,
	}

	return v, v.parseVideoInfo(string(body))
}

// GetStream returns the HTTP response for a specific format
func (c *Client) GetStream(video *Video, format *Format) (*http.Response, error) {
	return c.GetStreamContext(context.Background(), video, format)
}

// GetStreamContext returns the HTTP response for a specific format with a context
func (c *Client) GetStreamContext(ctx context.Context, video *Video, format *Format) (*http.Response, error) {
	url, err := c.GetStreamURLContext(ctx, video, format)
	if err != nil {
		return nil, err
	}

	return c.httpGet(ctx, url)
}

// GetStreamURL returns the url for a specific format
func (c *Client) GetStreamURL(video *Video, format *Format) (string, error) {
	return c.GetStreamURLContext(context.Background(), video, format)
}

// GetStreamURL returns the url for a specific format with a context
func (c *Client) GetStreamURLContext(ctx context.Context, video *Video, format *Format) (string, error) {
	if format.URL != "" {
		return format.URL, nil
	}

	cipher := format.Cipher
	if cipher == "" {
		return "", ErrCipherNotFound
	}

	return c.decipherURL(ctx, video.ID, cipher)
}

func (c *Client)GetStreamContextLangth(ctx context.Context, video *Video, format *Format) (int64, error) {
	length, err := strconv.Atoi(format.ContentLength)
	if err == nil {
		return int64(length), nil
	}
	rsp, err := c.GetStreamContext(ctx, video, format)
	if err != nil {
		return 0, err
	}
	defer rsp.Body.Close()

	format.ContentLength = fmt.Sprintf("%d", rsp.ContentLength)
	return rsp.ContentLength, nil
}

func GetSliceUrls(urls string, offset, size int64) string {
	urls = fmt.Sprintf("%s?range=%d-%d", urls, offset, offset + size - 1)
	return urls
}

func (c *Client) GetSliceStreamContext(ctx context.Context, video *Video, format *Format, offset, size int64) ([]byte, error) {
	urls, err := c.GetStreamURLContext(ctx, video, format)
	if err != nil {
		return nil, err
	}

	body, err := c.HttpGet(ctx, GetSliceUrls(urls, offset, size))
	if err != nil {
		return nil, err
	}
	if len(body) == int(size) {
		return body, nil
	}

	return body, fmt.Errorf("slice stream rsponse body [%d != %d]", len(body), size )
}

func (c *Client) HttpGet(ctx context.Context, url string) ([]byte, error)  {
	rsp, err := c.httpGet(ctx, url)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	values, err := ioutil.ReadAll(rsp.Body)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return values, nil
}

// httpGet does a HTTP GET request, checks the response to be a 200 OK and returns it
func (c *Client) httpGet(ctx context.Context, url string) (resp *http.Response, err error) {
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	if c.Debug {
		log.Println("GET", url)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, ErrUnexpectedStatusCode(resp.StatusCode)
	}

	return
}

// httpGetBodyBytes reads the whole HTTP body and returns it
func (c *Client) httpGetBodyBytes(ctx context.Context, url string) ([]byte, error) {
	resp, err := c.httpGet(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
