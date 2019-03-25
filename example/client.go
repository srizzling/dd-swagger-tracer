package main

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/sirupsen/logrus"
	"github.com/transurbantech/commons-golang-tracing/example/config"
)

func main() {

	logger := logrus.New()

	// load env vars
	conf, err := config.New()
	if err != nil {
		logger.WithError(err).Fatal("failed to load config")
	}

	// new SSM client
	sess, err := session.NewSession()
	if err != nil {
		logger.WithError(err).Fatal("failed to create aws session")
	}

	// inject ssm client into your conf to allow lazy-loading vars from SSM
	conf.WithSsm(ssm.New(sess))

	// gather creds from environment variables
	base, oauth2, err := conf.GetConnectionDetails("cls", []string{"m2m:external"})
	if err != nil {
		logger.WithError(err).Fatal("failed to get connection details")
	}

	// Get a http client with auth injected in
	client := oauth2.Client(context.Background())

	// use the client as normal (it will fetch the token as needed)
	url, _ := base.Parse("/pets")
	resp, err := client.Get(url.String())
	if err != nil {
		logger.WithError(err).Fatal("failed to request resource")
	}

	// response
	if resp.StatusCode >= 400 {
		logger.
			WithField("URL", url).
			WithField("Status", resp.Status).
			Error("Client with auth failure")
		return
	}

	logger.
		WithField("URL", url).
		WithField("Status", resp.Status).
		Info("Client with auth success")

}
