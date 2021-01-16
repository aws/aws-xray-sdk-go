package xray

import (
	"context"
	"errors"
	"net/http/httptrace"
	"strings"

	"github.com/aws/aws-xray-sdk-go/header"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryClientInterceptor provides gRPC unary client interceptor.
func UnaryClientInterceptor(host string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return Capture(ctx, host, func(ctx context.Context) error {
			seg := GetSegment(ctx)
			if seg == nil {
				return errors.New("failed to record gRPC transaction: segment cannot be found")
			}

			ct, e := NewClientTrace(ctx)
			if e != nil {
				return e
			}
			md := metadata.Pairs(TraceIDHeaderKey, seg.DownstreamHeader().String())
			ctx2 := metadata.NewOutgoingContext(httptrace.WithClientTrace(ctx, ct.httpTrace), md)

			seg.Lock()
			seg.Namespace = "remote"
			seg.GetHTTP().GetRequest().URL = "grpc://" + host + method
			seg.GetHTTP().GetRequest().Method = method
			seg.Unlock()

			err := invoker(ctx2, method, req, reply, cc, opts...)

			if err != nil {
				seg.Lock()
				seg.Error = true
				seg.Unlock()
				ct.subsegments.GotConn(nil, err)
			}

			return err
		})
	}
}

// UnaryServerInterceptor provides gRPC unary server interceptor.
func UnaryServerInterceptor(ctx context.Context, sn SegmentNamer) grpc.UnaryServerInterceptor {
	cfg := GetRecorder(ctx)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		md, ok := metadata.FromIncomingContext(ctx)

		var traceID string
		if ok && len(md.Get(TraceIDHeaderKey)) == 1 {
			traceID = md.Get(TraceIDHeaderKey)[0]
		}
		traceHeader := header.FromString(traceID)

		var host, url string
		if len(md.Get(":authority")) == 1 {
			host = md.Get(":authority")[0]
			url = "grpc://" + host + info.FullMethod
		}
		name := sn.Name(host)

		ctx = context.WithValue(ctx, RecorderContextKey{}, cfg)

		var seg *Segment
		ctx, seg = NewSegmentFromHeader(ctx, name, nil, traceHeader)
		defer seg.Close(nil)

		seg.Lock()
		seg.GetHTTP().GetRequest().ClientIP, seg.GetHTTP().GetRequest().XForwardedFor = clientIPFromGrpcMetadata(md)
		seg.GetHTTP().GetRequest().URL = url
		seg.GetHTTP().GetRequest().Method = info.FullMethod
		if len(md.Get("user-agent")) == 1 {
			seg.GetHTTP().GetRequest().UserAgent = md.Get("user-agent")[0]
		}
		seg.Unlock()

		resp, err = handler(ctx, req)
		if err != nil {
			seg.Lock()
			seg.Error = true
			seg.Unlock()
		}

		return resp, err
	}
}

func clientIPFromGrpcMetadata(md metadata.MD) (string, bool) {
	if len(md.Get("x-forwarded-for")) != 1 {
		return "", false
	}
	forwardedFor := md.Get("x-forwarded-for")[0]
	if forwardedFor != "" {
		return strings.TrimSpace(strings.Split(forwardedFor, ",")[0]), true
	}
	return "", false
}
