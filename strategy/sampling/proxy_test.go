package sampling

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/aws/aws-xray-sdk-go/v2/daemoncfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestClient(t *testing.T, body []byte) *svcProxy {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, _ *http.Request) {
		_, err := res.Write(body)
		require.NoError(t, err)
	}))
	t.Cleanup(testServer.Close)

	u, err := url.Parse(testServer.URL)
	require.NoError(t, err)

	udpAddr, _ := net.ResolveUDPAddr("udp", u.Host)
	tcpAddr, _ := net.ResolveTCPAddr("tcp", u.Host)

	client, err := newProxy(&daemoncfg.DaemonEndpoints{
		UDPAddr: udpAddr,
		TCPAddr: tcpAddr,
	})

	require.NoError(t, err)

	return &client
}

func TestGetSamplingRules(t *testing.T) {
	body := []byte(`{
  "NextToken": null,
  "SamplingRuleRecords": [
    {
      "CreatedAt": 0,
      "ModifiedAt": 1649517389,
      "SamplingRule": {
        "Attributes": {},
        "FixedRate": 0.5,
        "HTTPMethod": "*",
        "Host": "*",
        "Priority": 10000,
        "ReservoirSize": 60,
        "ResourceARN": "*",
        "RuleARN": "arn:aws:xray:us-west-1:xxxxxxx:sampling-rule/Default",
        "RuleName": "Default",
        "ServiceName": "*",
        "ServiceType": "*",
        "URLPath": "*",
        "Version": 1
      }
    },
    {
      "CreatedAt": 1637691613,
      "ModifiedAt": 1643748669,
      "SamplingRule": {
        "Attributes": {},
        "FixedRate": 0.09,
        "HTTPMethod": "GET",
        "Host": "*",
        "Priority": 1,
        "ReservoirSize": 3,
        "ResourceARN": "*",
        "RuleARN": "arn:aws:xray:us-west-1:xxxxxxx:sampling-rule/test-rule",
        "RuleName": "test-rule",
        "ServiceName": "test-rule",
        "ServiceType": "local",
        "URLPath": "/aws-sdk-call",
        "Version": 1
      }
    },
    {
      "CreatedAt": 1639446197,
      "ModifiedAt": 1639446197,
      "SamplingRule": {
        "Attributes": {},
        "FixedRate": 0.09,
        "HTTPMethod": "*",
        "Host": "*",
        "Priority": 100,
        "ReservoirSize": 100,
        "ResourceARN": "*",
        "RuleARN": "arn:aws:xray:us-west-1:xxxxxxx:sampling-rule/test-rule-1",
        "RuleName": "test-rule-1",
        "ServiceName": "*",
        "ServiceType": "*",
        "URLPath": "*",
        "Version": 1
      }
    }
  ]
}`)
	client := createTestClient(t, body)

	samplingRules, err := (*client).GetSamplingRules()
	require.NoError(t, err)

	assert.Equal(t, *samplingRules[0].SamplingRule.RuleName, "Default")
	assert.Equal(t, *samplingRules[0].SamplingRule.ServiceType, "*")
	assert.Equal(t, *samplingRules[0].SamplingRule.Host, "*")
	assert.Equal(t, *samplingRules[0].SamplingRule.URLPath, "*")
	assert.Equal(t, *samplingRules[0].SamplingRule.ReservoirSize, int64(60))
	assert.Equal(t, *samplingRules[0].SamplingRule.FixedRate, 0.5)

	assert.Equal(t, *samplingRules[1].SamplingRule.RuleName, "test-rule")
	assert.Equal(t, *samplingRules[1].SamplingRule.ServiceType, "local")
	assert.Equal(t, *samplingRules[1].SamplingRule.Host, "*")
	assert.Equal(t, *samplingRules[1].SamplingRule.URLPath, "/aws-sdk-call")
	assert.Equal(t, *samplingRules[1].SamplingRule.ReservoirSize, int64(3))
	assert.Equal(t, *samplingRules[1].SamplingRule.FixedRate, 0.09)

	assert.Equal(t, *samplingRules[2].SamplingRule.RuleName, "test-rule-1")
	assert.Equal(t, *samplingRules[2].SamplingRule.ServiceType, "*")
	assert.Equal(t, *samplingRules[2].SamplingRule.Host, "*")
	assert.Equal(t, *samplingRules[2].SamplingRule.URLPath, "*")
	assert.Equal(t, *samplingRules[2].SamplingRule.ReservoirSize, int64(100))
	assert.Equal(t, *samplingRules[2].SamplingRule.FixedRate, 0.09)
}

func TestGetSamplingRulesWithMissingValues(t *testing.T) {
	body := []byte(`{
  "NextToken": null,
  "SamplingRuleRecords": [
    {
      "CreatedAt": 0,
      "ModifiedAt": 1639517389,
      "SamplingRule": {
        "Attributes": {},
        "FixedRate": 0.5,
        "HTTPMethod": "*",
        "Host": "*",
        "ResourceARN": "*",
        "RuleARN": "arn:aws:xray:us-west-1:xxxxxxx:sampling-rule/Default",
        "RuleName": "Default",
        "ServiceName": "*",
        "ServiceType": "*",
        "URLPath": "*",
        "Version": 1
      }
    }
  ]
}`)

	client := createTestClient(t, body)

	samplingRules, err := (*client).GetSamplingRules()
	require.NoError(t, err)

	// Priority and ReservoirSize are missing in API response so they are assigned as nil
	assert.Nil(t, samplingRules[0].SamplingRule.Priority)
	assert.Nil(t, samplingRules[0].SamplingRule.ReservoirSize)

	// other values are stored as expected
	assert.Equal(t, *samplingRules[0].SamplingRule.RuleName, "Default")
}

func TestGetSamplingTargets(t *testing.T) {
	body := []byte(`{
   "LastRuleModification": 123456,
   "SamplingTargetDocuments": [
      {
         "FixedRate": 6,
         "Interval": 6,
         "ReservoirQuota": 3,
         "ReservoirQuotaTTL": 456789,
         "RuleName": "r2"
      }
   ],
   "UnprocessedStatistics": [
      {
         "ErrorCode": "200",
         "Message": "ok",
         "RuleName": "r2"
      }
   ]
}`)

	client := createTestClient(t, body)

	samplingTragets, err := (*client).GetSamplingTargets(nil)
	require.NoError(t, err)

	assert.Equal(t, *samplingTragets.LastRuleModification, float64(123456))
	assert.Equal(t, *samplingTragets.SamplingTargetDocuments[0].FixedRate, float64(6))
	assert.Equal(t, *samplingTragets.SamplingTargetDocuments[0].Interval, int64(6))
	assert.Equal(t, *samplingTragets.SamplingTargetDocuments[0].ReservoirQuota, int64(3))
	assert.Equal(t, *samplingTragets.SamplingTargetDocuments[0].ReservoirQuotaTTL, float64(456789))
	assert.Equal(t, *samplingTragets.SamplingTargetDocuments[0].RuleName, "r2")
	assert.Equal(t, *samplingTragets.UnprocessedStatistics[0].RuleName, "r2")
	assert.Equal(t, *samplingTragets.UnprocessedStatistics[0].ErrorCode, "200")
	assert.Equal(t, *samplingTragets.UnprocessedStatistics[0].Message, "ok")
}

func TestGetSamplingTargetsMissingValues(t *testing.T) {
	body := []byte(`{
   "LastRuleModification": 123459,
   "SamplingTargetDocuments": [
      {
         "FixedRate": 5,
         "ReservoirQuotaTTL": 3456789,
         "RuleName": "r1"
      }
   ],
   "UnprocessedStatistics": [
      {
         "ErrorCode": "200",
         "Message": "ok",
         "RuleName": "r1"
      }
   ]
}`)

	client := createTestClient(t, body)

	samplingTragets, err := (*client).GetSamplingTargets(nil)
	require.NoError(t, err)

	assert.Nil(t, samplingTragets.SamplingTargetDocuments[0].Interval)
	assert.Nil(t, samplingTragets.SamplingTargetDocuments[0].ReservoirQuota)
}

func TestNilContext(t *testing.T) {
	client := createTestClient(t, []byte(``))
	samplingRulesOutput, err := (*client).GetSamplingRules()
	require.Error(t, err)
	require.Nil(t, samplingRulesOutput)

	samplingTargetsOutput, err := (*client).GetSamplingTargets(nil)
	require.Error(t, err)
	require.Nil(t, samplingTargetsOutput)
}

// Same as `newProxy` but returns type `proxy` instead of `svcProxy` for testing purposes
func createProxyForTesting(d *daemoncfg.DaemonEndpoints) (proxy, error) {
	if d == nil {
		d = daemoncfg.GetDaemonEndpoints()
	}
	url := "http://" + d.TCPAddr.String()

	// Construct resolved URLs for getSamplingRules and getSamplingTargets API calls.
	samplingRulesURL := url + "/GetSamplingRules"
	samplingTargetsURL := url + "/SamplingTargets"

	p := &proxy{
		xray: &xrayClient{
			httpClient:         &http.Client{},
			samplingRulesURL:   samplingRulesURL,
			samplingTargetsURL: samplingTargetsURL,
		},
	}

	return *p, nil
}

func TestNewClient(t *testing.T) {
	udpAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:2020")
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:2020")

	xrayClient, err := createProxyForTesting(&daemoncfg.DaemonEndpoints{
		UDPAddr: udpAddr,
		TCPAddr: tcpAddr,
	})

	require.NoError(t, err)

	assert.Equal(t, "http://127.0.0.1:2020/GetSamplingRules", xrayClient.xray.samplingRulesURL)
	assert.Equal(t, "http://127.0.0.1:2020/SamplingTargets", xrayClient.xray.samplingTargetsURL)
}

func TestEndpointIsNotReachable(t *testing.T) {
	udpAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:2020")
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:2020")

	client, err := createProxyForTesting(&daemoncfg.DaemonEndpoints{
		UDPAddr: udpAddr,
		TCPAddr: tcpAddr,
	})

	_, err = client.GetSamplingRules()
	assert.Error(t, err)
}
