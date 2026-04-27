package wework

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/andybalholm/brotli"
)

var jar *cookiejar.Jar

func init() {
	cjar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}

	jar = cjar
}

// BaseClient represents a base HTTP client with common configurations
type BaseClient struct {
	*http.Client
	headers http.Header
}

// NewBaseClient creates a new BaseClient instance with default configurations
func NewBaseClient() (*BaseClient, error) {
	client := &http.Client{
		Jar:     jar,
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &BaseClient{
		Client: client,
		headers: http.Header{
			"Accept":           []string{"application/json, text/plain, */*"},
			"Content-Type":     []string{"application/json"},
			"Request-Source":   []string{"com.wework.ondemand/WorkplaceOne/Prod/iOS/2.71.0(26.1)"},
			"WeWorkMemberType": []string{"2"},
			"Origin":           []string{"https://members.wework.com"},
			"User-Agent":       []string{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.3.1 Safari/605.1.15"},
			"Sec-Fetch-Mode":   []string{"cors"},
			"Sec-Fetch-Dest":   []string{"empty"},
			"Sec-Fetch-Site":   []string{"same-origin"},
			"fe-pg":            []string{"/workplaceone/content2/dashboard"},
			"Referer":          []string{"https://members.wework.com/workplaceone/content2/dashboard"},
			"Accept-Encoding":  []string{"identity"},
			"Accept-Language":  []string{"en-US,en;q=0.9"},
		},
	}, nil
}

// Do overrides the default Do method to add common headers
func (c *BaseClient) Do(req *http.Request) (*http.Response, error) {
	merged := c.headers.Clone()
	for key, values := range req.Header {
		merged.Del(key)
		for _, value := range values {
			merged.Add(key, value)
		}
	}
	merged.Set("Host", req.URL.Host)
	req.Header = merged

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}

	// Check that the server actually sent compressed data.
	// Some JS/Node transports may auto-decompress while still exposing the
	// original Content-Encoding header, so fall back to the original body if the
	// decompressor rejects the stream.
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		originalBody := resp.Body
		reader, err = gzip.NewReader(resp.Body)
		if err == nil {
			resp.Body = reader
		} else {
			resp.Body = originalBody
		}
	case "br":
		resp.Body = io.NopCloser(brotli.NewReader(resp.Body))

	default:
		// No compression, just return the original response body
	}

	return resp, err
}
