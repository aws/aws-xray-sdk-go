package ec2

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const metadata = `{
  "availabilityZone" : "us-west-2a",
  "imageId" : "ami-0fe02940a29f8239b",
  "instanceId" : "i-032fe2d42797fb9a1",
  "instanceType" : "c5.xlarge"
}`

const (
	documentPath = "/dynamic/instance-identity/document"
	tokenPath    = "/api/token"
	ec2Endpoint  = "http://169.254.169.254/latest"
)

func TestEndpoint(t *testing.T) {
	req, _ := http.NewRequest("GET", ec2Endpoint, nil)
	if e, a := ec2Endpoint, req.URL.String(); e != a {
		t.Errorf("expect %v, got %v", e, a)
	}
}

func TestIMDSv2Success(t *testing.T) {
	// Start a local HTTP server
	serverToken := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

		assert.Equal(t, req.URL.String(), tokenPath)
		_, _ = rw.Write([]byte("success"))
	}))

	serverMetadata := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

		assert.Equal(t, req.URL.String(), documentPath)
		assert.Equal(t, req.Header.Get("X-aws-ec2-metadata-token"), "success")
		_, _ = rw.Write([]byte(metadata))
	}))

	defer serverToken.Close()
	defer serverMetadata.Close()

	client := &http.Client{
		Transport: http.DefaultTransport,
	}

	// token fetch success
	respToken, _ := getToken(serverToken.URL+"/", client)
	assert.NotEqual(t, respToken, "")

	// successfully metadata fetch using IMDS v2
	respMetadata, _ := getMetadata(serverMetadata.URL+"/", client, respToken)
	ec2Metadata, _ := ioutil.ReadAll(respMetadata.Body)
	assert.Equal(t, []byte(metadata), ec2Metadata)
}

func TestIMDSv2Failv1Success(t *testing.T) {
	// Start a local HTTP server
	serverToken := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

		assert.Equal(t, req.URL.String(), tokenPath)
		_, _ = rw.Write([]byte("success"))
	}))

	serverMetadata := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

		assert.Equal(t, req.URL.String(), documentPath)
		_, _ = rw.Write([]byte(metadata))
	}))

	defer serverToken.Close()
	defer serverMetadata.Close()

	client := &http.Client{
		Transport: http.DefaultTransport,
	}

	// token fetch fail
	respToken, _ := getToken("/", client)
	assert.Equal(t, respToken, "")

	// fallback to IMDSv1 and successfully metadata fetch using IMDSv1
	respMetadata, _ := getMetadata(serverMetadata.URL+"/", client, respToken)
	ec2Metadata, _ := ioutil.ReadAll(respMetadata.Body)
	assert.Equal(t, []byte(metadata), ec2Metadata)
}

func TestIMDSv2Failv1Fail(t *testing.T) {
	// Start a local HTTP server
	serverToken := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, req.URL.String(), tokenPath)
		_, _ = rw.Write([]byte("success"))
	}))

	serverMetadata := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

		assert.Equal(t, req.URL.String(), documentPath)
		_, _ = rw.Write([]byte(metadata))
	}))

	defer serverToken.Close()
	defer serverMetadata.Close()

	client := &http.Client{
		Transport: http.DefaultTransport,
	}

	// token fetch fail
	respToken, _ := getToken("/", client)
	assert.Equal(t, respToken, "")

	// fallback to IMDSv1 and fail metadata fetch using IMDSv1
	_, err := getMetadata("/", client, respToken)
	assert.Error(t, err)
}
