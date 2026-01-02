package client

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRequest(t *testing.T) {
	tests := []struct {
		name       string
		httpCfg    *HTTPConfig
		method     string
		url        string
		wantErr    bool
		wantHeader string
	}{
		{
			name: "valid GET request",
			httpCfg: &HTTPConfig{
				HeaderAuthorizationName: "Authorization",
				TokenJWT:                "my-jwt-token",
			},
			method:     "GET",
			url:        "http://localhost:8080/api/test",
			wantErr:    false,
			wantHeader: "Bearer my-jwt-token",
		},
		{
			name: "valid POST request",
			httpCfg: &HTTPConfig{
				HeaderAuthorizationName: "Authorization",
				TokenJWT:                "another-token",
			},
			method:     "POST",
			url:        "https://api.flecto.io/api/resource",
			wantErr:    false,
			wantHeader: "Bearer another-token",
		},
		{
			name: "custom header name",
			httpCfg: &HTTPConfig{
				HeaderAuthorizationName: "X-Custom-Auth",
				TokenJWT:                "custom-token",
			},
			method:     "GET",
			url:        "http://localhost/api",
			wantErr:    false,
			wantHeader: "Bearer custom-token",
		},
		{
			name: "empty token",
			httpCfg: &HTTPConfig{
				HeaderAuthorizationName: "Authorization",
				TokenJWT:                "",
			},
			method:     "GET",
			url:        "http://localhost/api",
			wantErr:    false,
			wantHeader: "Bearer ",
		},
		{
			name: "invalid url",
			httpCfg: &HTTPConfig{
				HeaderAuthorizationName: "Authorization",
				TokenJWT:                "token",
			},
			method:  "GET",
			url:     "://invalid-url",
			wantErr: true,
		},
		{
			name: "invalid method with spaces",
			httpCfg: &HTTPConfig{
				HeaderAuthorizationName: "Authorization",
				TokenJWT:                "token",
			},
			method:  "INVALID METHOD",
			url:     "http://localhost/api",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := NewRequest(tt.httpCfg, tt.method, tt.url, nil)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, req)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, req)
				assert.Equal(t, tt.method, req.Method)
				assert.Equal(t, tt.url, req.URL.String())
				assert.Equal(t, tt.wantHeader, req.Header.Get(tt.httpCfg.HeaderAuthorizationName))
			}
		})
	}
}

func TestNewRequest_WithBody(t *testing.T) {
	httpCfg := &HTTPConfig{
		HeaderAuthorizationName: "Authorization",
		TokenJWT:                "test-token",
	}
	body := strings.NewReader(`{"key": "value"}`)

	req, err := NewRequest(httpCfg, "POST", "http://localhost/api", body)

	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.NotNil(t, req.Body)
}