package xray

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func UnaryClientInterceptor(name string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, seg := BeginSegment(ctx, name)
		ctx = metadata.AppendToOutgoingContext(ctx, TraceIDHeaderKey, seg.DownstreamHeader().String())
		err := invoker(ctx, method, req, reply, cc, opts...)
		seg.Close(err)
		return err
	}
}
