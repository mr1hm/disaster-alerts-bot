package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Disaster struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Magnitude float64   `json:"magnitude"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Location  string    `json:"location"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
}

type QueryOptions struct {
	MinMagnitude  *float64
	DisasterTypes []string
	Since         *time.Time
}

type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *APIClient) GetDisasters(ctx context.Context, opts QueryOptions) ([]Disaster, error) {
	u, err := url.Parse(c.baseURL + "/disasters")
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()
	if opts.MinMagnitude != nil {
		q.Set("min_magnitude", fmt.Sprintf("%.1f", *opts.MinMagnitude))
	}
	if len(opts.DisasterTypes) > 0 {
		for _, t := range opts.DisasterTypes {
			q.Add("type", t)
		}
	}
	if opts.Since != nil {
		q.Set("since", opts.Since.Format(time.RFC3339))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching disasters: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var disasters []Disaster
	if err := json.NewDecoder(resp.Body).Decode(&disasters); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return disasters, nil
}
