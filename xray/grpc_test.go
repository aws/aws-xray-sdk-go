package xray

import (
	"context"
	"encoding/json"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
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

func newGrpcClient(ctx context.Context, t *testing.T, lis *bufconn.Listener, opts ...grpc.DialOption) (client pb.TestServiceClient, closeFunc func()) {
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

type testCase struct {
	name string

	// responseErrorStatusCode makes the test suite call grpc method `PingError` to trigger the testing server to
	// return an error response.
	// If responseErrorStatusCode is codes.OK, the test suite call `Ping` to get a success response
	responseErrorStatusCode codes.Code

	expectedThrottle bool
	expectedError bool
	expectedFault bool
}

func (t testCase) isTestForSuccessResponse() bool {
	return t.responseErrorStatusCode == codes.OK
}

func (t testCase) getExpectedURL() string {
	if t.isTestForSuccessResponse() {
		return "grpc://bufnet/mwitkow.testproto.TestService/Ping"
	}
	return "grpc://bufnet/mwitkow.testproto.TestService/PingError"
}

func (t testCase) getExpectedContentLength() int {
	if t.isTestForSuccessResponse() {
		return proto.Size(&pb.PingResponse{Value: "something", Counter: 1})
	}
	return 0
}

func TestGrpcUnaryClientInterceptor(t *testing.T) {
	lis := newGrpcServer(
		t,
		grpc_middleware.WithUnaryServerChain(
			UnaryServerInterceptor(NewFixedSegmentNamer("test")),
		),
	)
	client, closeFunc := newGrpcClient(context.Background(), t, lis, grpc.WithUnaryInterceptor(UnaryClientInterceptor("bufnet")))
	defer closeFunc()

	testCases := []testCase{
		{
			name: "success response",
			responseErrorStatusCode: codes.OK,
			expectedThrottle: false,
			expectedError: false,
			expectedFault: false,
		},
		{
			name: "error response",
			responseErrorStatusCode: codes.Unauthenticated,
			expectedThrottle: false,
			expectedError: true,
			expectedFault: true,
		},
		{
			name: "throttle response",
			responseErrorStatusCode: codes.ResourceExhausted,
			expectedThrottle: true,
			expectedFault: true,
			expectedError: false,
		},
		{
			name: "fault response",
			responseErrorStatusCode: codes.Internal,
			expectedThrottle: false,
			expectedError: false,
			expectedFault: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, td := NewTestDaemon()
			defer td.Close()

			ctx2, root := BeginSegment(ctx, "Test")
			var err error
			if tc.isTestForSuccessResponse() {
				_, err = client.Ping(
					ctx2,
					&pb.PingRequest{
						Value:                "something",
						SleepTimeMs:          9999,
					},
				)
				require.NoError(t, err)
			} else {
				_, err = client.PingError(
					ctx2,
					&pb.PingRequest{Value: "something", ErrorCodeReturned: uint32(tc.responseErrorStatusCode)})
				require.Error(t, err)
			}
			root.Close(nil)

			seg, err := td.Recv()
			require.NoError(t, err)

			var subseg *Segment
			assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg))
			assert.Equal(t, "remote", subseg.Namespace)
			assert.Equal(t, tc.getExpectedURL(), subseg.HTTP.Request.URL)
			assert.Equal(t, false, subseg.HTTP.Request.XForwardedFor)
			assert.Equal(t, tc.expectedThrottle, subseg.Throttle)
			assert.Equal(t, tc.expectedError, subseg.Error)
			assert.Equal(t, tc.expectedFault, subseg.Fault)
			assert.Equal(t, tc.getExpectedContentLength(), subseg.HTTP.Response.ContentLength)
		})
	}
}

func TestUnaryServerInterceptorWithContext(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	lis := newGrpcServer(
		t,
		grpc_middleware.WithUnaryServerChain(
			UnaryServerInterceptorWithContext(ctx, NewFixedSegmentNamer("test")),
		),
	)
	client, closeFunc := newGrpcClient(context.Background(), t, lis)
	defer closeFunc()

	testCases := []testCase{
		{
			name:                    "success response",
			responseErrorStatusCode: codes.OK,
			expectedThrottle:        false,
			expectedError:           false,
			expectedFault:           false,
		},
		{
			name: "error response",
			responseErrorStatusCode: codes.Unauthenticated,
			expectedThrottle: false,
			expectedError: true,
			expectedFault: false,
		},
		{
			name: "throttle response",
			responseErrorStatusCode: codes.ResourceExhausted,
			expectedThrottle: true,
			expectedError: false,
			expectedFault: false,
		},
		{
			name: "fault response",
			responseErrorStatusCode: codes.Internal,
			expectedThrottle: false,
			expectedError: false,
			expectedFault: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.isTestForSuccessResponse() {
				_, err := client.Ping(
					context.Background(),
					&pb.PingRequest{Value: "something", SleepTimeMs: 9999},
				)
				require.NoError(t, err)
			} else {
				_, err := client.PingError(
					context.Background(),
					&pb.PingRequest{Value: "something", ErrorCodeReturned: uint32(tc.responseErrorStatusCode)},
				)
				require.Error(t, err)
			}


			seg, err := td.Recv()
			require.NoError(t, err)

			assert.Equal(t, tc.getExpectedURL(), seg.HTTP.Request.URL)
			assert.Equal(t, false, seg.HTTP.Request.XForwardedFor)
			assert.Regexp(t, regexp.MustCompile(`^grpc-go/`), seg.HTTP.Request.UserAgent)
			assert.Equal(t, "TestVersion", seg.Service.Version)
			assert.Equal(t, tc.expectedThrottle, seg.Throttle)
			assert.Equal(t, tc.expectedError, seg.Error)
			assert.Equal(t, tc.expectedFault, seg.Fault)
			assert.Equal(t, tc.getExpectedContentLength(), seg.HTTP.Response.ContentLength)
		})
	}
}

func TestUnaryServerAndClientInterceptor(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	lis := newGrpcServer(
		t,
		grpc_middleware.WithUnaryServerChain(
			UnaryServerInterceptorWithContext(ctx, NewFixedSegmentNamer("test")),
		),
	)
	client, closeFunc := newGrpcClient(context.Background(), t, lis, grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		md := metadata.Pairs(TraceIDHeaderKey, "Root=fakeid; Parent=reqid; Sampled=1")
		ctx = metadata.NewOutgoingContext(ctx, md)
		return invoker(ctx, method, req, reply, cc, opts...)
	}))
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
