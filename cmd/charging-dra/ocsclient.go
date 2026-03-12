package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-ocs/internal/logging"
	"go-ocs/internal/nchf"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type OcsClient struct {
	mu      sync.RWMutex
	clients map[string]*http.Client
	// shared defaults:
	timeout time.Duration
}

// NewOcsClient creates a new OcsClient with a shared default HTTP transport based on the provided configuration.
func NewOcsClient(config *Config) *OcsClient {

	transport := &http.Transport{
		MaxIdleConns:        config.ChargingDRA.MaxIdleConns,
		MaxIdleConnsPerHost: config.ChargingDRA.MaxIdleConnsPerHost,
		IdleConnTimeout:     config.ChargingDRA.IdleConnTimeout,
	}

	client := &http.Client{
		Transport: transport,
	}

	return &OcsClient{
		clients: map[string]*http.Client{"": client},
	}
}

// getClient returns (and lazily creates if needed) an HTTP client keyed by scheme and host for connection reuse.
func (c *OcsClient) getClient(rawURL string) (*http.Client, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", err
	}

	host := u.Host
	if !strings.Contains(host, ":") {
		if u.Scheme == "https" {
			host += ":443"
		} else {
			host += ":80"
		}
	}
	key := u.Scheme + "://" + host

	c.mu.RLock()
	cl := c.clients[key]
	c.mu.RUnlock()
	if cl != nil {
		return cl, key, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if cl = c.clients[key]; cl != nil {
		return cl, key, nil
	}

	tr := &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     90 * time.Second,
		// plus timeouts like TLSHandshakeTimeout etc
	}
	cl = &http.Client{
		Transport: tr,
		// Prefer ctx timeouts; keep this unset or set as a hard cap
		Timeout: 0,
	}
	c.clients[key] = cl
	return cl, key, nil
}

// Send dispatches the ChargingDataRequest to the appropriate OCS endpoint based on its request type.
func (c *OcsClient) Send(ctx context.Context, ocsURL string, payload *nchf.ChargingDataRequest) (*nchf.ChargingDataResponse, error) {

	switch payload.GetRequestType() {
	case 1:
		return c.Charging(ctx, ocsURL, payload)
	case 2:
		return c.Update(ctx, ocsURL, payload)
	case 3:
		return c.Release(ctx, ocsURL, payload)
	default:
		return nil, fmt.Errorf("unsupported request type: %d", payload.GetRequestType())
	}
}

// Charging sends a new charging request to the OCS.
func (c *OcsClient) Charging(ctx context.Context, ocsURL string, req *nchf.ChargingDataRequest) (*nchf.ChargingDataResponse, error) {
	return c.post(ctx, ocsURL, "/chargingdata", req)
}

// Update sends an update request for an existing charging session to the OCS.
func (c *OcsClient) Update(ctx context.Context, ocsURL string, req *nchf.ChargingDataRequest) (*nchf.ChargingDataResponse, error) {
	return c.post(ctx, ocsURL, fmt.Sprintf("/chargingdata/%s/update", url.PathEscape(*req.ChargingId)), req)
}

// Release sends a release request for an existing charging session to the OCS.
func (c *OcsClient) Release(ctx context.Context, ocsURL string, req *nchf.ChargingDataRequest) (*nchf.ChargingDataResponse, error) {
	return c.post(ctx, ocsURL, fmt.Sprintf("/chargingdata/%s/release", url.PathEscape(*req.ChargingId)), req)
}

// post performs the HTTP POST to the OCS, marshals the request payload, and decodes the response.
func (c *OcsClient) post(ctx context.Context, ocsURL string, path string, payload *nchf.ChargingDataRequest) (*nchf.ChargingDataResponse, error) {
	cl, _, err := c.getClient(ocsURL)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	nchfUrl := ocsURL + path

	// Defensive copy: avoids any chance of the underlying byte slice being mutated by other code
	// (rare, but can happen with pooling / reuse patterns).
	bodyBytes := append([]byte(nil), b...)

	// Log a small preview to confirm we are not sending an all-zero payload.
	// Keep it short to avoid log spam / sensitive data leakage.
	preview := string(bodyBytes)
	if len(preview) > 512 {
		preview = preview[:512] + "..."
	}
	logging.Debug("Sending ocs request", "url", nchfUrl, "bodyLen", len(bodyBytes), "bodyPreview", preview)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, nchfUrl, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.ContentLength = int64(len(bodyBytes))

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := cl.Do(httpReq)
	if err != nil {
		return nil, err // caller can map to 5012, etc.
	}
	defer resp.Body.Close()

	out := nchf.NewChargingDataResponse()
	out.SetRequestType(payload.GetRequestType())

	// Non-2xx: optionally decode ProblemDetails if your Nchf service returns it
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		out.InvocationResult = &nchf.InvocationResult{Error: &nchf.ProblemDetails{}}
		if err := json.NewDecoder(resp.Body).Decode(out.InvocationResult.Error); err != nil {
			return nil, fmt.Errorf("decode problem response: %w", err)
		}
		logging.Error("ocs request failed", "ocsURL", ocsURL, "statusCode", resp.StatusCode, "error", out.InvocationResult.Error)
	} else {
		raw, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("read response body: %w", readErr)
		}

		logging.Debug("OCS response raw",
			"status", resp.StatusCode,
			"len", len(raw),
			"body", string(raw),
		)

		// Now decode from the bytes
		if err := json.Unmarshal(raw, out); err != nil {
			// include more detail below
			return nil, fmt.Errorf("decode response: %w", err)
		}
	}

	return out, nil
}
