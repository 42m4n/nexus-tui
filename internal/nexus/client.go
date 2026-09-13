package nexus

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	base   string
	user   string
	pass   string
	token  string
	http   *http.Client
	Writes bool
}

func New(rawURL string, insecure bool, username, password, token string) (*Client, error) {
	rawURL = strings.TrimRight(rawURL, "/")
	if !strings.Contains(rawURL, "://") {
		rawURL = "http://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	tr := &http.Transport{}
	if insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // ponytail: opt-in via --insecure, expected for self-signed Nexus
	}
	return &Client{
		base:  u.String() + "/service/rest/v1",
		user:  username,
		pass:  password,
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second, Transport: tr},
	}, nil
}

func (c *Client) auth(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
		return
	}
	if c.user != "" {
		req.SetBasicAuth(c.user, c.pass)
	}
}

// do performs a request and decodes a JSON response when out is non-nil.
func (c *Client) do(method, path string, query url.Values, out any) error {
	u := c.base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequest(method, u, nil)
	if err != nil {
		return err
	}
	c.auth(req)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(body)))
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) get(path string, query url.Values, out any) error {
	return c.do(http.MethodGet, path, query, out)
}

// getPage fetches every page of a continuation-token paged endpoint.
func getPage[T any](c *Client, path string, query url.Values, extract func(page[T]) []T) ([]T, error) {
	var all []T
	token := ""
	for {
		q := url.Values{}
		for k, v := range query {
			q[k] = v
		}
		if token != "" {
			q.Set("continuationToken", token)
		}
		var p page[T]
		if err := c.get(path, q, &p); err != nil {
			return all, err
		}
		all = append(all, extract(p)...)
		if p.ContinuationToken == "" {
			return all, nil
		}
		token = p.ContinuationToken
	}
}

func (c *Client) Status() (string, error) {
	if _, err := c.getRaw("/status"); err != nil {
		return "", err
	}
	if _, err := c.getRaw("/status/writable"); err != nil {
		return "read-only", nil
	}
	return "writable", nil
}

// getRaw returns the HTTP status code of a bodiless endpoint.
func (c *Client) getRaw(path string) (int, error) {
	req, err := http.NewRequest(http.MethodGet, c.base+path, nil)
	if err != nil {
		return 0, err
	}
	c.auth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp.StatusCode, nil
	}
	return resp.StatusCode, fmt.Errorf("%s: %s", path, resp.Status)
}

func (c *Client) Repositories() ([]Repository, error) {
	var repos []Repository
	err := c.get("/repositories", nil, &repos)
	return repos, err
}

func (c *Client) Components(repository string) ([]Component, error) {
	if repository == "" {
		return nil, fmt.Errorf("repository required")
	}
	return getPage(c, "/components", url.Values{"repository": {repository}},
		func(p page[Component]) []Component { return p.Items })
}

func (c *Client) Search(q url.Values) ([]Component, error) {
	return getPage(c, "/search", q, func(p page[Component]) []Component { return p.Items })
}

func (c *Client) Tasks() ([]Task, error) {
	return getPage(c, "/tasks", nil, func(p page[Task]) []Task { return p.Items })
}

func (c *Client) Users() ([]User, error) {
	var out []User
	err := c.get("/security/users", nil, &out)
	return out, err
}

func (c *Client) Roles() ([]Role, error) {
	var out []Role
	err := c.get("/security/roles", nil, &out)
	return out, err
}

func (c *Client) Privileges() ([]Privilege, error) {
	var out []Privilege
	err := c.get("/security/privileges", nil, &out)
	return out, err
}

func (c *Client) BlobStores() ([]BlobStore, error) {
	var out []BlobStore
	err := c.get("/blobstores", nil, &out)
	return out, err
}

// Delete removes a resource. Guarded by Writes and used only for the two
// destructive actions in scope: repositories and users.
func (c *Client) Delete(path string) error {
	if !c.Writes {
		return fmt.Errorf("writes are disabled; start with --allow-writes")
	}
	return c.do(http.MethodDelete, path, nil, nil)
}

// postJSON runs a bodyless POST (tasks run/stop, license, etc).
func (c *Client) Post(path string) error {
	var buf bytes.Buffer
	req, err := http.NewRequest(http.MethodPost, c.base+path, &buf)
	if err != nil {
		return err
	}
	c.auth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("POST %s: %s", path, resp.Status)
}
