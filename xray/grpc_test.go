package xray

import (
	"context"
	"encoding/json"
	"net"
	"regexp"
	"sync"
	"testing"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	pb "github.com/grpc-ecosystem/go-grpc-middleware/testing/testproto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type testGRPCPingService struct {
	counter int32
	mut     sync.Mutex

	pb.TestServiceServer
}

func (s *testGRPCPingService) Ping(_ context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	time.Sleep(time.Duration(req.SleepTimeMs) * time.Millisecond)

	s.mut.Lock()
	s.counter++
	counter := s.counter
	s.mut.Unlock()

	return &pb.PingResponse{
		Value:   req.Value,
		Counter: counter,
	}, nil
}

func (s *testGRPCPingService) PingError(_ context.Context, req *pb.PingRequest) (*pb.Empty, error) {
	code := codes.Code(req.ErrorCodeReturned)
	return nil, status.Errorf(code, "Userspace error.")
}

func newGrpcServer(t *testing.T, opts ...grpc.ServerOption) *bufconn.Listener {
	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)

	s := grpc.NewServer(opts...)
	pb.RegisterTestServiceServer(s, &testGRPCPingService{})
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Fatal(err)
		}
	}()

	return lis
}

func newGrpcClient(t *testing.T, ctx context.Context, lis *bufconn.Listener, opts ...grpc.DialOption) (client pb.TestServiceClient, closeFunc func()) {
	var bufDialer = func(ctx context.Context, address string) (net.Conn, error) {
		return lis.Dial()
	}

	opts = append(opts, grpc.WithContextDialer(bufDialer), grpc.WithInsecure())

	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		opts...,
	)
	if err != nil {
		t.Fatal(err)
	}
	closeFunc = func() {
		if err := conn.Close(); err != nil {
			t.Fatal(err)
		}
	}
	client = pb.NewTestServiceClient(conn)
	return
}

func TestGrpcUnaryClientInterceptor(t *testing.T) {
	lis := newGrpcServer(
		t,
		grpc_middleware.WithUnaryServerChain(
			UnaryServerInterceptor(context.Background(), NewFixedSegmentNamer("test")),
		),
	)
	client, closeFunc := newGrpcClient(
		t,
		context.Background(),
		lis,
		grpc.WithUnaryInterceptor(UnaryClientInterceptor("bufnet")),
	)
	defer closeFunc()

	t.Run("success response", func(t *testing.T) {
		ctx, td := NewTestDaemon()
		defer td.Close()

		ctx2, root := BeginSegment(ctx, "Test")
		_, err := client.Ping(
			ctx2,
			&pb.PingRequest{Value: "something", SleepTimeMs: 9999},
		)
		root.Close(nil)
		if !assert.NoError(t, err) {
			return
		}

		seg, err := td.Recv()
		if !assert.NoError(t, err) {
			return
		}

		var subseg *Segment
		assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg))
		assert.Equal(t, "remote", subseg.Namespace)
		assert.Equal(t, "grpc://bufnet/mwitkow.testproto.TestService/Ping", subseg.HTTP.Request.URL)
		assert.Equal(t, false, subseg.HTTP.Request.XForwardedFor)
		assert.False(t, subseg.Throttle)
		assert.False(t, subseg.Error)
		assert.False(t, subseg.Fault)
	})

	t.Run("error response", func(t *testing.T) {
		ctx, td := NewTestDaemon()
		defer td.Close()

		ctx2, root := BeginSegment(ctx, "Test")
		_, err := client.PingError(
			ctx2,
			&pb.PingRequest{Value: "something", ErrorCodeReturned: uint32(codes.Internal)},
		)
		root.Close(nil)
		if !assert.Error(t, err) {
			return
		}

		seg, err := td.Recv()
		if !assert.NoError(t, err) {
			return
		}

		var subseg *Segment
		assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg))
		assert.Equal(t, "remote", subseg.Namespace)
		assert.Equal(t, "grpc://bufnet/mwitkow.testproto.TestService/PingError", subseg.HTTP.Request.URL)
		assert.Equal(t, false, subseg.HTTP.Request.XForwardedFor)
		assert.False(t, subseg.Throttle)
		assert.True(t, subseg.Error)
		assert.False(t, seg.Fault)
	})
}

func TestUnaryServerInterceptor(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	lis := newGrpcServer(
		t,
		grpc_middleware.WithUnaryServerChain(
			UnaryServerInterceptor(ctx, NewFixedSegmentNamer("test")),
		),
	)
	client, closeFunc := newGrpcClient(t, context.Background(), lis)
	defer closeFunc()

	t.Run("success response", func(t *testing.T) {
		_, err := client.Ping(
			context.Background(),
			&pb.PingRequest{Value: "something", SleepTimeMs: 9999},
		)

		if !assert.NoError(t, err) {
			return
		}

		seg, err := td.Recv()
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, "grpc://bufnet/mwitkow.testproto.TestService/Ping", seg.HTTP.Request.URL)
		assert.Equal(t, false, seg.HTTP.Request.XForwardedFor)
		assert.Regexp(t, regexp.MustCompile(`^grpc-go/`), seg.HTTP.Request.UserAgent)
		assert.Equal(t, "TestVersion", seg.Service.Version)
	})

	t.Run("error response", func(t *testing.T) {
		_, err := client.PingError(
			context.Background(),
			&pb.PingRequest{Value: "something", ErrorCodeReturned: uint32(codes.Internal)},
		)

		if !assert.Error(t, err) {
			return
		}

		seg, err := td.Recv()
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, "grpc://bufnet/mwitkow.testproto.TestService/PingError", seg.HTTP.Request.URL)
		assert.Equal(t, false, seg.HTTP.Request.XForwardedFor)
		assert.Regexp(t, regexp.MustCompile(`^grpc-go/`), seg.HTTP.Request.UserAgent)
		assert.Equal(t, "TestVersion", seg.Service.Version)
		assert.False(t, seg.Throttle)
		assert.True(t, seg.Error)
		assert.False(t, seg.Fault)
	})
}

func TestUnaryServerAndClientInterceptor(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	lis := newGrpcServer(
		t,
		grpc_middleware.WithUnaryServerChain(
			UnaryServerInterceptor(ctx, NewFixedSegmentNamer("test")),
		),
	)
	client, closeFunc := newGrpcClient(
		t,
		context.Background(),
		lis,
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			md := metadata.Pairs(TraceIDHeaderKey, "Root=fakeid; Parent=reqid; Sampled=1")
			ctx = metadata.NewOutgoingContext(ctx, md)
			return invoker(ctx, method, req, reply, cc, opts...)
		}),
	)
	defer closeFunc()

	_, err := client.Ping(
		context.Background(),
		&pb.PingRequest{Value: "something", SleepTimeMs: 9999},
	)

	if !assert.NoError(t, err) {
		return
	}

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "fakeid", seg.TraceID)
	assert.Equal(t, "reqid", seg.ParentID)
	assert.Equal(t, true, seg.Sampled)
	assert.Equal(t, "TestVersion", seg.Service.Version)
}
