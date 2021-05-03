package xray

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

// serve serves http request using provided fasthttp handler
func serveFasthttp(handler fasthttp.RequestHandler, req *http.Request) (*http.Response, error) {
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go func() {
		err := fasthttp.Serve(ln, handler)
		if err != nil {
			panic(fmt.Errorf("failed to serve: %v", err))
		}
	}()

	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return ln.Dial()
			},
		},
	}

	return client.Do(req)
}

func TestFastHTTPHandler(t *testing.T) {
	ctx1, td := NewTestDaemon()
	cfg := GetRecorder(ctx1)
	defer td.Close()

	handler := func(ctx *fasthttp.RequestCtx) {
		fmt.Fprint(ctx, "It's working!")
	}

	r, err := http.NewRequest("POST", "http://test/", nil)
	if err != nil {
		t.Error(err)
	}

	r.Header.Set("User-Agent", "UA_test")

	fh := NewFastHTTP(cfg)
	res, err := serveFasthttp(fh.Handler(NewFixedSegmentNamer("test"), handler), r)
	if err != nil {
		t.Error(err)
	}

	defer res.Body.Close()
	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, http.MethodPost, seg.HTTP.Request.Method)
	assert.Equal(t, "http://test/", seg.HTTP.Request.URL)

	// fasthttputil force string `pipe` to RemoteAddr.
	assert.Equal(t, "pipe", seg.HTTP.Request.ClientIP)
	assert.Equal(t, "UA_test", seg.HTTP.Request.UserAgent)
}
