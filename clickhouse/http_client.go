package clickhouse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// httpClient communicates with ClickHouse via its HTTP interface (port 8123).
// This avoids any native driver dependency and works with all ClickHouse
// versions that support the HTTP interface (i.e., all of them).
type httpClient struct {
	baseURL  string
	database string
	user     string
	password string
	hc       *http.Client
}

func newHTTPClient(dsn, defaultDatabase string) (*httpClient, error) {
	base, database, user, password, err := parseDSN(dsn)
	if err != nil {
		return nil, err
	}
	if database == "" {
		database = defaultDatabase
	}
	return &httpClient{
		baseURL:  base,
		database: database,
		user:     user,
		password: password,
		hc:       &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// ping executes a simple SELECT 1 to verify connectivity.
func (c *httpClient) ping(ctx context.Context) error {
	return c.exec(ctx, "SELECT 1")
}

// exec sends a DDL or non-SELECT statement.
func (c *httpClient) exec(ctx context.Context, query string) error {
	endpoint := c.endpoint(query, "")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, http.NoBody)
	if err != nil {
		return err
	}
	c.addAuth(req)
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("clickhouse exec failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// insertJSONEachRow sends rows encoded as newline-delimited JSON objects.
func (c *httpClient) insertJSONEachRow(ctx context.Context, table string, rows []map[string]any) error {
	if len(rows) == 0 {
		return nil
	}
	var buf bytes.Buffer
	for _, row := range rows {
		b, err := json.Marshal(row)
		if err != nil {
			return fmt.Errorf("marshal row: %w", err)
		}
		buf.Write(b)
		buf.WriteByte('\n')
	}

	query := fmt.Sprintf("INSERT INTO `%s`.`%s` FORMAT JSONEachRow", c.database, table)
	endpoint := c.endpoint(query, "")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &buf)
	if err != nil {
		return err
	}
	c.addAuth(req)
	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("clickhouse insert failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (c *httpClient) endpoint(query, format string) string {
	params := url.Values{}
	params.Set("query", query)
	if format != "" {
		params.Set("default_format", format)
	}
	return c.baseURL + "/?" + params.Encode()
}

func (c *httpClient) addAuth(req *http.Request) {
	if c.user != "" {
		req.SetBasicAuth(c.user, c.password)
	}
}
