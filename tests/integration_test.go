package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// testState holds per-scenario state.
type testState struct {
	baseURL    string
	token      string
	output     string
	taskIDs    map[string]int64
	projectIDs map[string]int64
	binaryPath string
	configDir  string // temp config dir for setup tests
}

// vikunjaContainer holds the shared container across all scenarios.
var vikunjaContainer testcontainers.Container
var vikunjaURL string
var vikunjaToken string
var binaryPath string

func TestFeatures(t *testing.T) {
	// Build the CLI binary
	binaryPath = buildBinary(t)
	defer os.Remove(binaryPath)

	// Start Vikunja container
	ctx := context.Background()
	var err error
	vikunjaContainer, vikunjaURL, err = startVikunja(ctx)
	if err != nil {
		t.Fatalf("failed to start Vikunja container: %v", err)
	}
	defer testcontainers.CleanupContainer(t, vikunjaContainer)

	// Create a user and get API token
	vikunjaToken, err = setupUser(vikunjaURL)
	if err != nil {
		t.Fatalf("failed to setup user: %v", err)
	}

	suite := godog.TestSuite{
		ScenarioInitializer: initializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned from godog")
	}
}

func buildBinary(t *testing.T) string {
	t.Helper()
	tmpBin := "/tmp/vikunja-cli-test"
	cmd := exec.Command("go", "build", "-o", tmpBin, "github.com/jrogala/vikunja-cli")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, out)
	}
	return tmpBin
}

func startVikunja(ctx context.Context) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        "vikunja/vikunja:latest",
		ExposedPorts: []string{"3456/tcp"},
		Env: map[string]string{
			"VIKUNJA_DATABASE_TYPE":              "sqlite",
			"VIKUNJA_DATABASE_PATH":              "/tmp/vikunja.db",
			"VIKUNJA_SERVICE_JWTSECRET":          "testsecret1234567890",
			"VIKUNJA_SERVICE_PUBLICURL":          "http://localhost:3456/",
			"VIKUNJA_SERVICE_ENABLEREGISTRATION": "true",
			"VIKUNJA_FILES_BASEPATH":             "/tmp/files",
		},
		Tmpfs: map[string]string{
			"/tmp":    "rw",
			"/.cache": "rw",
		},
		WaitingFor: wait.ForHTTP("/api/v1/info").
			WithPort("3456/tcp").
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("start container: %w", err)
	}

	endpoint, err := container.Endpoint(ctx, "http")
	if err != nil {
		return nil, "", fmt.Errorf("get endpoint: %w", err)
	}

	return container, endpoint + "/api/v1", nil
}

func setupUser(baseURL string) (string, error) {
	// Register a user
	regBody := `{"username":"testuser","password":"testpassword123","email":"test@test.com"}`
	resp, err := http.Post(baseURL+"/register", "application/json", strings.NewReader(regBody))
	if err != nil {
		return "", fmt.Errorf("register: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("register failed %d: %s", resp.StatusCode, body)
	}

	// Login to get JWT
	loginBody := `{"username":"testuser","password":"testpassword123"}`
	resp, err = http.Post(baseURL+"/login", "application/json", strings.NewReader(loginBody))
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}
	defer resp.Body.Close()

	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", fmt.Errorf("decode login: %w", err)
	}

	// Create an API token
	tokenBody := `{"title":"test-token","permissions":{"tasks":["read","create","update","delete"],"projects":["read","create","update","delete"]}}`
	req, _ := http.NewRequest("PUT", baseURL+"/tokens", strings.NewReader(tokenBody))
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		// Fallback to JWT if API token creation fails
		return loginResp.Token, nil
	}
	if tokenResp.Token == "" {
		return loginResp.Token, nil
	}
	return tokenResp.Token, nil
}

func initializeScenario(sc *godog.ScenarioContext) {
	s := &testState{
		taskIDs:    make(map[string]int64),
		projectIDs: make(map[string]int64),
		binaryPath: binaryPath,
	}

	// Reset state before each scenario
	sc.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		s.baseURL = vikunjaURL
		s.token = vikunjaToken
		s.output = ""
		s.taskIDs = make(map[string]int64)
		s.projectIDs = make(map[string]int64)
		s.configDir = ""
		// Clean up tasks from previous scenarios
		s.cleanupTasks()
		return ctx, nil
	})

	// Clean up temp config dirs after each scenario
	sc.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		// Guard: only remove dirs under /tmp to never accidentally nuke user config
		if s.configDir != "" && strings.HasPrefix(s.configDir, os.TempDir()) {
			os.RemoveAll(s.configDir)
		}
		return ctx, nil
	})

	// Background
	sc.Step(`^a running Vikunja instance$`, s.aRunningVikunjaInstance)
	sc.Step(`^an authenticated user$`, s.anAuthenticatedUser)

	// Task steps
	sc.Step(`^I list tasks$`, s.iListTasks)
	sc.Step(`^I list all tasks$`, s.iListAllTasks)
	sc.Step(`^I list tasks in project "([^"]*)"$`, s.iListTasksInProject)
	sc.Step(`^I list tasks with JSON output$`, s.iListTasksJSON)
	sc.Step(`^I create a task "([^"]*)" in project (\d+)$`, s.iCreateTaskInProject)
	sc.Step(`^I create a task "([^"]*)" in project (\d+) with priority (\d+)$`, s.iCreateTaskWithPriority)
	sc.Step(`^I create a task "([^"]*)" in project (\d+) with due date "([^"]*)"$`, s.iCreateTaskWithDue)
	sc.Step(`^a task "([^"]*)" exists in project (\d+)$`, s.aTaskExistsInProjectID)
	sc.Step(`^a task "([^"]*)" exists in project "([^"]*)"$`, s.aTaskExistsInProjectName)
	sc.Step(`^task "([^"]*)" is completed$`, s.taskIsCompleted)
	sc.Step(`^I complete the task "([^"]*)"$`, s.iCompleteTask)
	sc.Step(`^I edit task "([^"]*)" with --title "([^"]*)"$`, s.iEditTaskTitle)
	sc.Step(`^I edit task "([^"]*)" with --priority (\d+)$`, s.iEditTaskPriority)
	sc.Step(`^I edit task "([^"]*)" with --description "([^"]*)"$`, s.iEditTaskDescription)
	sc.Step(`^I edit task "([^"]*)" with --undone$`, s.iEditTaskUndone)
	sc.Step(`^I edit task "([^"]*)" with --description ""$`, s.iEditTaskClearDescription)
	sc.Step(`^a task "([^"]*)" exists in project (\d+) with description "([^"]*)"$`, s.aTaskExistsWithDescription)
	sc.Step(`^task "([^"]*)" should have description "([^"]*)"$`, s.taskShouldHaveDescription)
	sc.Step(`^I delete the task "([^"]*)"$`, s.iDeleteTask)
	sc.Step(`^task "([^"]*)" should not be done$`, s.taskShouldNotBeDone)

	// Project steps
	sc.Step(`^a project "([^"]*)" exists$`, s.aProjectExists)
	sc.Step(`^I list projects$`, s.iListProjects)
	sc.Step(`^I list projects with JSON output$`, s.iListProjectsJSON)

	// Setup steps
	sc.Step(`^I run setup with valid URL and token$`, s.iRunSetupValid)
	sc.Step(`^I run setup with invalid URL$`, s.iRunSetupInvalidURL)
	sc.Step(`^I run setup with valid URL and invalid token$`, s.iRunSetupInvalidToken)
	sc.Step(`^the config file should exist$`, s.configFileShouldExist)
	sc.Step(`^the config file should contain the URL$`, s.configShouldContainURL)
	sc.Step(`^the config file should contain the token$`, s.configShouldContainToken)
	sc.Step(`^I list tasks without env vars$`, s.iListTasksWithoutEnvVars)
	sc.Step(`^the output should not contain "([^"]*)"$`, s.iShouldNotSee)

	// Assertion steps
	sc.Step(`^I should see "([^"]*)"$`, s.iShouldSee)
	sc.Step(`^I should not see "([^"]*)"$`, s.iShouldNotSee)
	sc.Step(`^the output should contain "([^"]*)"$`, s.outputShouldContain)
	sc.Step(`^task "([^"]*)" should exist$`, s.taskShouldExist)
	sc.Step(`^task "([^"]*)" should not exist$`, s.taskShouldNotExist)
	sc.Step(`^task "([^"]*)" should be done$`, s.taskShouldBeDone)
	sc.Step(`^task "([^"]*)" should have priority (\d+)$`, s.taskShouldHavePriority)
	sc.Step(`^task "([^"]*)" should have a due date$`, s.taskShouldHaveDueDate)
	sc.Step(`^the output should be valid JSON$`, s.outputShouldBeJSON)
	sc.Step(`^the JSON should contain a task titled "([^"]*)"$`, s.jsonShouldContainTask)
}

// CLI runner
func (s *testState) runCLI(args ...string) error {
	cmd := exec.Command(s.binaryPath, args...)
	cmd.Env = append(os.Environ(),
		"VIKUNJA_URL="+s.baseURL,
		"VIKUNJA_TOKEN="+s.token,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	s.output = out.String()
	return err
}

// API helper
func (s *testState) apiRequest(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, s.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (s *testState) cleanupTasks() {
	data, err := s.apiRequest("GET", "/tasks", nil)
	if err != nil {
		return
	}
	var tasks []struct {
		ID int64 `json:"id"`
	}
	if json.Unmarshal(data, &tasks) != nil {
		return
	}
	for _, t := range tasks {
		s.apiRequest("DELETE", fmt.Sprintf("/tasks/%d", t.ID), nil)
	}
}

// Background steps

func (s *testState) aRunningVikunjaInstance() error {
	if s.baseURL == "" {
		return fmt.Errorf("Vikunja not running")
	}
	return nil
}

func (s *testState) anAuthenticatedUser() error {
	if s.token == "" {
		return fmt.Errorf("no auth token")
	}
	return nil
}

// Task action steps

func (s *testState) iListTasks() error {
	return s.runCLI("task", "list")
}

func (s *testState) iListAllTasks() error {
	return s.runCLI("task", "list", "--all")
}

func (s *testState) iListTasksInProject(name string) error {
	projectID, ok := s.projectIDs[name]
	if !ok {
		return fmt.Errorf("unknown project %q", name)
	}
	return s.runCLI("task", "list", "-p", fmt.Sprintf("%d", projectID))
}

func (s *testState) iListTasksJSON() error {
	return s.runCLI("--json", "task", "list")
}

func (s *testState) iCreateTaskInProject(title string, projectID int) error {
	return s.runCLI("task", "add", "-p", fmt.Sprintf("%d", projectID), title)
}

func (s *testState) iCreateTaskWithPriority(title string, projectID, priority int) error {
	return s.runCLI("task", "add", "-p", fmt.Sprintf("%d", projectID), "--priority", fmt.Sprintf("%d", priority), title)
}

func (s *testState) iCreateTaskWithDue(title string, projectID int, due string) error {
	return s.runCLI("task", "add", "-p", fmt.Sprintf("%d", projectID), "--due-date", due, title)
}

func (s *testState) aTaskExistsInProjectID(title string, projectID int) error {
	data, err := s.apiRequest("PUT", fmt.Sprintf("/projects/%d/tasks", projectID), map[string]any{"title": title})
	if err != nil {
		return err
	}
	var task struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(data, &task); err != nil {
		return fmt.Errorf("create task %q: %w", title, err)
	}
	s.taskIDs[title] = task.ID
	return nil
}

func (s *testState) aTaskExistsInProjectName(title, projectName string) error {
	projectID, ok := s.projectIDs[projectName]
	if !ok {
		return fmt.Errorf("unknown project %q", projectName)
	}
	data, err := s.apiRequest("PUT", fmt.Sprintf("/projects/%d/tasks", projectID), map[string]any{"title": title})
	if err != nil {
		return err
	}
	var task struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(data, &task); err != nil {
		return fmt.Errorf("create task %q: %w", title, err)
	}
	s.taskIDs[title] = task.ID
	return nil
}

func (s *testState) taskIsCompleted(title string) error {
	id, ok := s.taskIDs[title]
	if !ok {
		return fmt.Errorf("unknown task %q", title)
	}
	_, err := s.apiRequest("POST", fmt.Sprintf("/tasks/%d", id), map[string]any{"done": true})
	return err
}

func (s *testState) iCompleteTask(title string) error {
	id, ok := s.taskIDs[title]
	if !ok {
		return fmt.Errorf("unknown task %q", title)
	}
	return s.runCLI("task", "done", fmt.Sprintf("%d", id))
}

func (s *testState) iDeleteTask(title string) error {
	id, ok := s.taskIDs[title]
	if !ok {
		return fmt.Errorf("unknown task %q", title)
	}
	return s.runCLI("task", "delete", fmt.Sprintf("%d", id))
}

// Project action steps

func (s *testState) aProjectExists(name string) error {
	data, err := s.apiRequest("PUT", "/projects", map[string]any{"title": name})
	if err != nil {
		return err
	}
	var project struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(data, &project); err != nil {
		return fmt.Errorf("create project %q: %w", name, err)
	}
	s.projectIDs[name] = project.ID
	return nil
}

func (s *testState) iListProjects() error {
	return s.runCLI("project", "list")
}

func (s *testState) iListProjectsJSON() error {
	return s.runCLI("--json", "project", "list")
}

// Assertion steps

func (s *testState) iShouldSee(text string) error {
	if !strings.Contains(s.output, text) {
		return fmt.Errorf("expected %q in output:\n%s", text, s.output)
	}
	return nil
}

func (s *testState) iShouldNotSee(text string) error {
	if strings.Contains(s.output, text) {
		return fmt.Errorf("did not expect %q in output:\n%s", text, s.output)
	}
	return nil
}

func (s *testState) outputShouldContain(text string) error {
	return s.iShouldSee(text)
}

func (s *testState) taskShouldExist(title string) error {
	data, err := s.apiRequest("GET", "/tasks", nil)
	if err != nil {
		return err
	}
	var tasks []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(data, &tasks); err != nil {
		return err
	}
	for _, t := range tasks {
		if t.Title == title {
			return nil
		}
	}
	return fmt.Errorf("task %q not found", title)
}

func (s *testState) taskShouldNotExist(title string) error {
	data, err := s.apiRequest("GET", "/tasks", nil)
	if err != nil {
		return err
	}
	var tasks []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(data, &tasks); err != nil {
		return err
	}
	for _, t := range tasks {
		if t.Title == title {
			return fmt.Errorf("task %q still exists", title)
		}
	}
	return nil
}

func (s *testState) taskShouldBeDone(title string) error {
	id, ok := s.taskIDs[title]
	if !ok {
		return fmt.Errorf("unknown task %q", title)
	}
	data, err := s.apiRequest("GET", fmt.Sprintf("/tasks/%d", id), nil)
	if err != nil {
		return err
	}
	var task struct {
		Done bool `json:"done"`
	}
	if err := json.Unmarshal(data, &task); err != nil {
		return err
	}
	if !task.Done {
		return fmt.Errorf("task %q is not done", title)
	}
	return nil
}

func (s *testState) taskShouldHavePriority(title string, expected int) error {
	data, err := s.apiRequest("GET", "/tasks", nil)
	if err != nil {
		return err
	}
	var tasks []struct {
		Title    string `json:"title"`
		Priority int    `json:"priority"`
	}
	if err := json.Unmarshal(data, &tasks); err != nil {
		return err
	}
	for _, t := range tasks {
		if t.Title == title {
			if t.Priority != expected {
				return fmt.Errorf("task %q has priority %d, want %d", title, t.Priority, expected)
			}
			return nil
		}
	}
	return fmt.Errorf("task %q not found", title)
}

func (s *testState) taskShouldHaveDueDate(title string) error {
	data, err := s.apiRequest("GET", "/tasks", nil)
	if err != nil {
		return err
	}
	var tasks []struct {
		Title   string `json:"title"`
		DueDate string `json:"due_date"`
	}
	if err := json.Unmarshal(data, &tasks); err != nil {
		return err
	}
	for _, t := range tasks {
		if t.Title == title {
			if t.DueDate == "" || strings.HasPrefix(t.DueDate, "0001") {
				return fmt.Errorf("task %q has no due date", title)
			}
			return nil
		}
	}
	return fmt.Errorf("task %q not found", title)
}

func (s *testState) outputShouldBeJSON() error {
	var v interface{}
	if err := json.Unmarshal([]byte(s.output), &v); err != nil {
		return fmt.Errorf("output is not valid JSON: %w\noutput:\n%s", err, s.output)
	}
	return nil
}

func (s *testState) jsonShouldContainTask(title string) error {
	var tasks []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(s.output), &tasks); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}
	for _, t := range tasks {
		if t.Title == title {
			return nil
		}
	}
	return fmt.Errorf("task %q not found in JSON output", title)
}

// Edit steps

func (s *testState) iEditTaskTitle(title, newTitle string) error {
	id, ok := s.taskIDs[title]
	if !ok {
		return fmt.Errorf("unknown task %q", title)
	}
	err := s.runCLI("task", "edit", fmt.Sprintf("%d", id), "--title", newTitle)
	if err != nil {
		return err
	}
	s.taskIDs[newTitle] = id
	delete(s.taskIDs, title)
	return nil
}

func (s *testState) iEditTaskPriority(title string, priority int) error {
	id, ok := s.taskIDs[title]
	if !ok {
		return fmt.Errorf("unknown task %q", title)
	}
	return s.runCLI("task", "edit", fmt.Sprintf("%d", id), "--priority", fmt.Sprintf("%d", priority))
}

func (s *testState) iEditTaskDescription(title, desc string) error {
	id, ok := s.taskIDs[title]
	if !ok {
		return fmt.Errorf("unknown task %q", title)
	}
	return s.runCLI("task", "edit", fmt.Sprintf("%d", id), "--description", desc)
}

func (s *testState) iEditTaskClearDescription(title string) error {
	id, ok := s.taskIDs[title]
	if !ok {
		return fmt.Errorf("unknown task %q", title)
	}
	return s.runCLI("task", "edit", fmt.Sprintf("%d", id), "--description", "")
}

func (s *testState) aTaskExistsWithDescription(title string, projectID int, desc string) error {
	data, err := s.apiRequest("PUT", fmt.Sprintf("/projects/%d/tasks", projectID), map[string]any{
		"title":       title,
		"description": desc,
	})
	if err != nil {
		return err
	}
	var task struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(data, &task); err != nil {
		return fmt.Errorf("create task %q: %w", title, err)
	}
	s.taskIDs[title] = task.ID
	return nil
}

func (s *testState) taskShouldHaveDescription(title, expected string) error {
	id, ok := s.taskIDs[title]
	if !ok {
		return fmt.Errorf("unknown task %q", title)
	}
	data, err := s.apiRequest("GET", fmt.Sprintf("/tasks/%d", id), nil)
	if err != nil {
		return err
	}
	var task struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal(data, &task); err != nil {
		return err
	}
	if task.Description != expected {
		return fmt.Errorf("task %q description = %q, want %q", title, task.Description, expected)
	}
	return nil
}

func (s *testState) iEditTaskUndone(title string) error {
	id, ok := s.taskIDs[title]
	if !ok {
		return fmt.Errorf("unknown task %q", title)
	}
	return s.runCLI("task", "edit", fmt.Sprintf("%d", id), "--undone")
}

func (s *testState) taskShouldNotBeDone(title string) error {
	id, ok := s.taskIDs[title]
	if !ok {
		return fmt.Errorf("unknown task %q", title)
	}
	data, err := s.apiRequest("GET", fmt.Sprintf("/tasks/%d", id), nil)
	if err != nil {
		return err
	}
	var task struct {
		Done bool `json:"done"`
	}
	if err := json.Unmarshal(data, &task); err != nil {
		return err
	}
	if task.Done {
		return fmt.Errorf("task %q is still done", title)
	}
	return nil
}

// Setup steps

func (s *testState) runCLIWithStdin(stdin string, envOverride []string, args ...string) error {
	cmd := exec.Command(s.binaryPath, args...)
	if envOverride != nil {
		cmd.Env = envOverride
	} else {
		cmd.Env = append(os.Environ(),
			"VIKUNJA_URL="+s.baseURL,
			"VIKUNJA_TOKEN="+s.token,
		)
	}
	cmd.Stdin = strings.NewReader(stdin)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	_ = cmd.Run() // don't fail on non-zero exit — we check output
	s.output = out.String()
	return nil
}

func (s *testState) setupConfigDir() error {
	dir, err := os.MkdirTemp("", "vikunja-cli-test-*")
	if err != nil {
		return err
	}
	s.configDir = dir
	return nil
}

func (s *testState) configEnv() []string {
	// Minimal env: PATH, HOME override for XDG, no VIKUNJA_* vars
	path := os.Getenv("PATH")
	return []string{
		"PATH=" + path,
		"HOME=" + s.configDir,
		"XDG_CONFIG_HOME=" + s.configDir,
	}
}

func (s *testState) iRunSetupValid() error {
	if err := s.setupConfigDir(); err != nil {
		return err
	}
	stdin := s.baseURL + "\n" + s.token + "\n"
	return s.runCLIWithStdin(stdin, s.configEnv(), "setup")
}

func (s *testState) iRunSetupInvalidURL() error {
	if err := s.setupConfigDir(); err != nil {
		return err
	}
	stdin := "http://localhost:1/api/v1\nfaketoken\n"
	return s.runCLIWithStdin(stdin, s.configEnv(), "setup")
}

func (s *testState) iRunSetupInvalidToken() error {
	if err := s.setupConfigDir(); err != nil {
		return err
	}
	stdin := s.baseURL + "\nbadtoken\n"
	return s.runCLIWithStdin(stdin, s.configEnv(), "setup")
}

func (s *testState) configFileShouldExist() error {
	path := filepath.Join(s.configDir, "vikunja-cli", "config.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("config file not found at %s", path)
	}
	return nil
}

func (s *testState) configShouldContainURL() error {
	path := filepath.Join(s.configDir, "vikunja-cli", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !strings.Contains(string(data), s.baseURL) {
		return fmt.Errorf("config does not contain URL %s:\n%s", s.baseURL, data)
	}
	return nil
}

func (s *testState) configShouldContainToken() error {
	path := filepath.Join(s.configDir, "vikunja-cli", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !strings.Contains(string(data), s.token) {
		return fmt.Errorf("config does not contain token:\n%s", data)
	}
	return nil
}

func (s *testState) iListTasksWithoutEnvVars() error {
	// Run with only the config dir set, no VIKUNJA_* env vars
	return s.runCLIWithStdin("", s.configEnv(), "task", "list")
}
