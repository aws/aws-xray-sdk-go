package xrayfasthttp

import (
	"net/http"
	"net/url"

	"github.com/aws/aws-xray-sdk-go/header"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/valyala/fasthttp"
)

func Handler(sn xray.SegmentNamer, h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		name := sn.Name(string(ctx.Request.Host()))
		traceHeader := header.FromString(string(ctx.Request.Header.Peek(xray.TraceIDHeaderKey)))

		r := &http.Request{
			Host:       string(ctx.Host()),
			RequestURI: string(ctx.RequestURI()),
			Method:     string(ctx.Method()),
			RemoteAddr: ctx.RemoteAddr().String(),
		}

		rURL, err := url.ParseRequestURI(r.RequestURI)
		if err != nil {
			ctx.Logger().Printf("cannot parse requestURI %q: %s", r.RequestURI, err)
			ctx.Error("Internal Server Error", fasthttp.StatusInternalServerError)
			return
		}

		r.URL = rURL

		hdr := make(http.Header)
		ctx.Request.Header.VisitAll(func(k, v []byte) {
			sk := string(k)
			sv := string(v)
			switch sk {
			case "Transfer-Encoding":
				r.TransferEncoding = append(r.TransferEncoding, sv)
			default:
				hdr.Set(sk, sv)
			}
		})

		r.Header = hdr
		r.TLS = ctx.TLSConnectionState()

		_, seg := xray.NewSegmentFromHeader(ctx, name, r, traceHeader)
		defer seg.Close(nil)

		ctx.SetUserValue(xray.ContextKey, seg)
		httpTrace(seg, h, ctx, traceHeader)
	}
}

func httpTrace(seg *xray.Segment, h fasthttp.RequestHandler, ctx *fasthttp.RequestCtx, traceHeader *header.Header) {
	traceID := seg.TraceHeaderID(traceHeader)
	ctx.Request.Header.Set(xray.TraceIDHeaderKey, traceID)
	h(ctx)
	seg.HTTPCapture(ctx.Response.StatusCode(), ctx.Response.Header.ContentLength())
}
