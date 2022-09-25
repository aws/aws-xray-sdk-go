package xray

import (
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFiberHandler(t *testing.T) {
	ctx1, td := NewTestDaemon()
	cfg := GetRecorder(ctx1)
	defer td.Close()

	fh := NewFiberInstrumentor(cfg)
	handler := fh.Handler(NewFixedSegmentNamer("test"), func(ctx *fiber.Ctx) error {
		return nil
	})
	rc := genericFiberRequestCtx()
	if err := handler(rc); err != nil {
		t.Error("Error calling handler:", err)
	}

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, fiber.StatusOK, rc.Response().StatusCode())
	assert.Equal(t, fiber.MethodPost, seg.HTTP.Request.Method)
	assert.Equal(t, "http://localhost/path", seg.HTTP.Request.URL)
	assert.Equal(t, "1.2.3.5", seg.HTTP.Request.ClientIP)
	assert.Equal(t, "UA_test", seg.HTTP.Request.UserAgent)
}

func genericFiberRequestCtx() *fiber.Ctx {
	return fiber.New().AcquireCtx(genericRequestCtx())
}
