// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
package daemoncfg

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var portErr = "invalid daemon address port"
var addrErr = "invalid daemon address"

func TestGetDaemonEndpoints1(t *testing.T) { // default address set to udp and tcp
	udpAddr := "127.0.0.1:2000"
	tcpAddr := "127.0.0.1:2000"
	udpEndpt, _ := resolveUDPAddr(udpAddr)
	tcpEndpt, _ := resolveTCPAddr(tcpAddr)
	dEndpt := GetDaemonEndpoints()

	assert.Equal(t, dEndpt.UDPAddr, udpEndpt)
	assert.Equal(t, dEndpt.TCPAddr, tcpEndpt)
}

func TestGetDaemonEndpoints2(t *testing.T) { // default address set to udp and tcp
	udpAddr := "127.0.0.1:4000"
	tcpAddr := "127.0.0.1:5000"
	udpEndpt, _ := resolveUDPAddr(udpAddr)
	tcpEndpt, _ := resolveTCPAddr(tcpAddr)

	dAddr := "tcp:" + tcpAddr + " udp:" + udpAddr

	os.Setenv("AWS_XRAY_DAEMON_ADDRESS", dAddr) // env variable gets precedence over provided daemon addr
	defer os.Unsetenv("AWS_XRAY_DAEMON_ADDRESS")

	dEndpt := GetDaemonEndpoints()

	assert.Equal(t, dEndpt.UDPAddr, udpEndpt)
	assert.Equal(t, dEndpt.TCPAddr, tcpEndpt)
}

func TestGetDaemonEndpointsFromEnv1(t *testing.T) {
	udpAddr := "127.0.0.1:4000"
	tcpAddr := "127.0.0.1:5000"
	udpEndpt, _ := resolveUDPAddr(udpAddr)
	tcpEndpt, _ := resolveTCPAddr(tcpAddr)

	dAddr := "tcp:" + tcpAddr + " udp:" + udpAddr

	os.Setenv("AWS_XRAY_DAEMON_ADDRESS", dAddr)
	defer os.Unsetenv("AWS_XRAY_DAEMON_ADDRESS")

	dEndpt, _ := GetDaemonEndpointsFromEnv()

	assert.Equal(t, dEndpt.UDPAddr, udpEndpt)
	assert.Equal(t, dEndpt.TCPAddr, tcpEndpt)
}

func TestGetDaemonEndpointsFromEnv2(t *testing.T) {
	os.Setenv("AWS_XRAY_DAEMON_ADDRESS", "")
	defer os.Unsetenv("AWS_XRAY_DAEMON_ADDRESS")

	dEndpt, err := GetDaemonEndpointsFromEnv()

	assert.Nil(t, dEndpt)
	assert.Nil(t, err)
}

func TestGetDefaultDaemonEndpoints(t *testing.T) {
	udpAddr := "127.0.0.1:2000"
	tcpAddr := "127.0.0.1:2000"
	udpEndpt, _ := resolveUDPAddr(udpAddr)
	tcpEndpt, _ := resolveTCPAddr(tcpAddr)

	dEndpt := GetDefaultDaemonEndpoints()

	assert.Equal(t, dEndpt.UDPAddr, udpEndpt)
	assert.Equal(t, dEndpt.TCPAddr, tcpEndpt)
}

func TestGetDaemonEndpointsFromString1(t *testing.T) {
	udpAddr := "127.0.0.1:2000"
	tcpAddr := "127.0.0.1:2000"
	udpEndpt, _ := resolveUDPAddr(udpAddr)
	tcpEndpt, _ := resolveTCPAddr(tcpAddr)
	dAddr := udpAddr
	dEndpt, err := GetDaemonEndpointsFromString(dAddr)

	assert.Nil(t, err)
	assert.Equal(t, dEndpt.UDPAddr, udpEndpt)
	assert.Equal(t, dEndpt.TCPAddr, tcpEndpt)
}

func TestGetDaemonEndpointsFromString2(t *testing.T) {

	udpAddr := "127.0.0.1:2000"
	tcpAddr := "127.0.0.1:2000"

	dAddr := "127.0.0.1:2001"

	os.Setenv("AWS_XRAY_DAEMON_ADDRESS", udpAddr) // env variable gets precedence over provided daemon addr
	defer os.Unsetenv("AWS_XRAY_DAEMON_ADDRESS")

	udpEndpt, _ := resolveUDPAddr(udpAddr)
	tcpEndpt, _ := resolveTCPAddr(tcpAddr)

	dEndpt, err := GetDaemonEndpointsFromString(dAddr)

	assert.Nil(t, err)
	assert.Equal(t, dEndpt.UDPAddr, udpEndpt)
	assert.Equal(t, dEndpt.TCPAddr, tcpEndpt)
}

func TestGetDaemonEndpointsFromString3(t *testing.T) {
	udpAddr := "127.0.0.2:2001"
	tcpAddr := "127.0.0.1:2000"
	udpEndpt, _ := resolveUDPAddr(udpAddr)
	tcpEndpt, _ := resolveTCPAddr(tcpAddr)
	dAddr := "tcp:" + tcpAddr + " udp:" + udpAddr
	dEndpt, err := GetDaemonEndpointsFromString(dAddr)

	assert.Nil(t, err)
	assert.Equal(t, dEndpt.UDPAddr, udpEndpt)
	assert.Equal(t, dEndpt.TCPAddr, tcpEndpt)
}

func TestGetDaemonEndpointsFromString4(t *testing.T) {
	udpAddr := "127.0.0.2:2001"
	tcpAddr := "127.0.0.1:2000"
	udpEndpt, _ := resolveUDPAddr(udpAddr)
	tcpEndpt, _ := resolveTCPAddr(tcpAddr)
	dAddr := "udp:" + udpAddr + " tcp:" + tcpAddr
	dEndpt, err := GetDaemonEndpointsFromString(dAddr)

	assert.Nil(t, err)
	assert.Equal(t, dEndpt.UDPAddr, udpEndpt)
	assert.Equal(t, dEndpt.TCPAddr, tcpEndpt)
}

func TestGetDaemonEndpointsFromString5(t *testing.T) {
	udpAddr := "127.0.0.2:2001"
	tcpAddr := "127.0.0.1:2000"
	udpEndpt, _ := resolveUDPAddr(udpAddr)
	tcpEndpt, _ := resolveTCPAddr(tcpAddr)
	dAddr := "udp:" + udpAddr + " tcp:" + tcpAddr
	os.Setenv("AWS_XRAY_DAEMON_ADDRESS", dAddr) // env variable gets precedence over provided daemon addr
	defer os.Unsetenv("AWS_XRAY_DAEMON_ADDRESS")
	dEndpt, err := GetDaemonEndpointsFromString("tcp:127.0.0.5:2001 udp:127.0.0.5:2001")

	assert.Nil(t, err)
	assert.Equal(t, dEndpt.UDPAddr, udpEndpt)
	assert.Equal(t, dEndpt.TCPAddr, tcpEndpt)
}

func TestGetDaemonEndpointsFromStringInvalid1(t *testing.T) { // "udp:127.0.0.5:2001 udp:127.0.0.5:2001"
	udpAddr := "127.0.0.2:2001"
	tcpAddr := "127.0.0.1:2000"
	dAddr := "udp:" + udpAddr + " udp:" + tcpAddr
	dEndpt, err := GetDaemonEndpointsFromString(dAddr)

	assert.NotNil(t, err)
	assert.True(t, strings.Contains(fmt.Sprint(err), addrErr))
	assert.Nil(t, dEndpt)
}

func TestGetDaemonEndpointsFromStringInvalid2(t *testing.T) { // "tcp:127.0.0.5:2001 tcp:127.0.0.5:2001"
	udpAddr := "127.0.0.2:2001"
	tcpAddr := "127.0.0.1:2000"
	dAddr := "tcp:" + udpAddr + " tcp:" + tcpAddr
	dEndpt, err := GetDaemonEndpointsFromString(dAddr)

	assert.NotNil(t, err)
	assert.True(t, strings.Contains(fmt.Sprint(err), addrErr))
	assert.Nil(t, dEndpt)
}

func TestGetDaemonEndpointsFromStringInvalid3(t *testing.T) { // env variable set is invalid, string passed is valid
	udpAddr := "127.0.0.2:2001"
	tcpAddr := "127.0.0.1:2000"
	os.Setenv("AWS_XRAY_DAEMON_ADDRESS", "tcp:127.0.0.5:2001 tcp:127.0.0.5:2001") // invalid
	defer os.Unsetenv("AWS_XRAY_DAEMON_ADDRESS")
	dAddr := "udp:" + udpAddr + " tcp:" + tcpAddr
	dEndpt, err := GetDaemonEndpointsFromString(dAddr)

	assert.NotNil(t, err)
	assert.True(t, strings.Contains(fmt.Sprint(err), addrErr))
	assert.Nil(t, dEndpt)
}

func TestGetDaemonEndpointsFromStringInvalid4(t *testing.T) {
	udpAddr := "1.2.1:2a" // error in resolving address port
	tcpAddr := "127.0.0.1:2000"

	dAddr := "udp:" + udpAddr + " tcp:" + tcpAddr
	dEndpt, err := GetDaemonEndpointsFromString(dAddr)
	assert.True(t, strings.Contains(fmt.Sprint(err), portErr))
	assert.NotNil(t, err)
	assert.Nil(t, dEndpt)
}

func TestGetDaemonEndpointsFromStringInvalid5(t *testing.T) {
	udpAddr := "127.0.0.2:2001"
	tcpAddr := "127.0.a.1:2000" // error in resolving address

	dAddr := "udp:" + udpAddr + " tcp:" + tcpAddr
	dEndpt, err := GetDaemonEndpointsFromString(dAddr)

	assert.NotNil(t, err)
	assert.Nil(t, dEndpt)
}

func TestGetDaemonEndpointsFromStringInvalid6(t *testing.T) {
	udpAddr := "127.0.0.2:2001"
	dAddr := "udp:" + udpAddr // no tcp address present
	dEndpt, err := GetDaemonEndpointsFromString(dAddr)

	assert.NotNil(t, err)
	assert.True(t, strings.Contains(fmt.Sprint(err), addrErr))
	assert.Nil(t, dEndpt)
}

func TestGetDaemonEndpointsFromStringInvalid7(t *testing.T) {
	dAddr := ""
	dEndpt, err := GetDaemonEndpointsFromString(dAddr) // address passed is nil and env variable not set

	assert.Nil(t, err)
	assert.Nil(t, dEndpt)
}

func TestGetDaemonEndpointsForHostname1(t *testing.T) { // parsing hostname - single form
	udpAddr := "127.0.0.1:2000"
	tcpAddr := "127.0.0.1:2000"
	udpEndpt, _ := resolveUDPAddr(udpAddr)
	tcpEndpt, _ := resolveTCPAddr(tcpAddr)
	dEndpt, _ := GetDaemonEndpointsFromString("localhost:2000")

	assert.Equal(t, dEndpt.UDPAddr, udpEndpt)
	assert.Equal(t, dEndpt.TCPAddr, tcpEndpt)
}

func TestGetDaemonEndpointsForHostname2(t *testing.T) { // Invalid hostname - single form
	dEndpt, err := GetDaemonEndpointsFromString("XYZ:2000")
	assert.NotNil(t, err)
	assert.Nil(t, dEndpt)
}

func TestGetDaemonEndpointsForHostname3(t *testing.T) { // parsing hostname - double form
	udpAddr := "127.0.0.1:2000"
	tcpAddr := "127.0.0.1:2000"
	udpEndpt, _ := resolveUDPAddr(udpAddr)
	tcpEndpt, _ := resolveTCPAddr(tcpAddr)
	dEndpt, _ := GetDaemonEndpointsFromString("tcp:localhost:2000 udp:localhost:2000")

	assert.Equal(t, dEndpt.UDPAddr, udpEndpt)
	assert.Equal(t, dEndpt.TCPAddr, tcpEndpt)
}

func TestGetDaemonEndpointsForHostname4(t *testing.T) { // Invalid hostname - double form
	dEndpt, err := GetDaemonEndpointsFromString("tcp:ABC:2000 udp:XYZ:2000")
	assert.NotNil(t, err)
	assert.Nil(t, dEndpt)
}

func TestGetDaemonEndpointsForHostname5(t *testing.T) { // Invalid hostname - double form
	dEndpt, err := GetDaemonEndpointsFromString("tcp:localhost:2000 tcp:localhost:2000")
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(fmt.Sprint(err), addrErr))
	assert.Nil(t, dEndpt)
}

func TestGetDaemonEndpointsForHostname6(t *testing.T) { // Invalid port - single form
	dEndpt, err := GetDaemonEndpointsFromString("localhost:")
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(fmt.Sprint(err), portErr))
	assert.Nil(t, dEndpt)
}

func TestGetDaemonEndpointsForHostname7(t *testing.T) { // Invalid port - double form
	dEndpt, err := GetDaemonEndpointsFromString("tcp:localhost:r4 tcp:localhost:2000")
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(fmt.Sprint(err), portErr))
	assert.Nil(t, dEndpt)
}

// Benchmarks
func BenchmarkGetDaemonEndpoints(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetDaemonEndpoints()
	}
}

func BenchmarkGetDaemonEndpointsFromEnv_DoubleParse(b *testing.B) {
	os.Setenv("AWS_XRAY_DAEMON_ADDRESS", "tcp:127.0.0.1:2000 udp:127.0.0.1:2000")

	for i := 0; i < b.N; i++ {
		_, err := GetDaemonEndpointsFromEnv()
		if err != nil {
			return
		}
	}
	os.Unsetenv("AWS_XRAY_DAEMON_ADDRESS")
}

func BenchmarkGetDaemonEndpointsFromEnv_SingleParse(b *testing.B) {
	os.Setenv("AWS_XRAY_DAEMON_ADDRESS", "udp:127.0.0.1:2000")

	for i := 0; i < b.N; i++ {
		_, err := GetDaemonEndpointsFromEnv()
		if err != nil {
			return
		}
	}
	os.Unsetenv("AWS_XRAY_DAEMON_ADDRESS")
}

func BenchmarkGetDaemonEndpointsFromString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetDaemonEndpointsFromString("udp:127.0.0.1:2000")
		if err != nil {
			return
		}
	}
}
