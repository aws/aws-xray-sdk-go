package xray

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/aws/aws-xray-sdk-go/internal/logger"

	"google.golang.org/grpc/codes"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/status"

	"github.com/aws/aws-xray-sdk-go/header"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryClientInterceptor provides gRPC unary client interceptor.
func UnaryClientInterceptor(clientInterceptorOptions ...ClientInterceptorOption) grpc.UnaryClientInterceptor {
	var option clientInterceptorOption
	for _, interceptorOption := range clientInterceptorOptions {
		interceptorOption.apply(&option)
	}

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		host := cc.Target()
		if option.host != nil {
			host = *option.host
		}
		var segmentName string
		if option.segmentNamer == nil {
			segmentName = inferServiceName(method)
		} else {
			segmentName = option.segmentNamer.Name(host)
		}
		return Capture(ctx, segmentName, func(ctx context.Context) error {
			seg := GetSegment(ctx)
			if seg == nil {
				return errors.New("failed to record gRPC transaction: segment cannot be found")
			}

			md := metadata.Pairs(TraceIDHeaderKey, seg.DownstreamHeader().String())
			ctx = metadata.NewOutgoingContext(ctx, md)

			seg.Lock()
			seg.Namespace = "remote"
			seg.GetHTTP().GetRequest().URL = "grpc://" + host + method
			seg.GetHTTP().GetRequest().Method = http.MethodPost
			seg.Unlock()

			err := invoker(ctx, method, req, reply, cc, opts...)

			recordContentLength(seg, reply)
			if err != nil {
				classifyErrorStatus(seg, err)
			}

			return err
		})
	}
}

// UnaryServerInterceptor provides gRPC unary server interceptor.
func UnaryServerInterceptor(serverInterceptorOptions ...ServerInterceptorOption) grpc.UnaryServerInterceptor {
	var interceptorOptions serverInterceptorOption
	for _, options := range serverInterceptorOptions {
		options.apply(&interceptorOptions)
	}

	var cfg *Config
	if interceptorOptions.ctx != nil {
		cfg = GetRecorder(interceptorOptions.ctx)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		md, ok := metadata.FromIncomingContext(ctx)

		var traceID string
		if ok && len(md.Get(TraceIDHeaderKey)) == 1 {
			traceID = md.Get(TraceIDHeaderKey)[0]
		}
		traceHeader := header.FromString(traceID)

		var host string

		if len(md.Get(":authority")) == 1 {
			host = md.Get(":authority")[0]
		}
		requestURL := url.URL{
			Scheme: "grpc",
			Host:   host,
			Path:   info.FullMethod,
		}

		var name string
		if interceptorOptions.segmentNamer == nil {
			name = inferServiceName(info.FullMethod)
		} else {
			name = interceptorOptions.segmentNamer.Name(host)
		}

		if cfg != nil {
			ctx = context.WithValue(ctx, RecorderContextKey{}, cfg)
		}

		var seg *Segment
		ctx, seg = NewSegmentFromHeader(ctx, name, &http.Request{
			Host:   host,
			URL:    &requestURL,
			Method: http.MethodPost,
		}, traceHeader)
		defer seg.Close(nil)

		seg.Lock()
		seg.GetHTTP().GetRequest().ClientIP, seg.GetHTTP().GetRequest().XForwardedFor = clientIPFromGrpcMetadata(md)
		seg.GetHTTP().GetRequest().URL = requestURL.String()
		seg.GetHTTP().GetRequest().Method = http.MethodPost
		if len(md.Get("user-agent")) == 1 {
			seg.GetHTTP().GetRequest().UserAgent = md.Get("user-agent")[0]
		}
		seg.Unlock()

		resp, err = handler(ctx, req)
		if err != nil {
			classifyErrorStatus(seg, err)
		}
		recordContentLength(seg, resp)
		if headerErr := addResponseTraceHeader(ctx, seg, traceHeader); headerErr != nil {
			logger.Debug("fail to send the grpc trace header")
		}

		return resp, err
	}
}

func classifyErrorStatus(seg *Segment, err error) {
	seg.Lock()
	defer seg.Unlock()
	grpcStatus, ok := status.FromError(err)
	if !ok {
		seg.Fault = true
		return
	}
	switch grpcStatus.Code() {
	case codes.Canceled, codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.PermissionDenied, codes.Unauthenticated, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange:
		seg.Error = true
	case codes.Unknown, codes.DeadlineExceeded, codes.Unimplemented, codes.Internal, codes.Unavailable, codes.DataLoss:
		seg.Fault = true
	case codes.ResourceExhausted:
		seg.Throttle = true
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

func recordContentLength(seg *Segment, reply interface{}) {
	seg.Lock()
	defer seg.Unlock()
	if protoMessage, isProtoMessage := reply.(proto.Message); isProtoMessage {
		seg.GetHTTP().GetResponse().ContentLength = proto.Size(protoMessage)
	}
}

func addResponseTraceHeader(ctx context.Context, seg *Segment, incomingTraceHeader *header.Header) error {
	var respHeader bytes.Buffer
	respHeader.WriteString("Root=")
	respHeader.WriteString(seg.TraceID)
	if incomingTraceHeader.SamplingDecision == header.Requested {
		respHeader.WriteString(";Sampled=")
		respHeader.WriteString(strconv.Itoa(btoi(seg.Sampled)))
	}

	headers := metadata.New(map[string]string{
		TraceIDHeaderKey: respHeader.String(),
	})
	return grpc.SendHeader(ctx, headers)
}

func inferServiceName(fullMethodName string) string {
	fullMethodName = fullMethodName[1:]
	return fullMethodName[:strings.Index(fullMethodName, "/")]
}

type ServerInterceptorOption interface {
	apply(option *serverInterceptorOption)
}

type serverInterceptorOption struct {
	ctx          context.Context
	segmentNamer SegmentNamer
}

func newFuncServerInterceptorOption(f func(option *serverInterceptorOption)) ServerInterceptorOption {
	return funcServerInterceptorOption{f: f}
}

type funcServerInterceptorOption struct {
	f func(option *serverInterceptorOption)
}

func (f funcServerInterceptorOption) apply(option *serverInterceptorOption) {
	f.f(option)
}

// ServerInterceptorWithContext makes the interceptor inherit xray.Config associated with ctx.
func ServerInterceptorWithContext(ctx context.Context) ServerInterceptorOption {
	return newFuncServerInterceptorOption(func(option *serverInterceptorOption) {
		option.ctx = ctx
	})
}

// ServerInterceptorWithSegmentNamer makes the interceptor use the segment namer to name the segment.
func ServerInterceptorWithSegmentNamer(sn SegmentNamer) ServerInterceptorOption {
	return newFuncServerInterceptorOption(func(option *serverInterceptorOption) {
		option.segmentNamer = sn
	})
}

// ClientInterceptorOption allows customize ClientUnaryInterceptor.
type ClientInterceptorOption interface {
	apply(option *clientInterceptorOption)
}

type clientInterceptorOption struct {
	segmentNamer SegmentNamer
	host         *string
}

func newFuncClientInterceptorOption(f func(option *clientInterceptorOption)) ClientInterceptorOption {
	return funcClientInterceptorOption{f: f}
}

type funcClientInterceptorOption struct {
	f func(option *clientInterceptorOption)
}

func (f funcClientInterceptorOption) apply(option *clientInterceptorOption) {
	f.f(option)
}

// ClientInterceptorWithSegmentNamer makes the interceptor use the segment namer to name the segment.
func ClientInterceptorWithSegmentNamer(sn SegmentNamer) ClientInterceptorOption {
	return newFuncClientInterceptorOption(func(option *clientInterceptorOption) {
		option.segmentNamer = sn
	})
}

// ClientInterceptorWithHost overrides the host of the URL recorded to the segment. It also overrides the host passed into
// the custom segment namer.
func ClientInterceptorWithHost(host string) ClientInterceptorOption {
	return newFuncClientInterceptorOption(func(option *clientInterceptorOption) {
		option.host = &host
	})
}
