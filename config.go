package client

import (
	"fmt"
	"net/http"
	"time"
)

type HTTPConfig struct {
	Client                  HTTPClient
	HeaderAuthorizationName string
	TokenJWT                string
}

type Config struct {
	ManagerUrl    string
	NamespaceCode string
	ProjectCode   string

	Http *HTTPConfig

	IntervalCheck time.Duration
}

func NewDefaultConfig() *Config {
	return &Config{
		Http: &HTTPConfig{
			Client:                  http.DefaultClient,
			HeaderAuthorizationName: "Authorization",
		},
		IntervalCheck: 5 * time.Minute,
	}
}

func (c *Config) GetUrlApi() string {
	return fmt.Sprintf("%s/api", c.ManagerUrl)
}

func (c *Config) GetUrlApiProject() string {
	return fmt.Sprintf("%s/namespace/%s/project/%s", c.GetUrlApi(), c.NamespaceCode, c.ProjectCode)
}

func (c *Config) GetUrlApiVersion() string {
	return fmt.Sprintf("%s/version", c.GetUrlApiProject())
}

func (c *Config) GetUrlApiRedirects() string {
	return fmt.Sprintf("%s/redirects", c.GetUrlApiProject())
}

func (c *Config) GetUrlApiPages() string {
	return fmt.Sprintf("%s/pages", c.GetUrlApiProject())
}
