package ngrok

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Tunnel represents an ngrok tunnel instance
type Tunnel struct {
	cmd       *exec.Cmd
	publicURL string
}

// Start starts an ngrok tunnel using the CLI
func Start(port, authToken string) (*Tunnel, error) {
	var cmd *exec.Cmd

	if authToken != "" {
		cmd = exec.Command("ngrok", "http", port, "--authtoken", authToken, "--log=stderr", "--log-level=crit")
	} else {
		cmd = exec.Command("ngrok", "http", port, "--log=stderr", "--log-level=warn")
	}

	// Only show errors, suppress verbose output
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ngrok: %w", err)
	}

	// Reap the process and detect early exit (auth failure, session
	// limit, port in use, etc.) instead of reporting a false success.
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()

	deadline := time.After(15 * time.Second)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case err := <-exited:
			return nil, fmt.Errorf("ngrok exited before establishing a tunnel: %v", err)
		case <-deadline:
			cmd.Process.Kill()
			return nil, fmt.Errorf("timed out waiting for ngrok tunnel forwarding to port %s", port)
		case <-tick.C:
			if publicURL, err := getPublicURL(port); err == nil {
				return &Tunnel{
					cmd:       cmd,
					publicURL: publicURL,
				}, nil
			}
		}
	}
}

// getPublicURL queries the local ngrok agent API for a tunnel forwarding to
// the given port. If port 4040 is already taken by another agent, ngrok picks
// the next free port for its API, so a few candidates are checked. Tunnels
// forwarding elsewhere are ignored — they belong to other agents.
func getPublicURL(port string) (string, error) {
	for _, apiPort := range []string{"4040", "4041", "4042", "4043"} {
		publicURL, err := queryAgentAPI(apiPort, port)
		if err == nil {
			return publicURL, nil
		}
	}
	return "", fmt.Errorf("no ngrok tunnel forwarding to port %s found", port)
}

// queryAgentAPI asks one ngrok agent API for a tunnel forwarding to port
func queryAgentAPI(apiPort, port string) (string, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:" + apiPort + "/api/tunnels")
	if err != nil {
		return "", fmt.Errorf("failed to query ngrok API: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Tunnels []struct {
			PublicURL string `json:"public_url"`
			Proto     string `json:"proto"`
			Config    struct {
				Addr string `json:"addr"`
			} `json:"config"`
		} `json:"tunnels"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse ngrok API response: %w", err)
	}

	// Prefer the HTTPS tunnel, but accept any proto forwarding to our port
	var fallback string
	for _, tunnel := range result.Tunnels {
		if !strings.HasSuffix(tunnel.Config.Addr, ":"+port) {
			continue
		}
		if tunnel.Proto == "https" {
			return tunnel.PublicURL, nil
		}
		fallback = tunnel.PublicURL
	}

	if fallback != "" {
		return fallback, nil
	}

	return "", fmt.Errorf("no tunnel forwarding to port %s on this agent", port)
}

// GetPublicURL returns the public URL of the tunnel
func (t *Tunnel) GetPublicURL() string {
	return t.publicURL
}

// Stop stops the ngrok tunnel
func (t *Tunnel) Stop() error {
	if t.cmd != nil && t.cmd.Process != nil {
		return t.cmd.Process.Kill()
	}
	return nil
}
