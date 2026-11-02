package vault

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/log"
)

var ErrTokenInvalid = errors.New("vault token invalid")

type Client interface {
	Login() error
	LookupToken() error
	ListPath(path string) ([]string, error)
	GetSecret(path string) (map[string]any, error)
}

type CLIClient struct {
	namespace string
	token     string
}

func NewCLIClient(namespace string) *CLIClient {
	return &CLIClient{namespace: namespace}
}

// command builds a vault invocation; flags must follow the subcommand.
func (c *CLIClient) command(subcmd []string, flags ...string) *exec.Cmd {
	args := append([]string{}, subcmd...)
	if c.namespace != "" {
		args = append(args, "-namespace="+c.namespace)
	}
	args = append(args, flags...)
	cmd := exec.Command("vault", args...) //nolint:gosec
	if c.token != "" {
		// Override any VAULT_TOKEN from the environment, which beats the token helper.
		cmd.Env = append(os.Environ(), "VAULT_TOKEN="+c.token)
	}
	return cmd
}

func (c *CLIClient) Login() error {
	var stdout bytes.Buffer
	cmd := c.command([]string{"login"}, "-method", "oidc", "-token-only")
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vault OIDC login failed: %w", err)
	}

	c.token = strings.TrimSpace(stdout.String())
	if c.token == "" {
		return fmt.Errorf("vault OIDC login returned no token")
	}

	return nil
}

func (c *CLIClient) LookupToken() error {
	log.Debug("Looking up token", "namespace", c.namespace)

	var stderr bytes.Buffer
	cmd := c.command([]string{"token", "lookup"}, "-format=json")
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		if strings.Contains(stderr.String(), "no such host") {
			return fmt.Errorf("can't reach vault — are you on vpn?")
		}
		return fmt.Errorf("%w for namespace %q", ErrTokenInvalid, c.namespace)
	}

	if c.namespace == "" {
		return nil
	}

	var response struct {
		Data struct {
			NamespacePath string `json:"namespace_path"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return fmt.Errorf("parsing token lookup output: %w", err)
	}

	// A parent-namespace token passes lookup but may have no access in the target namespace.
	tokenNS := strings.Trim(response.Data.NamespacePath, "/")
	if tokenNS != strings.Trim(c.namespace, "/") {
		log.Debug("Token namespace mismatch", "token_namespace", tokenNS, "target", c.namespace)
		return fmt.Errorf("%w for namespace %q", ErrTokenInvalid, c.namespace)
	}

	return nil
}

func (c *CLIClient) ListPath(path string) ([]string, error) {
	log.Debug("Listing", "path", path)

	cmd := c.command([]string{"kv", "list"}, "-format=json", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Debug("List failed", "path", path)
		return nil, nil
	}

	var items []string
	if err := json.Unmarshal(output, &items); err != nil {
		return nil, fmt.Errorf("parsing list output for %s: %w", path, err)
	}

	return items, nil
}

func (c *CLIClient) GetSecret(path string) (map[string]any, error) {
	log.Debug("Getting secret", "path", path)

	cmd := c.command([]string{"kv", "get"}, "-format=json", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		errOut := string(output)
		if strings.Contains(errOut, "permission denied") {
			return nil, fmt.Errorf("permission denied at %s — have you logged in?", path)
		}
		if strings.Contains(errOut, "no such host") {
			return nil, fmt.Errorf("can't reach vault — are you on vpn?")
		}
		log.Debug("No data", "path", path)
		return nil, nil
	}

	var response map[string]any
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("parsing secret at %s: %w", path, err)
	}

	return extractData(response), nil
}

func extractData(response map[string]any) map[string]any {
	data, ok := response["data"].(map[string]any)
	if !ok {
		return nil
	}

	// KV v2: nested .data.data
	if inner, ok := data["data"].(map[string]any); ok {
		return inner
	}

	return data
}
