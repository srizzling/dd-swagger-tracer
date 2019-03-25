package ecs

import (
	"io/ioutil"
	"net/http"
	"time"
)

const (
	ec2MetadataIp4Url = "http://169.254.169.254/latest/meta-data/local-ipv4"

	// Since we're hitting the virtualhost, this should be very quick
	// Another reason to keep this quick is because it _will_ timeout when running locally
	timeoutSeconds = 1
)

// GetHostIP returns Private IPV4 address of the EC2 host
// its called from.
// It will return an error if not running in an EC2 Environment
// This can be run from within bridged network to discover an IP
func GetPrivateHostIP() (string, error) {

	// golang http client has no timeout, need to add one
	timeout := time.Duration(timeoutSeconds * time.Second)
	client := http.Client{
		Timeout: timeout,
	}

	// get metadata
	resp, err := client.Get(ec2MetadataIp4Url)
	if err != nil {
		return "", err
	}

	// read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// stringify
	text := string(body)

	return text, nil
}
