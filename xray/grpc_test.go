package xray

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_testing "github.com/grpc-ecosystem/go-grpc-middleware/testing"
	pb_testproto "github.com/grpc-ecosystem/go-grpc-middleware/testing/testproto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

type grpcBaseTestSuite struct {
	*grpc_testing.InterceptorTestSuite
}

func newGrpcBaseTestSuite(t *testing.T) *grpcBaseTestSuite {
	return &grpcBaseTestSuite{
		InterceptorTestSuite: &grpc_testing.InterceptorTestSuite{
			TestService: &grpc_testing.TestPingService{T: t},
		},
	}
}

func TestGrpcClientSuit(t *testing.T) {
	b := newGrpcBaseTestSuite(t)
	b.InterceptorTestSuite.ServerOpts = []grpc.ServerOption{
		grpc_middleware.WithUnaryServerChain(
			UnaryServerInterceptor(context.Background(), NewFixedSegmentNamer("test")),
		),
	}

	b.InterceptorTestSuite.ClientOpts = []grpc.DialOption{
		grpc.WithUnaryInterceptor(UnaryClientInterceptor("localhost")),
	}

	suite.Run(t, &grpcClientTestSuite{b})
}

type grpcClientTestSuite struct {
	*grpcBaseTestSuite
}

func (s *grpcClientTestSuite) TestUnaryClientInterceptor() {
	ctx, td := NewTestDaemon()
	defer td.Close()

	ctx2, root := BeginSegment(ctx, "Test")
	_, err := s.Client.Ping(
		ctx2,
		&pb_testproto.PingRequest{Value: "something", SleepTimeMs: 9999},
	)
	root.Close(nil)
	if !assert.NoError(s.T(), err) {
		return
	}

	seg, err := td.Recv()
	if !assert.NoError(s.T(), err) {
		return
	}

	var subseg *Segment
	assert.NoError(s.T(), json.Unmarshal(seg.Subsegments[0], &subseg))
	assert.Equal(s.T(), "remote", subseg.Namespace)
	assert.Equal(s.T(), "grpc://localhost/mwitkow.testproto.TestService/Ping", subseg.HTTP.Request.URL)
	assert.Equal(s.T(), false, subseg.HTTP.Request.XForwardedFor)
	assert.False(s.T(), subseg.Throttle)
	assert.False(s.T(), subseg.Error)
	assert.False(s.T(), subseg.Fault)
}

func (s *grpcClientTestSuite) TestUnaryClientInterceptorWithError() {
	ctx, td := NewTestDaemon()
	defer td.Close()

	ctx2, root := BeginSegment(ctx, "Test")
	_, err := s.Client.PingError(
		ctx2,
		&pb_testproto.PingRequest{Value: "something", ErrorCodeReturned: uint32(codes.Internal)},
	)
	root.Close(nil)
	if !assert.Error(s.T(), err) {
		return
	}

	seg, err := td.Recv()
	if !assert.NoError(s.T(), err) {
		return
	}

	var subseg *Segment
	assert.NoError(s.T(), json.Unmarshal(seg.Subsegments[0], &subseg))
	assert.Equal(s.T(), "remote", subseg.Namespace)
	assert.Equal(s.T(), "grpc://localhost/mwitkow.testproto.TestService/PingError", subseg.HTTP.Request.URL)
	assert.Equal(s.T(), false, subseg.HTTP.Request.XForwardedFor)
	assert.False(s.T(), subseg.Throttle)
	assert.True(s.T(), subseg.Error)
	assert.False(s.T(), seg.Fault)
}

func TestGrpcServerSuit(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	b := newGrpcBaseTestSuite(t)
	b.InterceptorTestSuite.ServerOpts = []grpc.ServerOption{
		grpc_middleware.WithUnaryServerChain(
			UnaryServerInterceptor(ctx, NewFixedSegmentNamer("test")),
		),
	}

	suite.Run(t, &grpcServerTestSuite{b, td})
}

type grpcServerTestSuite struct {
	*grpcBaseTestSuite
	td *TestDaemon
}

func (s *grpcServerTestSuite) TestUnaryServerInterceptor() {
	_, err := s.Client.Ping(
		s.DeadlineCtx(time.Now().Add(3*time.Second)),
		&pb_testproto.PingRequest{Value: "something", SleepTimeMs: 9999},
	)

	if !assert.NoError(s.T(), err) {
		return
	}

	seg, err := s.td.Recv()
	if !assert.NoError(s.T(), err) {
		return
	}

	assert.Equal(s.T(), "grpc://localhost/mwitkow.testproto.TestService/Ping", seg.HTTP.Request.URL)
	assert.Equal(s.T(), false, seg.HTTP.Request.XForwardedFor)
	assert.Regexp(s.T(), regexp.MustCompile(`^grpc-go/`), seg.HTTP.Request.UserAgent)
	assert.Equal(s.T(), "TestVersion", seg.Service.Version)
}

func (s *grpcServerTestSuite) TestUnaryServerInterceptorWithError() {
	_, err := s.Client.PingError(
		s.DeadlineCtx(time.Now().Add(3*time.Second)),
		&pb_testproto.PingRequest{Value: "something", ErrorCodeReturned: uint32(codes.Internal)},
	)

	if !assert.Error(s.T(), err) {
		return
	}

	seg, err := s.td.Recv()
	if !assert.NoError(s.T(), err) {
		return
	}

	assert.Equal(s.T(), "grpc://localhost/mwitkow.testproto.TestService/PingError", seg.HTTP.Request.URL)
	assert.Equal(s.T(), false, seg.HTTP.Request.XForwardedFor)
	assert.Regexp(s.T(), regexp.MustCompile(`^grpc-go/`), seg.HTTP.Request.UserAgent)
	assert.Equal(s.T(), "TestVersion", seg.Service.Version)
	assert.False(s.T(), seg.Throttle)
	assert.True(s.T(), seg.Error)
	assert.False(s.T(), seg.Fault)
}

func TestGrpcServerWithParentTracerSuit(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	b := newGrpcBaseTestSuite(t)
	b.InterceptorTestSuite.ServerOpts = []grpc.ServerOption{
		grpc_middleware.WithUnaryServerChain(
			UnaryServerInterceptor(ctx, NewFixedSegmentNamer("test")),
		),
	}

	b.InterceptorTestSuite.ClientOpts = []grpc.DialOption{
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			md := metadata.Pairs(TraceIDHeaderKey, "Root=fakeid; Parent=reqid; Sampled=1")
			ctx = metadata.NewOutgoingContext(ctx, md)
			return invoker(ctx, method, req, reply, cc, opts...)
		}),
	}

	suite.Run(t, &grpcServerWithParentTracerSuit{b, td})
}

type grpcServerWithParentTracerSuit struct {
	*grpcBaseTestSuite
	td *TestDaemon
}

func (s *grpcServerWithParentTracerSuit) TestUnaryServerInterceptor() {
	_, err := s.Client.Ping(
		s.DeadlineCtx(time.Now().Add(3*time.Second)),
		&pb_testproto.PingRequest{Value: "something", SleepTimeMs: 9999},
	)

	if !assert.NoError(s.T(), err) {
		return
	}

	seg, err := s.td.Recv()
	if !assert.NoError(s.T(), err) {
		return
	}

	assert.Equal(s.T(), "fakeid", seg.TraceID)
	assert.Equal(s.T(), "reqid", seg.ParentID)
	assert.Equal(s.T(), true, seg.Sampled)
	assert.Equal(s.T(), "TestVersion", seg.Service.Version)
}
