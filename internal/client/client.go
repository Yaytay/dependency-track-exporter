package client

import (
	"context"
	"dependency-track-exporter/internal/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *config.Logger
}

func NewClient(baseURL, apiKey string, timeout time.Duration, logger *config.Logger) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

type Project struct {
	UUID       string            `json:"uuid"`
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Classifier string            `json:"classifier"`
	Metrics    ProjectMetrics    `json:"metrics"`
	Tags       []ProjectTag      `json:"tags,omitempty"`
	Properties []ProjectProperty `json:"properties,omitempty"`
}

type ProjectMetrics struct {
	Critical   int `json:"critical"`
	High       int `json:"high"`
	Medium     int `json:"medium"`
	Low        int `json:"low"`
	Unassigned int `json:"unassigned"`
}

type ProjectTag struct {
	Name string `json:"name"`
}

type ProjectProperty struct {
	GroupName     string `json:"groupName"`
	PropertyName  string `json:"propertyName"`
	PropertyValue string `json:"propertyValue"`
	PropertyType  string `json:"propertyType"`
	Description   string `json:"description"`
}

type VulnerabilityCounts struct {
	Critical   int
	High       int
	Medium     int
	Low        int
	Unassigned int
}

type ProjectSnapshot struct {
	Project Project
	Counts  VulnerabilityCounts
}

func (c *Client) FetchProjectSnapshots(ctx context.Context) ([]ProjectSnapshot, error) {
	projects, err := c.fetchProjects(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]ProjectSnapshot, 0, len(projects))
	for _, p := range projects {
		out = append(out, ProjectSnapshot{
			Project: p,
			Counts: VulnerabilityCounts{
				Critical:   p.Metrics.Critical,
				High:       p.Metrics.High,
				Medium:     p.Metrics.Medium,
				Low:        p.Metrics.Low,
				Unassigned: p.Metrics.Unassigned,
			},
		})
	}
	return out, nil
}

func (c *Client) fetchProjects(ctx context.Context) ([]Project, error) {
	const pageSize = 100

	projects := make([]Project, 0, pageSize)
	for pageNumber := 1; ; pageNumber++ {
		u, err := url.Parse(c.baseURL + "/api/v1/project")
		if err != nil {
			return nil, err
		}

		q := u.Query()
		q.Set("pageNumber", strconv.Itoa(pageNumber))
		q.Set("pageSize", strconv.Itoa(pageSize))
		u.RawQuery = q.Encode()

		c.logger.Debug("fetching projects page", "page_number", pageNumber, "page_size", pageSize)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Api-Key", c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var page []Project
		err = json.NewDecoder(resp.Body).Decode(&page)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		projects = append(projects, page...)

		if len(page) < pageSize {
			break
		}
	}

	c.logger.Debug("fetched all projects", "projects", len(projects))

	return projects, nil
}

func (c *Client) fetchProjectVulnerabilityCounts(ctx context.Context, projectUUID string) (VulnerabilityCounts, error) {
	u, err := url.Parse(c.baseURL + "/api/v1/finding/project/" + projectUUID)
	if err != nil {
		return VulnerabilityCounts{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return VulnerabilityCounts{}, err
	}
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return VulnerabilityCounts{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return VulnerabilityCounts{}, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var findings []struct {
		Vulnerability struct {
			Severity string `json:"severity"`
		} `json:"vulnerability"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&findings); err != nil {
		return VulnerabilityCounts{}, err
	}

	var counts VulnerabilityCounts
	for _, f := range findings {
		switch strings.ToLower(f.Vulnerability.Severity) {
		case "critical":
			counts.Critical++
		case "high":
			counts.High++
		case "medium":
			counts.Medium++
		case "low":
			counts.Low++
		default:
			counts.Unassigned++
		}
	}
	return counts, nil
}
