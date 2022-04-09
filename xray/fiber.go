package xray

import (
	"context"
	"github.com/aws/aws-xray-sdk-go/header"
	"github.com/gofiber/fiber/v2"
)

type FiberHandler interface {
	Handler(SegmentNamer, fiber.Handler) fiber.Handler
}

type fiberHandler struct {
	cfg *Config
}

// NewFiberInstrumentor returns a new FiberHandler that
// provides a Handler function to satisfy the fiber.Handler
// interface.
func NewFiberInstrumentor(cfg *Config) FiberHandler {
	return &fiberHandler{
		cfg: cfg,
	}
}

// Handler wraps the provided fiber.Handler.
func (h *fiberHandler) Handler(sn SegmentNamer, handler fiber.Handler) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		auxCtx := context.Background()
		if h.cfg != nil {
			auxCtx = context.WithValue(context.Background(), fasthttpContextConfigKey, h.cfg)
			ctx.Locals(fasthttpContextConfigKey, h.cfg)
		}

		name := sn.Name(ctx.Hostname())
		traceHeader := header.FromString(ctx.Get(TraceIDHeaderKey))

		req, err := fasthttpToNetHTTPRequest(ctx.Context())
		if err != nil {
			return err
		}

		_, seg := NewSegmentFromHeader(auxCtx, name, req, traceHeader)
		defer seg.Close(nil)

		ctx.Locals(fasthttpContextKey, seg)
		httpCaptureRequest(seg, req)
		return fiberTrace(seg, handler, ctx, traceHeader)
	}
}

func fiberTrace(seg *Segment, handler fiber.Handler, ctx *fiber.Ctx, traceHeader *header.Header) error {
	ctx.Set(TraceIDHeaderKey, generateTraceIDHeaderValue(seg, traceHeader))
	handlerErr := handler(ctx)

	seg.Lock()
	seg.GetHTTP().GetResponse().ContentLength = ctx.Context().Response.Header.ContentLength()
	seg.Unlock()
	HttpCaptureResponse(seg, ctx.Response().StatusCode())
	return handlerErr
}
