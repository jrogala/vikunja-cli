package cmd

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(setupCmd)
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure Vikunja URL and API token. Writes to ~/.config/vikunja-cli/config.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		// URL
		fmt.Print("Vikunja URL (e.g. https://vikunja.example.com/api/v1): ")
		urlInput, _ := reader.ReadString('\n')
		urlInput = strings.TrimSpace(urlInput)
		if urlInput == "" {
			return fmt.Errorf("URL is required")
		}

		// Validate URL
		if err := pingVikunja(urlInput); err != nil {
			return fmt.Errorf("cannot reach Vikunja at %s: %w", urlInput, err)
		}
		fmt.Println("  Connected.")

		// Token
		fmt.Print("API token (tk_...): ")
		tokenInput, _ := reader.ReadString('\n')
		tokenInput = strings.TrimSpace(tokenInput)
		if tokenInput == "" {
			return fmt.Errorf("token is required")
		}

		// Validate token
		if err := validateToken(urlInput, tokenInput); err != nil {
			return fmt.Errorf("token validation failed: %w", err)
		}
		fmt.Println("  Authenticated.")

		// Write config
		configDir, err := os.UserConfigDir()
		if err != nil {
			return fmt.Errorf("cannot determine config dir: %w", err)
		}
		dir := filepath.Join(configDir, "vikunja-cli")
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("cannot create config dir: %w", err)
		}
		configPath := filepath.Join(dir, "config.yaml")

		content := fmt.Sprintf("url: %s\ntoken: %s\n", urlInput, tokenInput)
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			return fmt.Errorf("cannot write config: %w", err)
		}

		fmt.Printf("Config saved to %s\n", configPath)
		return nil
	},
}

func pingVikunja(baseURL string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	url := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(url, "/api/v1") {
		url = url[:len(url)-7]
	}
	resp, err := client.Get(url + "/api/v1/info")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func validateToken(baseURL, token string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	url := strings.TrimRight(baseURL, "/") + "/projects"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid token")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}
