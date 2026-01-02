package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/flectolab/flecto-manager/common/types"
	"github.com/jonboulle/clockwork"
)

type Client interface {
	Init() error
	GetStateVersion() int
	RedirectMatch(host, uri string) (*types.Redirect, string)
	PageMatch(host, uri string) *types.Page
	Reload() error
	Start(ctx context.Context)
}

func New(cfg *Config) Client {
	c := &client{cfg: cfg, httpClient: cfg.Http.Client, clock: clockwork.NewRealClock()}
	c.State.Store(&State{})
	return c
}

type State struct {
	ProjectVersion  int
	RedirectMatcher types.RedirectTreeMatcher
	PageMatcher     types.PageTreeMatcher
}

type client struct {
	cfg        *Config
	httpClient HTTPClient
	State      atomic.Value
	clock      clockwork.Clock
	reloadMu   sync.Mutex
}

func (c *client) Init() error {
	err := c.loadState()
	if err != nil {
		return err
	}

	return nil
}

func (c *client) load() *State {
	return c.State.Load().(*State)
}

func (c *client) RedirectMatch(host, uri string) (*types.Redirect, string) {
	return c.load().RedirectMatcher.Match(host, uri)
}
func (c *client) PageMatch(host, uri string) *types.Page {
	return c.load().PageMatcher.Match(host, uri)
}

func (c *client) GetStateVersion() int {
	return c.load().ProjectVersion
}
func (c *client) Reload() error {
	if !c.reloadMu.TryLock() {
		return nil
	}
	defer c.reloadMu.Unlock()

	version, err := c.getProjectVersion()
	if err != nil {
		return err
	}
	if version != c.load().ProjectVersion {
		err = c.loadState()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *client) Start(ctx context.Context) {
	ticker := c.clock.NewTimer(c.cfg.IntervalCheck)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.Chan():
			_ = c.Reload()
		case <-ctx.Done():
			return
		}
	}
}

func (c *client) loadState() error {
	redirectTreeMatcher := types.NewRedirectTreeMatcher()
	pagesTreeMatcher := types.NewPageTreeMatcher()
	version, errVersion := c.getProjectVersion()
	if errVersion != nil {
		return errVersion
	}

	redirects, errRedirects := c.getProjectRedirects()
	if errRedirects != nil {
		return errRedirects
	}

	for _, redirect := range redirects {
		err := redirectTreeMatcher.Insert(&redirect)
		if err != nil {
			return err
		}
	}

	pages, errPages := c.getProjectPages()
	if errPages != nil {
		return errPages
	}
	for _, page := range pages {
		pagesTreeMatcher.Insert(&page)
	}
	state := &State{ProjectVersion: version, RedirectMatcher: redirectTreeMatcher, PageMatcher: pagesTreeMatcher}
	c.State.Store(state)
	return nil
}

func (c *client) getProjectVersion() (int, error) {
	req, err := NewRequest(c.cfg.Http, http.MethodGet, c.cfg.GetUrlApiVersion(), nil)
	if err != nil {
		return 0, err
	}
	resp, errReq := c.httpClient.Do(req)
	if errReq != nil {
		return 0, errReq
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code for %s: %s (%d) %v", c.cfg.GetUrlApiVersion(), resp.Status, resp.StatusCode, resp.Body)
	}

	body, errReadBody := io.ReadAll(resp.Body)
	if errReadBody != nil {
		return 0, errReadBody
	}
	version, errCastInt := strconv.Atoi(strings.TrimSpace(string(body)))
	if errCastInt != nil {
		return 0, errCastInt
	}

	return version, nil
}

func (c *client) getProjectRedirects() ([]types.Redirect, error) {
	redirects := make([]types.Redirect, 0)
	offset := 0
	limit := 100
	for {
		redirectList := types.RedirectList{}
		url := fmt.Sprintf("%s?limit=%d&offset=%d", c.cfg.GetUrlApiRedirects(), limit, offset)
		req, err := NewRequest(c.cfg.Http, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		resp, errReq := c.httpClient.Do(req)
		if errReq != nil {
			return nil, errReq
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code for %s: %s (%d) %v", c.cfg.GetUrlApiRedirects(), resp.Status, resp.StatusCode, resp.Body)
		}

		err = json.NewDecoder(resp.Body).Decode(&redirectList)
		if err != nil {
			return nil, err
		}
		_ = resp.Body.Close()
		redirects = append(redirects, redirectList.Items...)
		offset += limit
		if offset >= redirectList.Total {
			break
		}
	}

	return redirects, nil
}

func (c *client) getProjectPages() ([]types.Page, error) {
	pages := make([]types.Page, 0)
	offset := 0
	limit := 100
	for {
		pageList := types.PageList{}
		url := fmt.Sprintf("%s?limit=%d&offset=%d", c.cfg.GetUrlApiPages(), limit, offset)
		req, err := NewRequest(c.cfg.Http, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		resp, errReq := c.httpClient.Do(req)
		if errReq != nil {
			return nil, errReq
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code for %s: %s (%d) %v", c.cfg.GetUrlApiPages(), resp.Status, resp.StatusCode, resp.Body)
		}

		err = json.NewDecoder(resp.Body).Decode(&pageList)
		if err != nil {
			return nil, err
		}
		_ = resp.Body.Close()
		pages = append(pages, pageList.Items...)
		offset += limit
		if offset >= pageList.Total {
			break
		}
	}

	return pages, nil
}
