package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientHelper(t *testing.T) {
	test := assert.New(t)
	paths, err := GetHostAuthenticationInfoPath()
	test.Nil(err)
	test.EqualValues(paths, &AuthenticationInfoPath{
		CaPath:   "/opt/cni-demo/ca.crt",
		CertPath: "/opt/cni-demo/cert.crt",
		KeyPath:  "/opt/cni-demo/key.key",
	})
}
