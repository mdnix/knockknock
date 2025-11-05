package ipc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/Masterminds/semver/v3"
)

type Client struct {
	socketPath string
	httpClient *http.Client
}

func NewClient(socketPath string) (*Client, error) {
	httpClient := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socketPath)
			},
		},
	}

	return &Client{
		socketPath: socketPath,
		httpClient: httpClient,
	}, nil
}

func (c *Client) Versions(ctx context.Context) ([]semver.Version, error) {
	resp, err := c.versions()

	if err != nil {
		return nil, err
	}

	return resp.Versions, nil
}

func (c *Client) CheckForUpdate(ctx context.Context) (*semver.Version, []semver.Version, error) {
	resp, err := c.versions()

	if err != nil {
		return nil, nil, err
	}

	if len(resp.Versions) == 0 {
		return nil, nil, fmt.Errorf("no versions found in repository")
	}

	return resp.Update, resp.Versions, nil
}

func (c *Client) Update(ctx context.Context, version string) error {
	reqBody := UpdateRequest{
		Version: version,
	}

	body, err := json.Marshal(reqBody)

	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://unix/update", bytes.NewReader(body))

	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("failed to send update request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var updateResp UpdateResponse

	if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !updateResp.Success {
		return fmt.Errorf("update failed: %s", updateResp.Message)
	}

	return nil
}

func (c *Client) Rollback(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://unix/rollback", nil)

	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("failed to send rollback request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("rollback request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var rollbackResp RollbackResponse

	if err := json.NewDecoder(resp.Body).Decode(&rollbackResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !rollbackResp.Success {
		return fmt.Errorf("rollback failed: %s", rollbackResp.Message)
	}

	return nil
}

func (c *Client) History(ctx context.Context) ([]HistoryEntry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix/history", nil)

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("history request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var historyResp HistoryResponse

	if err := json.NewDecoder(resp.Body).Decode(&historyResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return historyResp.History, nil
}

func (c *Client) versions() (*VersionsResponse, error) {
	resp, err := c.httpClient.Get("http://unix/versions")

	if err != nil {
		return nil, fmt.Errorf("failed to query supervisor: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("supervisor returned status %d", resp.StatusCode)
	}

	var data VersionsResponse

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &data, nil
}

func parseVersions(versions []string) []semver.Version {
	result := make([]semver.Version, len(versions))

	for i, v := range versions {
		version, err := semver.NewVersion(v)

		if err != nil {
			slog.Warn("received invalid semver version from ipc server", "version", v)

			continue
		}

		result[i] = *version
	}

	return result
}
