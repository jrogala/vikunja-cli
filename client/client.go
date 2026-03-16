// Package client provides an HTTP client for the Vikunja API v2.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jrogala/vikunja-cli/config"
)

// Client is an authenticated Vikunja API client.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// New creates a Client from the given config.
func New(cfg *config.Config) *Client {
	baseURL := strings.TrimRight(cfg.URL, "/")
	if !strings.HasSuffix(baseURL, "/api/v1") {
		baseURL += "/api/v1"
	}
	return &Client{
		baseURL:    baseURL,
		token:      cfg.Token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) do(method, path string, body io.Reader) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}

// Task represents a Vikunja task.
type Task struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
	Priority    int    `json:"priority"`
	DueDate     string `json:"due_date"`
	ProjectID   int64  `json:"project_id"`
	Created     string `json:"created"`
	Updated     string `json:"updated"`
}

// Project represents a Vikunja project.
type Project struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	IsArchived  bool   `json:"is_archived"`
	IsFavorite  bool   `json:"is_favorite"`
}

// GetTasks returns all tasks matching the given params.
func (c *Client) GetTasks(params map[string]string) ([]Task, error) {
	path := "/tasks"
	if len(params) > 0 {
		v := url.Values{}
		for k, val := range params {
			v.Set(k, val)
		}
		path += "?" + v.Encode()
	}
	data, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetProjectTasks returns tasks for a specific project.
func (c *Client) GetProjectTasks(projectID int64, params map[string]string) ([]Task, error) {
	path := fmt.Sprintf("/projects/%d/tasks", projectID)
	if len(params) > 0 {
		v := url.Values{}
		for k, val := range params {
			v.Set(k, val)
		}
		path += "?" + v.Encode()
	}
	data, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetTask returns a single task by ID.
func (c *Client) GetTask(id int64) (*Task, error) {
	data, err := c.do("GET", fmt.Sprintf("/tasks/%d", id), nil)
	if err != nil {
		return nil, err
	}
	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// CreateTask creates a task in the given project.
func (c *Client) CreateTask(projectID int64, task map[string]any) (*Task, error) {
	body, err := json.Marshal(task)
	if err != nil {
		return nil, err
	}
	data, err := c.do("PUT", fmt.Sprintf("/projects/%d/tasks", projectID), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	var t Task
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateTask merges updates into an existing task (GET-merge-POST).
func (c *Client) UpdateTask(id int64, updates map[string]any) (*Task, error) {
	current, err := c.GetTask(id)
	if err != nil {
		return nil, fmt.Errorf("fetch task before update: %w", err)
	}
	full := map[string]any{
		"title":       current.Title,
		"description": current.Description,
		"done":        current.Done,
		"priority":    current.Priority,
		"due_date":    current.DueDate,
		"project_id":  current.ProjectID,
	}
	for k, v := range updates {
		full[k] = v
	}
	body, err := json.Marshal(full)
	if err != nil {
		return nil, err
	}
	data, err := c.do("POST", fmt.Sprintf("/tasks/%d", id), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	var t Task
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// CompleteTask marks a task as done.
func (c *Client) CompleteTask(id int64) (*Task, error) {
	return c.UpdateTask(id, map[string]any{"done": true})
}

// DeleteTask permanently removes a task.
func (c *Client) DeleteTask(id int64) error {
	_, err := c.do("DELETE", fmt.Sprintf("/tasks/%d", id), nil)
	return err
}

// GetProjects returns all projects.
func (c *Client) GetProjects() ([]Project, error) {
	data, err := c.do("GET", "/projects", nil)
	if err != nil {
		return nil, err
	}
	var projects []Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}
