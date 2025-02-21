package xray

import (
	"context"
	"encoding/json"
	"net"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-xray-sdk-go/v2/header"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	pb "github.com/grpc-ecosystem/go-grpc-middleware/v2/testing/testpb"
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

func (s *testGRPCPingService) PingError(_ context.Context, req *pb.PingErrorRequest) (*pb.PingErrorResponse, error) {
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
	expectedError    bool
	expectedFault    bool
}

func (t testCase) isTestForSuccessResponse() bool {
	return t.responseErrorStatusCode == codes.OK
}

func (t testCase) getExpectedURL() string {
	if t.isTestForSuccessResponse() {
		return "grpc://bufnet/testing.testpb.v1.TestService/Ping"
	}
	return "grpc://bufnet/testing.testpb.v1.TestService/PingError"
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
		grpc.UnaryInterceptor(UnaryServerInterceptor()),
	)
	client, closeFunc := newGrpcClient(context.Background(), t, lis, grpc.WithUnaryInterceptor(UnaryClientInterceptor()))
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
			name:                    "error response",
			responseErrorStatusCode: codes.Unauthenticated,
			expectedThrottle:        false,
			expectedError:           true,
			expectedFault:           true,
		},
		{
			name:                    "throttle response",
			responseErrorStatusCode: codes.ResourceExhausted,
			expectedThrottle:        true,
			expectedFault:           true,
			expectedError:           false,
		},
		{
			name:                    "fault response",
			responseErrorStatusCode: codes.Internal,
			expectedThrottle:        false,
			expectedError:           false,
			expectedFault:           true,
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
						Value:       "something",
						SleepTimeMs: 9999,
					},
				)
				require.NoError(t, err)
			} else {
				_, err = client.PingError(
					ctx2,
					&pb.PingErrorRequest{Value: "something", ErrorCodeReturned: uint32(tc.responseErrorStatusCode)})
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
	t.Run("default namer", func(t *testing.T) {
		lis := newGrpcServer(
			t,
			grpc.UnaryInterceptor(UnaryServerInterceptor()),
		)
		client, closeFunc := newGrpcClient(
			context.Background(),
			t,
			lis,
			grpc.WithUnaryInterceptor(UnaryClientInterceptor()))
		defer closeFunc()

		ctx, td := NewTestDaemon()
		defer td.Close()
		ctx, root := BeginSegment(ctx, "Test")
		_, err := client.Ping(ctx, &pb.PingRequest{Value: "something", SleepTimeMs: 9999})
		assert.NoError(t, err)
		root.Close(nil)

		seg, err := td.Recv()
		require.NoError(t, err)

		var subseg *Segment
		assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg))
		assert.Equal(t, "testing.testpb.v1.TestService", subseg.Name)
		assert.Equal(t, "grpc://bufnet/testing.testpb.v1.TestService/Ping", subseg.HTTP.Request.URL)
	})
	t.Run("custom namer", func(t *testing.T) {
		lis := newGrpcServer(
			t,
			grpc.UnaryInterceptor(UnaryServerInterceptor()),
		)
		client, closeFunc := newGrpcClient(
			context.Background(),
			t,
			lis,
			grpc.WithUnaryInterceptor(
				UnaryClientInterceptor(
					WithSegmentNamer(NewFixedSegmentNamer("custom")))))
		defer closeFunc()

		ctx, td := NewTestDaemon()
		defer td.Close()
		ctx, root := BeginSegment(ctx, "Test")
		_, err := client.Ping(ctx, &pb.PingRequest{Value: "something", SleepTimeMs: 9999})
		assert.NoError(t, err)
		root.Close(nil)

		seg, err := td.Recv()
		require.NoError(t, err)

		var subseg *Segment
		assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg))
		assert.Equal(t, "custom", subseg.Name)
		assert.Equal(t, "grpc://bufnet/testing.testpb.v1.TestService/Ping", subseg.HTTP.Request.URL)
	})
}

func TestUnaryServerInterceptor(t *testing.T) {
	testCases := []testCase{
		{
			name:                    "success response",
			responseErrorStatusCode: codes.OK,
			expectedThrottle:        false,
			expectedError:           false,
			expectedFault:           false,
		},
		{
			name:                    "error response",
			responseErrorStatusCode: codes.Unauthenticated,
			expectedThrottle:        false,
			expectedError:           true,
			expectedFault:           false,
		},
		{
			name:                    "throttle response",
			responseErrorStatusCode: codes.ResourceExhausted,
			expectedThrottle:        true,
			expectedError:           false,
			expectedFault:           false,
		},
		{
			name:                    "fault response",
			responseErrorStatusCode: codes.Internal,
			expectedThrottle:        false,
			expectedError:           false,
			expectedFault:           true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// Ideally we shouldn't be creating a local test daemon for each test case.
			// However, how the shared channel in the test daemon doesn't guarantee
			// segment isolation across test cases. Therefore, for now a local test daemon
			// is a workaround.
			ctx, td := NewTestDaemon()
			defer td.Close()

			lis := newGrpcServer(
				t,
				grpc.UnaryInterceptor(
					UnaryServerInterceptor(
						WithRecorder(GetRecorder(ctx)),
						WithSegmentNamer(NewFixedSegmentNamer("test")))),
			)
			client, closeFunc := newGrpcClient(context.Background(), t, lis)
			defer closeFunc()

			var respHeaders metadata.MD
			if tc.isTestForSuccessResponse() {
				_, err := client.Ping(
					context.Background(),
					&pb.PingRequest{Value: "something", SleepTimeMs: 9999},
					grpc.Header(&respHeaders),
				)
				require.NoError(t, err)
			} else {
				_, err := client.PingError(
					context.Background(),
					&pb.PingErrorRequest{Value: "something", ErrorCodeReturned: uint32(tc.responseErrorStatusCode)},
					grpc.Header(&respHeaders),
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
			respTraceHeaderSlice := respHeaders[TraceIDHeaderKey]
			require.NotNil(t, respTraceHeaderSlice)
			require.Len(t, respTraceHeaderSlice, 1)
			respTraceHeader := header.FromString(respTraceHeaderSlice[0])
			assert.Equal(t, seg.TraceID, respTraceHeader.TraceID)
			assert.Equal(t, header.Unknown, respTraceHeader.SamplingDecision)
		})
	}

	// test that the interceptor by default will name the segment by the gRPC service name
	t.Run("default namer", func(t *testing.T) {
		ctx, td := NewTestDaemon()
		defer td.Close()

		lis := newGrpcServer(
			t,
			grpc.UnaryInterceptor(
				UnaryServerInterceptor(
					WithRecorder(GetRecorder(ctx)))),
		)
		client, closeFunc := newGrpcClient(context.Background(), t, lis)
		defer closeFunc()
		_, err := client.Ping(context.Background(), &pb.PingRequest{Value: "something", SleepTimeMs: 9999})
		assert.NoError(t, err)
		segment, err := td.Recv()
		assert.NoError(t, err)
		assert.Equal(t, "testing.testpb.v1.TestService", segment.Name)
	})

	t.Run("chained interceptor", func(t *testing.T) {
		ctx, td := NewTestDaemon()
		defer td.Close()

		lis := newGrpcServer(
			t,
			grpc.ChainUnaryInterceptor(
				func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
					resp, err := handler(ctx, req)
					assert.NoError(t, grpc.SetHeader(ctx, metadata.Pairs("foo", "bar")))
					return resp, err
				},
				UnaryServerInterceptor(WithRecorder(GetRecorder(ctx))),
			),
		)
		client, closeFunc := newGrpcClient(context.Background(), t, lis)
		defer closeFunc()
		var respHeaders metadata.MD
		_, err := client.Ping(context.Background(), &pb.PingRequest{Value: "something", SleepTimeMs: 9999}, grpc.Header(&respHeaders))
		require.NoError(t, err)
		assert.Equal(t, []string{"bar"}, respHeaders["foo"])
		assert.NotNil(t, respHeaders[TraceIDHeaderKey])
	})
}

func TestUnaryServerAndClientInterceptor(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	lis := newGrpcServer(
		t,
		grpc.UnaryInterceptor(
			UnaryServerInterceptor(
				WithRecorder(GetRecorder(ctx)),
				WithSegmentNamer(NewFixedSegmentNamer("test")))),
	)
	client, closeFunc := newGrpcClient(context.Background(), t, lis, grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = metadata.AppendToOutgoingContext(ctx, TraceIDHeaderKey, "Root=fakeid; Parent=reqid; Sampled=1")
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

func TestInferServiceName(t *testing.T) {
	assert.Equal(t, "com.example.Service", inferServiceName("/com.example.Service/method"))
}
