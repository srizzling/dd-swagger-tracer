package config

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/oauth2/clientcredentials"

	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"

	"github.com/caarlos0/env"
	"github.com/pkg/errors"
)

// Config is a global struct for obtaining values from env or SSM
type Config struct {
	ssm ssmiface.SSMAPI

	Product           string `env:"PRODUCT"`
	EnvironmentName   string `env:"ENVIRONMENT_NAME"`
	EnvironmentNumber string `env:"ENVIRONMENT_NUMBER"`

	SsmPrefix string `env:"SSM_PREFIX"`
}

//Load configuration from environment variable
func New() (*Config, error) {

	var conf Config
	err := env.Parse(&conf)
	if err != nil {
		return nil, err
	}

	if conf.SsmPrefix == "" {
		conf.SsmPrefix = fmt.Sprintf(`/%s/%s/%s`, conf.Product, conf.EnvironmentName, conf.EnvironmentNumber)
	}

	return &conf, nil
}

func (c *Config) WithSsm(client ssmiface.SSMAPI) {
	c.ssm = client
}

// GetClientCredentialsForAsset assumes these exists in SSM and this runtime has access
// /${SSM_PREFIX}/${TARGET}/base-url
// /${SSM_PREFIX}/${TARGET}/oauth2/client-id
// /${SSM_PREFIX}/${TARGET}/oauth2/client-secret
// /${SSM_PREFIX}/${TARGET}/oauth2/token-url
// /${SSM_PREFIX}/${TARGET}/oauth2/scope-prefix
func (c *Config) GetConnectionDetails(target string, scopes []string) (*url.URL, *clientcredentials.Config, error) {

	if c.ssm == nil {
		return nil, nil, errors.New("SSM client not set, use .WithSsm() first")
	}

	decrypt := true
	recursive := true
	path := fmt.Sprintf("%s/%s/", c.SsmPrefix, target)

	// get all parameters for target
	response, err := c.ssm.GetParametersByPath(&ssm.GetParametersByPathInput{
		Recursive:      &recursive,
		WithDecryption: &decrypt,
		Path:           &path,
	})

	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to fetch parameters with path %s", path)
	}

	// none found
	if response.Parameters == nil || len(response.Parameters) == 0 {
		return nil, nil, fmt.Errorf("no config found for SSM path %s", path)
	}

	// parse
	var baseUrl *url.URL
	config := clientcredentials.Config{Scopes: scopes}
	for _, parameter := range response.Parameters {

		switch {

		case strings.HasSuffix(*parameter.Name, "/oauth2/client-id"):
			config.ClientID = *parameter.Value

		case strings.HasSuffix(*parameter.Name, "/oauth2/client-secret"):
			config.ClientSecret = *parameter.Value

		case strings.HasSuffix(*parameter.Name, "/oauth2/token-url"):
			config.TokenURL = *parameter.Value

		case strings.HasSuffix(*parameter.Name, "/oauth2/scope-prefix"):
			for i, scope := range config.Scopes {
				config.Scopes[i] = fmt.Sprintf("%s%s", *parameter.Value, scope)
			}

		case strings.HasSuffix(*parameter.Name, "/base-url"):
			baseUrl, err = url.Parse(*parameter.Value)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to base-url value %s", *parameter.Value)
			}

		}
	}

	return baseUrl, &config, nil

}
