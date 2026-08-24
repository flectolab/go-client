package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/common/types"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
)

// mockHTTPClient is a manual mock for HTTPClient interface
type mockHTTPClient struct {
	calls     []*http.Request
	responses []mockResponse
	callIndex int
}

type mockResponse struct {
	resp *http.Response
	err  error
}

func newMockHTTPClient() *mockHTTPClient {
	return &mockHTTPClient{
		calls:     make([]*http.Request, 0),
		responses: make([]mockResponse, 0),
	}
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.calls = append(m.calls, req)
	if m.callIndex >= len(m.responses) {
		return nil, errors.New("no more mock responses configured")
	}
	resp := m.responses[m.callIndex]
	m.callIndex++
	return resp.resp, resp.err
}

func (m *mockHTTPClient) expect(resp *http.Response, err error) {
	m.responses = append(m.responses, mockResponse{resp: resp, err: err})
}

func newTestClient() (*client, *mockHTTPClient, clockwork.FakeClock) {
	mockHTTP := newMockHTTPClient()
	fakeClock := clockwork.NewFakeClock()

	cfg := &Config{
		ManagerUrl:    "http://localhost:8080",
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		AgentName:     "test-node",
		AgentType:     types.AgentTypeDefault,
		Http: &HTTPConfig{
			Client:                  mockHTTP,
			HeaderAuthorizationName: "Authorization",
			TokenJWT:                "test-token",
		},
		IntervalCheck:  5 * time.Minute,
		RedirectsLimit: 100,
		PagesLimit:     100,
	}

	c := &client{
		cfg:        cfg,
		httpClient: mockHTTP,
		clock:      fakeClock,
	}
	c.State.Store(&State{})

	return c, mockHTTP, fakeClock
}

func makeVersionResponse(version string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(version)),
	}
}

func makeRedirectsResponse(redirects []types.Redirect, total int) *http.Response {
	list := types.RedirectList{
		Items:  redirects,
		Total:  total,
		Limit:  100,
		Offset: 0,
	}
	body, _ := json.Marshal(list)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBuffer(body)),
	}
}

func makePagesResponse(pages []types.Page, total int) *http.Response {
	list := types.PageList{
		Items:  pages,
		Total:  total,
		Limit:  100,
		Offset: 0,
	}
	body, _ := json.Marshal(list)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBuffer(body)),
	}
}

func makeErrorResponse(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Body:       io.NopCloser(bytes.NewBufferString("error")),
	}
}

func makeAgentResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString("")),
	}
}

func TestNew(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.ManagerUrl = "http://localhost:8080"
	cfg.NamespaceCode = "ns"
	cfg.ProjectCode = "proj"

	c := New(cfg)

	assert.NotNil(t, c)
	assert.Implements(t, (*Client)(nil), c)
}

func TestClient_Init_Success(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	redirects := []types.Redirect{
		{Type: types.RedirectTypeBasic, Source: "/old", Target: "/new", Status: types.RedirectStatusMovedPermanent},
	}
	pages := []types.Page{
		{Type: types.PageTypeBasic, Path: "/robots.txt", Content: "User-agent: *", ContentType: types.PageContentTypeTextPlain},
	}

	mockHTTP.expect(makeVersionResponse("1"), nil)
	mockHTTP.expect(makeVersionResponse("1"), nil)
	mockHTTP.expect(makeRedirectsResponse(redirects, 1), nil)
	mockHTTP.expect(makePagesResponse(pages, 1), nil)
	mockHTTP.expect(makeAgentResponse(), nil)

	err := c.Init()

	assert.NoError(t, err)
	assert.NotNil(t, c.State.Load())
	assert.Equal(t, 1, c.State.Load().(*State).ProjectVersion)
}

func TestClient_Init_VersionError(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	mockHTTP.expect(nil, errors.New("connection refused"))

	err := c.Init()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestClient_Init_InvalidAgentType(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	fakeClock := clockwork.NewFakeClock()

	cfg := &Config{
		ManagerUrl:    "http://localhost:8080",
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		AgentName:     "test-node",
		AgentType:     "invalid-type",
		Http: &HTTPConfig{
			Client:                  mockHTTP,
			HeaderAuthorizationName: "Authorization",
			TokenJWT:                "test-token",
		},
		IntervalCheck:  5 * time.Minute,
		RedirectsLimit: 100,
		PagesLimit:     100,
	}

	c := &client{
		cfg:        cfg,
		httpClient: mockHTTP,
		clock:      fakeClock,
	}
	c.State.Store(&State{})

	err := c.Init()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid agent type")
}

func TestClient_Init_RedirectsError(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	mockHTTP.expect(makeVersionResponse("1"), nil)
	mockHTTP.expect(makeVersionResponse("1"), nil)
	mockHTTP.expect(nil, errors.New("failed to fetch redirects"))
	mockHTTP.expect(makeAgentResponse(), nil)

	err := c.Init()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch redirects")
}

func Test_client_load(t *testing.T) {
	c, _, _ := newTestClient()

	state := c.load()
	assert.IsType(t, &State{}, state)
}

func Test_client_GetStateVersion(t *testing.T) {
	c, _, _ := newTestClient()
	c.State.Store(&State{ProjectVersion: 2})
	assert.Equal(t, c.GetStateVersion(), 2)
}

func Test_client_RedirectMatch(t *testing.T) {
	c, _, _ := newTestClient()
	redirects := []*types.Redirect{
		{Type: types.RedirectTypeBasic, Source: "/source", Target: "/target", Status: types.RedirectStatusMovedPermanent},
		{Type: types.RedirectTypeBasic, Source: "/foo", Target: "/bar", Status: types.RedirectStatusMovedPermanent},
	}
	tree := types.NewRedirectTreeMatcher()
	for _, redirect := range redirects {
		_ = tree.Insert(redirect)
	}
	c.State.Store(&State{RedirectMatcher: tree})
	redirect, target := c.RedirectMatch("example.com", "/source")
	assert.Equal(t, redirect, redirects[0])
	assert.Equal(t, target, redirects[0].Target)
}

func Test_client_PageMatch(t *testing.T) {
	c, _, _ := newTestClient()
	pages := []*types.Page{
		{Type: types.PageTypeBasic, Path: "/robots.txt", Content: "User-agent: *", ContentType: types.PageContentTypeTextPlain},
		{Type: types.PageTypeBasic, Path: "/sitemap.xml", Content: "<xml>", ContentType: types.PageContentTypeXML},
	}
	tree := types.NewPageTreeMatcher()
	for _, page := range pages {
		tree.Insert(page)
	}
	c.State.Store(&State{PageMatcher: tree})
	page := c.PageMatch("example.com", "/robots.txt")
	assert.Equal(t, page, pages[0])
}

func TestClient_getProjectVersion_Success(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		wantVersion int
	}{
		{
			name:        "simple version",
			response:    "42",
			wantVersion: 42,
		},
		{
			name:        "version with whitespace",
			response:    "  123  \n",
			wantVersion: 123,
		},
		{
			name:        "zero version",
			response:    "0",
			wantVersion: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, mockHTTP, _ := newTestClient()

			mockHTTP.expect(makeVersionResponse(tt.response), nil)

			version, err := c.getProjectVersion()

			assert.NoError(t, err)
			assert.Equal(t, tt.wantVersion, version)
		})
	}
}

func TestClient_getProjectVersion_HTTPError(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	mockHTTP.expect(nil, errors.New("network error"))

	version, err := c.getProjectVersion()

	assert.Error(t, err)
	assert.Equal(t, 0, version)
}

func TestClient_getProjectVersion_Non200Status(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "not found", statusCode: http.StatusNotFound},
		{name: "internal error", statusCode: http.StatusInternalServerError},
		{name: "unauthorized", statusCode: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, mockHTTP, _ := newTestClient()

			mockHTTP.expect(makeErrorResponse(tt.statusCode), nil)

			version, err := c.getProjectVersion()

			assert.Error(t, err)
			assert.Equal(t, 0, version)
			assert.Contains(t, err.Error(), "unexpected status code")
		})
	}
}

func TestClient_getProjectVersion_InvalidResponse(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	mockHTTP.expect(makeVersionResponse("not-a-number"), nil)

	version, err := c.getProjectVersion()

	assert.Error(t, err)
	assert.Equal(t, 0, version)
}

func TestClient_getProjectVersion_ReadBodyError(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(&errorReader{}),
	}

	mockHTTP.expect(resp, nil)

	version, err := c.getProjectVersion()

	assert.Error(t, err)
	assert.Equal(t, 0, version)
}

func TestClient_getProjectVersion_NewRequestError(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	fakeClock := clockwork.NewFakeClock()

	cfg := &Config{
		ManagerUrl:    "://invalid-url",
		NamespaceCode: "ns",
		ProjectCode:   "proj",
		Http: &HTTPConfig{
			Client:                  mockHTTP,
			HeaderAuthorizationName: "Authorization",
			TokenJWT:                "token",
		},
		IntervalCheck: 5 * time.Minute,
	}

	c := &client{
		cfg:        cfg,
		httpClient: mockHTTP,
		clock:      fakeClock,
	}

	version, err := c.getProjectVersion()

	assert.Error(t, err)
	assert.Equal(t, 0, version)
}

type errorReader struct{}

func (e *errorReader) Read(_ []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func TestClient_getProjectRedirects_Success(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	redirects := []types.Redirect{
		{Type: types.RedirectTypeBasic, Source: "/old1", Target: "/new1", Status: types.RedirectStatusMovedPermanent},
		{Type: types.RedirectTypeBasic, Source: "/old2", Target: "/new2", Status: types.RedirectStatusFound},
	}

	mockHTTP.expect(makeRedirectsResponse(redirects, 2), nil)

	result, err := c.getProjectRedirects()

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "/old1", result[0].Source)
	assert.Equal(t, "/old2", result[1].Source)
}

func TestClient_getProjectRedirects_Pagination(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	page1 := make([]types.Redirect, 100)
	for i := 0; i < 100; i++ {
		page1[i] = types.Redirect{Type: types.RedirectTypeBasic, Source: "/page1", Target: "/target1"}
	}

	page2 := []types.Redirect{
		{Type: types.RedirectTypeBasic, Source: "/page2", Target: "/target2"},
	}

	resp1 := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(bytes.NewBuffer(func() []byte {
			list := types.RedirectList{Items: page1, Total: 101, Limit: 100, Offset: 0}
			b, _ := json.Marshal(list)
			return b
		}())),
	}

	resp2 := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(bytes.NewBuffer(func() []byte {
			list := types.RedirectList{Items: page2, Total: 101, Limit: 100, Offset: 100}
			b, _ := json.Marshal(list)
			return b
		}())),
	}

	mockHTTP.expect(resp1, nil)
	mockHTTP.expect(resp2, nil)

	result, err := c.getProjectRedirects()

	assert.NoError(t, err)
	assert.Len(t, result, 101)
}

func TestClient_getProjectRedirects_HTTPError(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	mockHTTP.expect(nil, errors.New("connection failed"))

	result, err := c.getProjectRedirects()

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestClient_getProjectRedirects_Non200Status(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	mockHTTP.expect(makeErrorResponse(http.StatusForbidden), nil)

	result, err := c.getProjectRedirects()

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unexpected status code")
}

func TestClient_getProjectRedirects_InvalidJSON(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString("invalid json")),
	}

	mockHTTP.expect(resp, nil)

	result, err := c.getProjectRedirects()

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestClient_getProjectRedirects_NewRequestError(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	fakeClock := clockwork.NewFakeClock()

	cfg := &Config{
		ManagerUrl:    "://invalid-url",
		NamespaceCode: "ns",
		ProjectCode:   "proj",
		Http: &HTTPConfig{
			Client:                  mockHTTP,
			HeaderAuthorizationName: "Authorization",
			TokenJWT:                "token",
		},
		IntervalCheck: 5 * time.Minute,
	}

	c := &client{
		cfg:        cfg,
		httpClient: mockHTTP,
		clock:      fakeClock,
	}

	result, err := c.getProjectRedirects()

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestClient_getProjectPages_Success(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	pages := []types.Page{
		{Type: types.PageTypeBasic, Path: "/robots.txt", Content: "User-agent: *", ContentType: types.PageContentTypeTextPlain},
		{Type: types.PageTypeBasic, Path: "/sitemap.xml", Content: "<xml>", ContentType: types.PageContentTypeXML},
	}

	mockHTTP.expect(makePagesResponse(pages, 2), nil)

	result, err := c.getProjectPages()

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "/robots.txt", result[0].Path)
	assert.Equal(t, "/sitemap.xml", result[1].Path)
}

func TestClient_getProjectPages_Pagination(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	page1 := make([]types.Page, 100)
	for i := 0; i < 100; i++ {
		page1[i] = types.Page{Type: types.PageTypeBasic, Path: "/page1", Content: "content"}
	}

	page2 := []types.Page{
		{Type: types.PageTypeBasic, Path: "/page2", Content: "content2"},
	}

	resp1 := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(bytes.NewBuffer(func() []byte {
			list := types.PageList{Items: page1, Total: 101, Limit: 100, Offset: 0}
			b, _ := json.Marshal(list)
			return b
		}())),
	}

	resp2 := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(bytes.NewBuffer(func() []byte {
			list := types.PageList{Items: page2, Total: 101, Limit: 100, Offset: 100}
			b, _ := json.Marshal(list)
			return b
		}())),
	}

	mockHTTP.expect(resp1, nil)
	mockHTTP.expect(resp2, nil)

	result, err := c.getProjectPages()

	assert.NoError(t, err)
	assert.Len(t, result, 101)
}

func TestClient_getProjectPages_HTTPError(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	mockHTTP.expect(nil, errors.New("connection failed"))

	result, err := c.getProjectPages()

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestClient_getProjectPages_Non200Status(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	mockHTTP.expect(makeErrorResponse(http.StatusForbidden), nil)

	result, err := c.getProjectPages()

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unexpected status code")
}

func TestClient_getProjectPages_InvalidJSON(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString("invalid json")),
	}

	mockHTTP.expect(resp, nil)

	result, err := c.getProjectPages()

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestClient_getProjectPages_NewRequestError(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	fakeClock := clockwork.NewFakeClock()

	cfg := &Config{
		ManagerUrl:    "://invalid-url",
		NamespaceCode: "ns",
		ProjectCode:   "proj",
		Http: &HTTPConfig{
			Client:                  mockHTTP,
			HeaderAuthorizationName: "Authorization",
			TokenJWT:                "token",
		},
		IntervalCheck: 5 * time.Minute,
	}

	c := &client{
		cfg:        cfg,
		httpClient: mockHTTP,
		clock:      fakeClock,
	}

	result, err := c.getProjectPages()

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestClient_loadState_Success(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	redirects := []types.Redirect{
		{Type: types.RedirectTypeBasic, Source: "/test", Target: "/target", Status: types.RedirectStatusMovedPermanent},
	}
	pages := []types.Page{
		{Type: types.PageTypeBasic, Path: "/robots.txt", Content: "User-agent: *", ContentType: types.PageContentTypeTextPlain},
	}

	mockHTTP.expect(makeVersionResponse("5"), nil)
	mockHTTP.expect(makeRedirectsResponse(redirects, 1), nil)
	mockHTTP.expect(makePagesResponse(pages, 1), nil)

	err := c.loadState()

	assert.NoError(t, err)
	assert.NotNil(t, c.State.Load())
	assert.Equal(t, 5, c.State.Load().(*State).ProjectVersion)
	assert.NotNil(t, c.State.Load().(*State).RedirectMatcher)
	assert.NotNil(t, c.State.Load().(*State).PageMatcher)
}

func TestClient_loadState_VersionError(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	mockHTTP.expect(nil, errors.New("version error"))

	err := c.loadState()

	assert.Error(t, err)
}

func TestClient_loadState_RedirectsError(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	mockHTTP.expect(makeVersionResponse("1"), nil)
	mockHTTP.expect(nil, errors.New("redirects error"))

	err := c.loadState()

	assert.Error(t, err)
}

func TestClient_loadState_InvalidRedirectRegex(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	redirects := []types.Redirect{
		{Type: types.RedirectTypeRegex, Source: "[invalid(regex", Target: "/target", Status: types.RedirectStatusMovedPermanent},
	}

	mockHTTP.expect(makeVersionResponse("1"), nil)
	mockHTTP.expect(makeRedirectsResponse(redirects, 1), nil)

	err := c.loadState()

	assert.Error(t, err)
}

func TestClient_loadState_PagesError(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	redirects := []types.Redirect{
		{Type: types.RedirectTypeBasic, Source: "/test", Target: "/target", Status: types.RedirectStatusMovedPermanent},
	}

	mockHTTP.expect(makeVersionResponse("1"), nil)
	mockHTTP.expect(makeRedirectsResponse(redirects, 1), nil)
	mockHTTP.expect(nil, errors.New("pages error"))

	err := c.loadState()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pages error")
}

func TestClient_Reload_NoVersionChange(t *testing.T) {
	c, mockHTTP, _ := newTestClient()
	c.State.Store(&State{ProjectVersion: 1, RedirectMatcher: types.NewRedirectTreeMatcher()})

	mockHTTP.expect(makeVersionResponse("1"), nil)
	mockHTTP.expect(makeAgentResponse(), nil)

	err := c.Reload()

	assert.NoError(t, err)
	assert.Equal(t, 1, c.State.Load().(*State).ProjectVersion)
	assert.Len(t, mockHTTP.calls, 2)
}

func TestClient_Reload_VersionChange(t *testing.T) {
	c, mockHTTP, _ := newTestClient()
	c.State.Store(&State{ProjectVersion: 1, RedirectMatcher: types.NewRedirectTreeMatcher()})

	redirects := []types.Redirect{
		{Type: types.RedirectTypeBasic, Source: "/new", Target: "/target", Status: types.RedirectStatusMovedPermanent},
	}
	pages := []types.Page{
		{Type: types.PageTypeBasic, Path: "/robots.txt", Content: "User-agent: *", ContentType: types.PageContentTypeTextPlain},
	}

	mockHTTP.expect(makeVersionResponse("2"), nil)
	mockHTTP.expect(makeVersionResponse("2"), nil)
	mockHTTP.expect(makeRedirectsResponse(redirects, 1), nil)
	mockHTTP.expect(makePagesResponse(pages, 1), nil)
	mockHTTP.expect(makeAgentResponse(), nil)

	err := c.Reload()

	assert.NoError(t, err)
	assert.Equal(t, 2, c.State.Load().(*State).ProjectVersion)
}

func TestClient_Reload_VersionError(t *testing.T) {
	c, mockHTTP, _ := newTestClient()
	c.State.Store(&State{ProjectVersion: 1, RedirectMatcher: types.NewRedirectTreeMatcher()})

	mockHTTP.expect(nil, errors.New("network error"))

	err := c.Reload()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network error")
	assert.Equal(t, 1, c.State.Load().(*State).ProjectVersion)
}

func TestClient_Reload_LoadStateError(t *testing.T) {
	c, mockHTTP, _ := newTestClient()
	c.State.Store(&State{ProjectVersion: 1, RedirectMatcher: types.NewRedirectTreeMatcher()})

	mockHTTP.expect(makeVersionResponse("2"), nil)
	mockHTTP.expect(nil, errors.New("failed to load version"))
	mockHTTP.expect(makeAgentResponse(), nil)

	err := c.Reload()

	assert.Error(t, err)
	assert.Equal(t, 1, c.State.Load().(*State).ProjectVersion)
}

func TestClient_Reload_TryLockFails(t *testing.T) {
	c, _, _ := newTestClient()
	c.State.Store(&State{ProjectVersion: 1, RedirectMatcher: types.NewRedirectTreeMatcher()})

	c.reloadMu.Lock()
	defer c.reloadMu.Unlock()

	err := c.Reload()

	assert.NoError(t, err)
	assert.Equal(t, 1, c.State.Load().(*State).ProjectVersion)
}

func TestClient_Start_ContextCancellation(t *testing.T) {
	c, _, fakeClock := newTestClient()

	c.State.Store(&State{ProjectVersion: 1, RedirectMatcher: types.NewRedirectTreeMatcher()})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		c.Start(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Start did not exit after context cancellation")
	}

	_ = fakeClock
}

func TestClient_Start_VersionCheckNoChange(t *testing.T) {
	c, mockHTTP, fakeClock := newTestClient()

	c.State.Store(&State{ProjectVersion: 1, RedirectMatcher: types.NewRedirectTreeMatcher()})

	ctx, cancel := context.WithCancel(context.Background())

	// Add multiple responses for version check (may be called multiple times)
	mockHTTP.expect(makeVersionResponse("1"), nil)
	mockHTTP.expect(makeAgentResponse(), nil)
	mockHTTP.expect(makeVersionResponse("1"), nil)
	mockHTTP.expect(makeAgentResponse(), nil)
	mockHTTP.expect(makeVersionResponse("1"), nil)
	mockHTTP.expect(makeAgentResponse(), nil)

	done := make(chan struct{})
	go func() {
		c.Start(ctx)
		close(done)
	}()

	fakeClock.Advance(5 * time.Minute)
	time.Sleep(50 * time.Millisecond)

	cancel()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Start did not exit")
	}

	assert.Equal(t, 1, c.State.Load().(*State).ProjectVersion)
}

func TestClient_Start_VersionChangeTriggersReload(t *testing.T) {
	c, mockHTTP, fakeClock := newTestClient()

	c.State.Store(&State{ProjectVersion: 1, RedirectMatcher: types.NewRedirectTreeMatcher()})

	ctx, cancel := context.WithCancel(context.Background())

	redirects := []types.Redirect{
		{Type: types.RedirectTypeBasic, Source: "/new", Target: "/target", Status: types.RedirectStatusMovedPermanent},
	}
	pages := []types.Page{
		{Type: types.PageTypeBasic, Path: "/robots.txt", Content: "User-agent: *", ContentType: types.PageContentTypeTextPlain},
	}

	mockHTTP.expect(makeVersionResponse("2"), nil)
	mockHTTP.expect(makeVersionResponse("2"), nil)
	mockHTTP.expect(makeRedirectsResponse(redirects, 1), nil)
	mockHTTP.expect(makePagesResponse(pages, 1), nil)
	mockHTTP.expect(makeAgentResponse(), nil)

	done := make(chan struct{})
	go func() {
		c.Start(ctx)
		close(done)
	}()

	fakeClock.BlockUntil(1)
	fakeClock.Advance(5 * time.Minute)
	time.Sleep(100 * time.Millisecond)

	cancel()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Start did not exit")
	}

	assert.Equal(t, 2, c.State.Load().(*State).ProjectVersion)
}

func TestClient_Start_LoadStateErrorDuringReload(t *testing.T) {
	c, mockHTTP, fakeClock := newTestClient()

	c.State.Store(&State{ProjectVersion: 1, RedirectMatcher: types.NewRedirectTreeMatcher()})

	ctx, cancel := context.WithCancel(context.Background())

	mockHTTP.expect(makeVersionResponse("2"), nil)
	mockHTTP.expect(makeVersionResponse("2"), nil)
	mockHTTP.expect(nil, errors.New("failed to load redirects"))
	mockHTTP.expect(makeAgentResponse(), nil)

	done := make(chan struct{})
	go func() {
		c.Start(ctx)
		close(done)
	}()

	fakeClock.BlockUntil(1)
	fakeClock.Advance(5 * time.Minute)
	time.Sleep(100 * time.Millisecond)

	cancel()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Start did not exit")
	}

	assert.Equal(t, 1, c.State.Load().(*State).ProjectVersion)
}

func TestClient_Start_VersionCheckError(t *testing.T) {
	c, mockHTTP, fakeClock := newTestClient()

	c.State.Store(&State{ProjectVersion: 1, RedirectMatcher: types.NewRedirectTreeMatcher()})

	ctx, cancel := context.WithCancel(context.Background())

	mockHTTP.expect(nil, errors.New("network error"))

	done := make(chan struct{})
	go func() {
		c.Start(ctx)
		close(done)
	}()

	fakeClock.BlockUntil(1)
	fakeClock.Advance(5 * time.Minute)
	time.Sleep(100 * time.Millisecond)

	cancel()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Start did not exit")
	}

	assert.Equal(t, 1, c.State.Load().(*State).ProjectVersion)
}

func TestClient_Start_TryLockFails(t *testing.T) {
	c, _, fakeClock := newTestClient()

	c.State.Store(&State{ProjectVersion: 1, RedirectMatcher: types.NewRedirectTreeMatcher()})

	ctx, cancel := context.WithCancel(context.Background())

	// Lock the mutex before starting - TryLock will fail
	c.reloadMu.Lock()

	done := make(chan struct{})
	go func() {
		c.Start(ctx)
		close(done)
	}()

	fakeClock.BlockUntil(1)
	fakeClock.Advance(5 * time.Minute)
	time.Sleep(50 * time.Millisecond)

	// Unlock and cancel
	c.reloadMu.Unlock()
	cancel()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Start did not exit")
	}

	// Version should remain unchanged since TryLock failed
	assert.Equal(t, 1, c.State.Load().(*State).ProjectVersion)
}

func TestClient_sendAgentStatus_Success(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	agent := types.Agent{
		Name:    "test-node",
		Type:    types.AgentTypeDefault,
		Version: 1,
		Status:  types.AgentStatusSuccess,
	}

	mockHTTP.expect(makeAgentResponse(), nil)

	err := c.sendAgentStatus(agent)

	assert.NoError(t, err)
	assert.Len(t, mockHTTP.calls, 1)
	assert.Equal(t, http.MethodPost, mockHTTP.calls[0].Method)
	assert.Contains(t, mockHTTP.calls[0].URL.String(), "/agents")
}

func TestClient_sendAgentStatus_HTTPError(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	agent := types.Agent{
		Name:    "test-node",
		Type:    types.AgentTypeDefault,
		Version: 1,
		Status:  types.AgentStatusSuccess,
	}

	mockHTTP.expect(nil, errors.New("network error"))

	err := c.sendAgentStatus(agent)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network error")
}

func TestClient_sendAgentStatus_Non200Status(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "bad request", statusCode: http.StatusBadRequest},
		{name: "internal error", statusCode: http.StatusInternalServerError},
		{name: "unauthorized", statusCode: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, mockHTTP, _ := newTestClient()

			agent := types.Agent{
				Name:    "test-node",
				Type:    types.AgentTypeDefault,
				Version: 1,
				Status:  types.AgentStatusSuccess,
			}

			mockHTTP.expect(makeErrorResponse(tt.statusCode), nil)

			err := c.sendAgentStatus(agent)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unexpected status code")
		})
	}
}

func TestClient_sendAgentStatus_ValidationError(t *testing.T) {
	c, _, _ := newTestClient()

	agent := types.Agent{
		Name:    "",
		Type:    types.AgentTypeDefault,
		Version: 1,
		Status:  types.AgentStatusSuccess,
	}

	err := c.sendAgentStatus(agent)

	assert.Error(t, err)
}

func TestClient_sendAgentStatus_NewRequestError(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	fakeClock := clockwork.NewFakeClock()

	cfg := &Config{
		ManagerUrl:    "://invalid-url",
		NamespaceCode: "ns",
		ProjectCode:   "proj",
		AgentName:     "test-node",
		AgentType:     types.AgentTypeDefault,
		Http: &HTTPConfig{
			Client:                  mockHTTP,
			HeaderAuthorizationName: "Authorization",
			TokenJWT:                "token",
		},
		IntervalCheck: 5 * time.Minute,
	}

	c := &client{
		cfg:        cfg,
		httpClient: mockHTTP,
		clock:      fakeClock,
	}

	agent := types.Agent{
		Name:    "test-node",
		Type:    types.AgentTypeDefault,
		Version: 1,
		Status:  types.AgentStatusSuccess,
	}

	err := c.sendAgentStatus(agent)

	assert.Error(t, err)
}

func TestClient_sendAgentHit_Success(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	mockHTTP.expect(makeAgentResponse(), nil)

	err := c.sendAgentHit("test-node")

	assert.NoError(t, err)
	assert.Len(t, mockHTTP.calls, 1)
	assert.Equal(t, http.MethodPatch, mockHTTP.calls[0].Method)
	assert.Contains(t, mockHTTP.calls[0].URL.String(), "/agents/test-node/hit")
}

func TestClient_sendAgentHit_HTTPError(t *testing.T) {
	c, mockHTTP, _ := newTestClient()

	mockHTTP.expect(nil, errors.New("network error"))

	err := c.sendAgentHit("test-node")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network error")
}

func TestClient_sendAgentHit_Non200Status(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "bad request", statusCode: http.StatusBadRequest},
		{name: "internal error", statusCode: http.StatusInternalServerError},
		{name: "unauthorized", statusCode: http.StatusUnauthorized},
		{name: "not found", statusCode: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, mockHTTP, _ := newTestClient()

			mockHTTP.expect(makeErrorResponse(tt.statusCode), nil)

			err := c.sendAgentHit("test-node")

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unexpected status code")
		})
	}
}

func TestClient_sendAgentHit_NewRequestError(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	fakeClock := clockwork.NewFakeClock()

	cfg := &Config{
		ManagerUrl:    "://invalid-url",
		NamespaceCode: "ns",
		ProjectCode:   "proj",
		AgentName:     "test-node",
		AgentType:     types.AgentTypeDefault,
		Http: &HTTPConfig{
			Client:                  mockHTTP,
			HeaderAuthorizationName: "Authorization",
			TokenJWT:                "token",
		},
		IntervalCheck: 5 * time.Minute,
	}

	c := &client{
		cfg:        cfg,
		httpClient: mockHTTP,
		clock:      fakeClock,
	}

	err := c.sendAgentHit("test-node")

	assert.Error(t, err)
}

// makeListResponse serializes a listing envelope as an HTTP 200 body.
func makeListResponse(payload any) *http.Response {
	body, _ := json.Marshal(payload)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBuffer(body)),
	}
}

// redirectsRange builds n redirects sourced /r{start}../r{start+n-1}, so a walk
// can be checked for order and for duplicates.
func redirectsRange(start, n int) []types.Redirect {
	items := make([]types.Redirect, n)
	for i := 0; i < n; i++ {
		items[i] = types.Redirect{
			Type:   types.RedirectTypeBasic,
			Source: fmt.Sprintf("/r%d", start+i),
			Target: "/target",
			Status: types.RedirectStatusMovedPermanent,
		}
	}
	return items
}

func redirectSources(redirects []types.Redirect) []string {
	sources := make([]string, len(redirects))
	for i := range redirects {
		sources[i] = redirects[i].Source
	}
	return sources
}

// pagesRange builds n pages pathed /p{start}../p{start+n-1}.
func pagesRange(start, n int) []types.Page {
	items := make([]types.Page, n)
	for i := 0; i < n; i++ {
		items[i] = types.Page{
			Type:        types.PageTypeBasic,
			Path:        fmt.Sprintf("/p%d", start+i),
			Content:     "content",
			ContentType: types.PageContentTypeTextPlain,
		}
	}
	return items
}

func pagePaths(pages []types.Page) []string {
	paths := make([]string, len(pages))
	for i := range pages {
		paths[i] = pages[i].Path
	}
	return paths
}

func queriesOf(calls []*http.Request) []string {
	queries := make([]string, 0, len(calls))
	for _, req := range calls {
		queries = append(queries, req.URL.RawQuery)
	}
	return queries
}

func TestClient_getProjectRedirects_Walk(t *testing.T) {
	type page struct {
		statusCode int // 0 serves list below with a 200
		list       types.RedirectList
	}

	tests := []struct {
		name        string
		limit       int
		pages       []page
		wantSources []string
		// wantQueries is compared whole, so it also proves a cursor request
		// carries no offset and vice versa.
		wantQueries []string
		wantErr     string
	}{
		{
			name:  "cursor walk over several pages",
			limit: 2,
			pages: []page{
				{list: types.RedirectList{Items: redirectsRange(0, 2), Total: 5, Limit: 2, Offset: 0, Next: "cur/1+="}},
				{list: types.RedirectList{Items: redirectsRange(2, 2), Total: 5, Limit: 2, Offset: 2, Next: "cur2"}},
				{list: types.RedirectList{Items: redirectsRange(4, 1), Total: 5, Limit: 2, Offset: 4}},
			},
			wantSources: []string{"/r0", "/r1", "/r2", "/r3", "/r4"},
			wantQueries: []string{"limit=2&offset=0", "limit=2&cursor=cur%2F1%2B%3D", "limit=2&cursor=cur2"},
		},
		{
			name:  "old server without cursor walks by offset",
			limit: 2,
			pages: []page{
				{list: types.RedirectList{Items: redirectsRange(0, 2), Total: 5, Limit: 2, Offset: 0}},
				{list: types.RedirectList{Items: redirectsRange(2, 2), Total: 5, Limit: 2, Offset: 2}},
				{list: types.RedirectList{Items: redirectsRange(4, 1), Total: 5, Limit: 2, Offset: 4}},
			},
			wantSources: []string{"/r0", "/r1", "/r2", "/r3", "/r4"},
			wantQueries: []string{"limit=2&offset=0", "limit=2&offset=2", "limit=2&offset=4"},
		},
		{
			name:  "short page ends the walk",
			limit: 5,
			pages: []page{
				{list: types.RedirectList{Items: redirectsRange(0, 3), Total: 99, Limit: 5, Offset: 0}},
			},
			wantSources: []string{"/r0", "/r1", "/r2"},
			wantQueries: []string{"limit=5&offset=0"},
		},
		{
			name:  "empty page ends the walk even when a cursor is still served",
			limit: 2,
			pages: []page{
				{list: types.RedirectList{Items: redirectsRange(0, 2), Total: 100, Limit: 2, Offset: 0, Next: "cur1"}},
				{list: types.RedirectList{Items: []types.Redirect{}, Total: 100, Limit: 2, Offset: 2, Next: "cur2"}},
			},
			wantSources: []string{"/r0", "/r1"},
			wantQueries: []string{"limit=2&offset=0", "limit=2&cursor=cur1"},
		},
		{
			name:  "non 200 mid walk propagates the error",
			limit: 2,
			pages: []page{
				{list: types.RedirectList{Items: redirectsRange(0, 2), Total: 5, Limit: 2, Offset: 0, Next: "cur1"}},
				{statusCode: http.StatusInternalServerError},
			},
			wantErr:     "unexpected status code for http://localhost:8080/api/namespace/test-ns/project/test-proj/redirects",
			wantQueries: []string{"limit=2&offset=0", "limit=2&cursor=cur1"},
		},
		{
			name:  "unset limit falls back to the default",
			limit: 0,
			pages: []page{
				{list: types.RedirectList{Items: redirectsRange(0, 1), Total: 1, Limit: DefaultRedirectsLimit, Offset: 0}},
			},
			wantSources: []string{"/r0"},
			wantQueries: []string{fmt.Sprintf("limit=%d&offset=0", DefaultRedirectsLimit)},
		},
		{
			name:  "negative limit falls back to the default",
			limit: -5,
			pages: []page{
				{list: types.RedirectList{Items: redirectsRange(0, 1), Total: 1, Limit: DefaultRedirectsLimit, Offset: 0}},
			},
			wantSources: []string{"/r0"},
			wantQueries: []string{fmt.Sprintf("limit=%d&offset=0", DefaultRedirectsLimit)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, mockHTTP, _ := newTestClient()
			c.cfg.RedirectsLimit = tt.limit
			for _, p := range tt.pages {
				if p.statusCode != 0 {
					mockHTTP.expect(makeErrorResponse(p.statusCode), nil)
					continue
				}
				mockHTTP.expect(makeListResponse(p.list), nil)
			}

			result, err := c.getProjectRedirects()

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSources, redirectSources(result))
			}
			assert.Equal(t, tt.wantQueries, queriesOf(mockHTTP.calls))
		})
	}
}

func TestClient_getProjectPages_Walk(t *testing.T) {
	type page struct {
		statusCode int // 0 serves list below with a 200
		list       types.PageList
	}

	tests := []struct {
		name        string
		limit       int
		pages       []page
		wantPaths   []string
		wantQueries []string
		wantErr     string
	}{
		{
			name:  "cursor walk over several pages",
			limit: 2,
			pages: []page{
				{list: types.PageList{Items: pagesRange(0, 2), Total: 5, Limit: 2, Offset: 0, Next: "cur1"}},
				{list: types.PageList{Items: pagesRange(2, 2), Total: 5, Limit: 2, Offset: 2, Next: "cur2"}},
				{list: types.PageList{Items: pagesRange(4, 1), Total: 5, Limit: 2, Offset: 4}},
			},
			wantPaths:   []string{"/p0", "/p1", "/p2", "/p3", "/p4"},
			wantQueries: []string{"limit=2&offset=0", "limit=2&cursor=cur1", "limit=2&cursor=cur2"},
		},
		{
			name:  "old server without cursor walks by offset",
			limit: 2,
			pages: []page{
				{list: types.PageList{Items: pagesRange(0, 2), Total: 3, Limit: 2, Offset: 0}},
				{list: types.PageList{Items: pagesRange(2, 1), Total: 3, Limit: 2, Offset: 2}},
			},
			wantPaths:   []string{"/p0", "/p1", "/p2"},
			wantQueries: []string{"limit=2&offset=0", "limit=2&offset=2"},
		},
		{
			name:  "empty page ends the walk even when a cursor is still served",
			limit: 2,
			pages: []page{
				{list: types.PageList{Items: pagesRange(0, 2), Total: 100, Limit: 2, Offset: 0, Next: "cur1"}},
				{list: types.PageList{Items: []types.Page{}, Total: 100, Limit: 2, Offset: 2, Next: "cur2"}},
			},
			wantPaths:   []string{"/p0", "/p1"},
			wantQueries: []string{"limit=2&offset=0", "limit=2&cursor=cur1"},
		},
		{
			name:  "non 200 mid walk propagates the error",
			limit: 2,
			pages: []page{
				{list: types.PageList{Items: pagesRange(0, 2), Total: 5, Limit: 2, Offset: 0, Next: "cur1"}},
				{statusCode: http.StatusInternalServerError},
			},
			wantErr:     "unexpected status code for http://localhost:8080/api/namespace/test-ns/project/test-proj/pages",
			wantQueries: []string{"limit=2&offset=0", "limit=2&cursor=cur1"},
		},
		{
			name:  "unset limit falls back to the default",
			limit: 0,
			pages: []page{
				{list: types.PageList{Items: pagesRange(0, 1), Total: 1, Limit: DefaultPagesLimit, Offset: 0}},
			},
			wantPaths:   []string{"/p0"},
			wantQueries: []string{fmt.Sprintf("limit=%d&offset=0", DefaultPagesLimit)},
		},
		{
			name:  "negative limit falls back to the default",
			limit: -5,
			pages: []page{
				{list: types.PageList{Items: pagesRange(0, 1), Total: 1, Limit: DefaultPagesLimit, Offset: 0}},
			},
			wantPaths:   []string{"/p0"},
			wantQueries: []string{fmt.Sprintf("limit=%d&offset=0", DefaultPagesLimit)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, mockHTTP, _ := newTestClient()
			c.cfg.PagesLimit = tt.limit
			for _, p := range tt.pages {
				if p.statusCode != 0 {
					mockHTTP.expect(makeErrorResponse(p.statusCode), nil)
					continue
				}
				mockHTTP.expect(makeListResponse(p.list), nil)
			}

			result, err := c.getProjectPages()

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantPaths, pagePaths(result))
			}
			assert.Equal(t, tt.wantQueries, queriesOf(mockHTTP.calls))
		})
	}
}

// trackedBody counts Close calls so a leaked connection shows up as zero.
type trackedBody struct {
	io.Reader
	closed int
}

func (b *trackedBody) Close() error {
	b.closed++
	return nil
}

func TestClient_getProjectRedirects_ClosesBody(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		payload    string
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			payload:    `{"Items":[],"Total":0,"Limit":2,"Offset":0}`,
		},
		{
			name:       "error status",
			statusCode: http.StatusForbidden,
			payload:    "forbidden",
		},
		{
			name:       "invalid json",
			statusCode: http.StatusOK,
			payload:    "invalid json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, mockHTTP, _ := newTestClient()
			body := &trackedBody{Reader: bytes.NewBufferString(tt.payload)}
			mockHTTP.expect(&http.Response{
				StatusCode: tt.statusCode,
				Status:     http.StatusText(tt.statusCode),
				Body:       body,
			}, nil)

			_, _ = c.getProjectRedirects()

			assert.Equal(t, 1, body.closed)
		})
	}
}

func TestClient_listingLimitsAreIndependent(t *testing.T) {
	c, mockHTTP, _ := newTestClient()
	c.cfg.RedirectsLimit = 2
	c.cfg.PagesLimit = 3

	mockHTTP.expect(makeListResponse(types.RedirectList{Items: redirectsRange(0, 1), Total: 1, Limit: 2, Offset: 0}), nil)
	mockHTTP.expect(makeListResponse(types.PageList{Items: pagesRange(0, 1), Total: 1, Limit: 3, Offset: 0}), nil)

	redirects, errRedirects := c.getProjectRedirects()
	pages, errPages := c.getProjectPages()

	assert.NoError(t, errRedirects)
	assert.NoError(t, errPages)
	assert.Equal(t, []string{"/r0"}, redirectSources(redirects))
	assert.Equal(t, []string{"/p0"}, pagePaths(pages))
	assert.Equal(t, []string{"limit=2&offset=0", "limit=3&offset=0"}, queriesOf(mockHTTP.calls))
}
