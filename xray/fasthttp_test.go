package xray

import (
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestFastHTTPHandler(t *testing.T) {
	ctx1, td := NewTestDaemon()
	cfg := GetRecorder(ctx1)
	defer td.Close()

	b := `{"body": "content"}`
	req := fasthttp.Request{}
	req.SetBodyString(b)
	req.SetRequestURI("/path")
	req.SetHost("localhost")
	req.Header.SetContentType("application/json")
	req.Header.SetContentLength(len(b))
	req.Header.SetMethod(http.MethodPost)
	req.Header.SetUserAgent("UA_test")

	rc := &fasthttp.RequestCtx{}
	rc.Init(&req, nil, nil)

	remoteAddr := &net.TCPAddr{
		IP:   []byte{1, 2, 3, 5},
		Port: 0,
	}
	rc.SetRemoteAddr(remoteAddr)

	fh := NewFastHTTP(cfg)
	handler := fh.Handler(NewFixedSegmentNamer("test"), func(ctx *fasthttp.RequestCtx) {})
	handler(rc)

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	assert.Equal(t, http.MethodPost, seg.HTTP.Request.Method)
	assert.Equal(t, "http://localhost/path", seg.HTTP.Request.URL)
	assert.Equal(t, "1.2.3.5", seg.HTTP.Request.ClientIP)
	assert.Equal(t, "UA_test", seg.HTTP.Request.UserAgent)
}
