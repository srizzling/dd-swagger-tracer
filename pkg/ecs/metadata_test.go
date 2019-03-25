package ecs

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGetPrivateHostIp(t *testing.T) {
	assert := assert.New(t)

	// func NewMockTransport() *MockTransport {
	// 	return &MockTransport{make(map[string]Responder), nil}
	// }
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Not running in EC2.. error out..
	ip, err := GetPrivateHostIP()
	assert.Equal("", ip)
	assert.Error(err)

	httpmock.RegisterResponder("GET", "http://169.254.169.254/latest/meta-data/local-ipv4", httpmock.NewStringResponder(200, `127.0.0.1`))

	ip, err = GetPrivateHostIP()
	assert.Equal("127.0.0.1", ip)
	assert.NoError(err)
}
