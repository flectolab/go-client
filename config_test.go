package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.Http)
	assert.NotNil(t, cfg.Http.Client)
	assert.Equal(t, "Authorization", cfg.Http.HeaderAuthorizationName)
	assert.Equal(t, 5*time.Minute, cfg.IntervalCheck)
	assert.Equal(t, DefaultRedirectsLimit, cfg.RedirectsLimit)
	assert.Equal(t, DefaultPagesLimit, cfg.PagesLimit)
	assert.Empty(t, cfg.ManagerUrl)
	assert.Empty(t, cfg.NamespaceCode)
	assert.Empty(t, cfg.ProjectCode)
	assert.NotEmpty(t, cfg.AgentName)
}

func TestConfig_GetUrlApi(t *testing.T) {
	tests := []struct {
		name       string
		managerUrl string
		want       string
	}{
		{
			name:       "simple url",
			managerUrl: "http://localhost:8080",
			want:       "http://localhost:8080/api",
		},
		{
			name:       "https url",
			managerUrl: "https://api.flecto.io",
			want:       "https://api.flecto.io/api",
		},
		{
			name:       "empty url",
			managerUrl: "",
			want:       "/api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{ManagerUrl: tt.managerUrl}
			got := cfg.GetUrlApi()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfig_GetUrlApiProject(t *testing.T) {
	tests := []struct {
		name          string
		managerUrl    string
		namespaceCode string
		projectCode   string
		want          string
	}{
		{
			name:          "normal values",
			managerUrl:    "http://localhost:8080",
			namespaceCode: "my-namespace",
			projectCode:   "my-project",
			want:          "http://localhost:8080/api/namespace/my-namespace/project/my-project",
		},
		{
			name:          "empty values",
			managerUrl:    "",
			namespaceCode: "",
			projectCode:   "",
			want:          "/api/namespace//project/",
		},
		{
			name:          "production url",
			managerUrl:    "https://manager.flecto.io",
			namespaceCode: "prod",
			projectCode:   "website",
			want:          "https://manager.flecto.io/api/namespace/prod/project/website",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ManagerUrl:    tt.managerUrl,
				NamespaceCode: tt.namespaceCode,
				ProjectCode:   tt.projectCode,
			}
			got := cfg.GetUrlApiProject()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfig_GetUrlApiVersion(t *testing.T) {
	tests := []struct {
		name          string
		managerUrl    string
		namespaceCode string
		projectCode   string
		want          string
	}{
		{
			name:          "normal values",
			managerUrl:    "http://localhost:8080",
			namespaceCode: "ns1",
			projectCode:   "proj1",
			want:          "http://localhost:8080/api/namespace/ns1/project/proj1/version",
		},
		{
			name:          "empty values",
			managerUrl:    "",
			namespaceCode: "",
			projectCode:   "",
			want:          "/api/namespace//project//version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ManagerUrl:    tt.managerUrl,
				NamespaceCode: tt.namespaceCode,
				ProjectCode:   tt.projectCode,
			}
			got := cfg.GetUrlApiVersion()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfig_GetUrlApiRedirects(t *testing.T) {
	tests := []struct {
		name          string
		managerUrl    string
		namespaceCode string
		projectCode   string
		want          string
	}{
		{
			name:          "normal values",
			managerUrl:    "http://localhost:8080",
			namespaceCode: "ns1",
			projectCode:   "proj1",
			want:          "http://localhost:8080/api/namespace/ns1/project/proj1/redirects",
		},
		{
			name:          "empty values",
			managerUrl:    "",
			namespaceCode: "",
			projectCode:   "",
			want:          "/api/namespace//project//redirects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ManagerUrl:    tt.managerUrl,
				NamespaceCode: tt.namespaceCode,
				ProjectCode:   tt.projectCode,
			}
			got := cfg.GetUrlApiRedirects()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfig_GetUrlApiPages(t *testing.T) {
	tests := []struct {
		name          string
		managerUrl    string
		namespaceCode string
		projectCode   string
		want          string
	}{
		{
			name:          "normal values",
			managerUrl:    "http://localhost:8080",
			namespaceCode: "ns1",
			projectCode:   "proj1",
			want:          "http://localhost:8080/api/namespace/ns1/project/proj1/pages",
		},
		{
			name:          "empty values",
			managerUrl:    "",
			namespaceCode: "",
			projectCode:   "",
			want:          "/api/namespace//project//pages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ManagerUrl:    tt.managerUrl,
				NamespaceCode: tt.namespaceCode,
				ProjectCode:   tt.projectCode,
			}
			got := cfg.GetUrlApiPages()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfig_GetUrlApiAgents(t *testing.T) {
	tests := []struct {
		name          string
		managerUrl    string
		namespaceCode string
		projectCode   string
		want          string
	}{
		{
			name:          "normal values",
			managerUrl:    "http://localhost:8080",
			namespaceCode: "ns1",
			projectCode:   "proj1",
			want:          "http://localhost:8080/api/namespace/ns1/project/proj1/agents",
		},
		{
			name:          "empty values",
			managerUrl:    "",
			namespaceCode: "",
			projectCode:   "",
			want:          "/api/namespace//project//agents",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ManagerUrl:    tt.managerUrl,
				NamespaceCode: tt.namespaceCode,
				ProjectCode:   tt.projectCode,
			}
			got := cfg.GetUrlApiAgents()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfig_GetUrlApiAgentsHit(t *testing.T) {
	tests := []struct {
		name          string
		managerUrl    string
		namespaceCode string
		projectCode   string
		agentName     string
		want          string
	}{
		{
			name:          "normal values",
			managerUrl:    "http://localhost:8080",
			namespaceCode: "ns1",
			projectCode:   "proj1",
			agentName:     "my-agent",
			want:          "http://localhost:8080/api/namespace/ns1/project/proj1/agents/my-agent/hit",
		},
		{
			name:          "empty values",
			managerUrl:    "",
			namespaceCode: "",
			projectCode:   "",
			agentName:     "",
			want:          "/api/namespace//project//agents//hit",
		},
		{
			name:          "agent name with special chars",
			managerUrl:    "http://localhost:8080",
			namespaceCode: "ns1",
			projectCode:   "proj1",
			agentName:     "agent-node-01",
			want:          "http://localhost:8080/api/namespace/ns1/project/proj1/agents/agent-node-01/hit",
		},
		{
			name:          "production url",
			managerUrl:    "https://manager.flecto.io",
			namespaceCode: "prod",
			projectCode:   "website",
			agentName:     "web-server-1",
			want:          "https://manager.flecto.io/api/namespace/prod/project/website/agents/web-server-1/hit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ManagerUrl:    tt.managerUrl,
				NamespaceCode: tt.namespaceCode,
				ProjectCode:   tt.projectCode,
			}
			got := cfg.GetUrlApiAgentsHit(tt.agentName)
			assert.Equal(t, tt.want, got)
		})
	}
}
func TestConfig_GetRedirectsLimit(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want int
	}{
		{
			name: "unset falls back to the default",
			cfg:  &Config{},
			want: DefaultRedirectsLimit,
		},
		{
			name: "zero falls back to the default",
			cfg:  &Config{RedirectsLimit: 0},
			want: DefaultRedirectsLimit,
		},
		{
			name: "negative falls back to the default",
			cfg:  &Config{RedirectsLimit: -10},
			want: DefaultRedirectsLimit,
		},
		{
			name: "positive value is used as is",
			cfg:  &Config{RedirectsLimit: 42},
			want: 42,
		},
		{
			name: "the pages limit does not leak in",
			cfg:  &Config{PagesLimit: 7},
			want: DefaultRedirectsLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.GetRedirectsLimit()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfig_GetPagesLimit(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want int
	}{
		{
			name: "unset falls back to the default",
			cfg:  &Config{},
			want: DefaultPagesLimit,
		},
		{
			name: "zero falls back to the default",
			cfg:  &Config{PagesLimit: 0},
			want: DefaultPagesLimit,
		},
		{
			name: "negative falls back to the default",
			cfg:  &Config{PagesLimit: -10},
			want: DefaultPagesLimit,
		},
		{
			name: "positive value is used as is",
			cfg:  &Config{PagesLimit: 42},
			want: 42,
		},
		{
			name: "the redirects limit does not leak in",
			cfg:  &Config{RedirectsLimit: 7},
			want: DefaultPagesLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.GetPagesLimit()
			assert.Equal(t, tt.want, got)
		})
	}
}
