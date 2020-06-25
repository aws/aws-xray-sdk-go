package ec2

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testMetadata = `{
  "accountId" : "123367104812",
  "architecture" : "x86_64",
  "availabilityZone" : "us-west-2a",
  "billingProducts" : null,
  "devpayProductCodes" : null,
  "marketplaceProductCodes" : null,
  "imageId" : "ami-0fe02940a29f8239b",
  "instanceId" : "i-032fe2d42797fb9a1",
  "instanceType" : "c5.xlarge",
  "kernelId" : null,
  "pendingTime" : "2020-04-21T21:16:47Z",
  "privateIp" : "172.19.57.109",
  "ramdiskId" : null,
  "region" : "us-west-2",
  "version" : "2017-09-30"
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
		_, _ = rw.Write([]byte(testMetadata))
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
	assert.Equal(t, []byte(testMetadata), ec2Metadata)
}

func TestIMDSv2Failv1Success(t *testing.T) {
	// Start a local HTTP server
	serverToken := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

		assert.Equal(t, req.URL.String(), tokenPath)
		_, _ = rw.Write([]byte("success"))
	}))

	serverMetadata := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

		assert.Equal(t, req.URL.String(), documentPath)
		_, _ = rw.Write([]byte(testMetadata))
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
	assert.Equal(t, []byte(testMetadata), ec2Metadata)
}

func TestIMDSv2Failv1Fail(t *testing.T) {
	// Start a local HTTP server
	serverToken := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, req.URL.String(), tokenPath)
		_, _ = rw.Write([]byte("success"))
	}))

	serverMetadata := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

		assert.Equal(t, req.URL.String(), documentPath)
		_, _ = rw.Write([]byte(testMetadata))
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
