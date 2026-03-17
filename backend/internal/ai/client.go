package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/machines-got-talent/backend/pkg/config"
)

// Comedian is the Go representation of one AI comedian agent.
type Comedian struct {
	ID          string `json:"id"`
	Personality string `json:"personality,omitempty"`
}

// Client talks to the Python AI agent service.
type Client struct {
	base string
	http *http.Client
}

func NewClient() *Client {
	return &Client{
		base: config.AIServiceBaseURL,
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateAgents asks the Python service to spin up n comedian agents and
// returns their IDs as assigned by the service.
func (c *Client) CreateAgents(n int) ([]Comedian, error) {
	body, _ := json.Marshal(map[string]int{"n": n})
	resp, err := c.http.Post(c.base+"/create_agents", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ai service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ai service error %d: %s", resp.StatusCode, raw)
	}

	var result struct {
		Agents []string `json:"agents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ai service decode error: %w", err)
	}

	comedians := make([]Comedian, len(result.Agents))
	for i, id := range result.Agents {
		comedians[i] = Comedian{ID: id}
	}
	return comedians, nil
}

// Chat sends a message to a specific comedian and returns their reply.
func (c *Client) Chat(agentID, message string) (string, string, error) {
	body, _ := json.Marshal(map[string]string{
		"agent_id": agentID,
		"message":  message,
	})
	resp, err := c.http.Post(c.base+"/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("ai service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("ai service error %d: %s", resp.StatusCode, raw)
	}

	var result struct {
		Personality string `json:"personality"`
		Response    string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("ai service decode error: %w", err)
	}
	return result.Personality, result.Response, nil
}
