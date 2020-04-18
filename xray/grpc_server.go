package xray

import (
	"context"
	"github.com/aws/aws-xray-sdk-go/header"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func UnaryServerInterceptor(name string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "missing metadata")
		}
		traceId := md.Get(TraceIDHeaderKey)
		traceIdHeader := ""
		if len(traceId) == 1 {
			traceIdHeader = traceId[0]
		}
		theader := header.FromString(traceIdHeader)
		ctx, seg := NewSegmentFromHeader(ctx, name, nil, theader)
		_ = seg.AddMetadata("method", info.FullMethod)
		m, err := handler(ctx, req)
		seg.Close(err)
		return m, err
	}
}
