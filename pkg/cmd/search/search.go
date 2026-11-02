package search

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/log"

	"vault-secret-searcher/pkg/client/vault"
)

const (
	modeKeys   = "keys"
	modeValues = "values"
)

type Cmd struct {
	Keys      string `short:"k" xor:"mode" required:"" placeholder:"TERM" help:"Search secret key names (case-insensitive)"`
	Values    string `short:"v" xor:"mode" required:"" placeholder:"TERM" help:"Search secret values (case-insensitive)"`
	Path      string `short:"p" default:"secret/" help:"Vault KV path to search recursively"`
	Login     bool   `short:"l" help:"Run vault OIDC login before searching"`
	Namespace string `short:"n" env:"VAULT_NAMESPACE" help:"Vault namespace to use"`

	client vault.Client
	out    io.Writer
}

func (c *Cmd) mode() string {
	if c.Values != "" {
		return modeValues
	}
	return modeKeys
}

func (c *Cmd) Run(_ context.Context) error {
	if c.client == nil {
		c.client = vault.NewCLIClient(c.Namespace)
	}
	if c.out == nil {
		c.out = os.Stdout
	}

	if c.Login {
		if err := c.client.Login(); err != nil {
			return err
		}
	}

	if err := c.client.LookupToken(); err != nil {
		if c.Namespace == "" || c.Login || !errors.Is(err, vault.ErrTokenInvalid) {
			return err
		}
		log.Debug("Token invalid, requesting login", "namespace", c.Namespace)
		if err := c.client.Login(); err != nil {
			return err
		}
		if err := c.client.LookupToken(); err != nil {
			return err
		}
	}

	return c.walkPath(c.Path)
}

func (c *Cmd) walkPath(path string) error {
	items, err := c.client.ListPath(path)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		return c.searchSecret(path)
	}

	for _, item := range items {
		fullPath := path + item
		if strings.HasSuffix(item, "/") {
			if err := c.walkPath(fullPath); err != nil {
				return err
			}
		} else {
			if err := c.searchSecret(fullPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Cmd) searchSecret(path string) error {
	data, err := c.client.GetSecret(path)
	if err != nil {
		return err
	}

	if data == nil {
		log.Debug("No data", "path", path)
		return nil
	}

	if c.mode() == modeKeys {
		c.matchKeys(path, data)
	} else {
		c.matchValues(path, data)
	}

	return nil
}

func (c *Cmd) matchKeys(path string, data map[string]any) {
	term := strings.ToLower(c.Keys)
	var matches []string

	for key := range data {
		if strings.Contains(strings.ToLower(key), term) {
			matches = append(matches, key)
		}
	}

	sort.Strings(matches)

	if len(matches) > 0 {
		c.printFoundIn(path)
		c.printf("  Matching keys:\n")
		for _, m := range matches {
			c.printf("    %s\n", m)
		}
	}
}

func (c *Cmd) matchValues(path string, data map[string]any) {
	term := strings.ToLower(c.Values)
	var matches []string

	for key, val := range data {
		entry := fmt.Sprintf("%s=%v", key, val)
		if strings.Contains(strings.ToLower(entry), term) {
			matches = append(matches, entry)
		}
	}

	sort.Strings(matches)

	if len(matches) > 0 {
		c.printFoundIn(path)
		c.printf("  Matching entries:\n")
		for _, m := range matches {
			c.printf("    %s\n", m)
		}
	}
}

func (c *Cmd) printFoundIn(path string) {
	if c.Namespace != "" {
		c.printf("[%s] ", c.Namespace)
	}
	c.printf("Found in: %s\n", path)
}

func (c *Cmd) printf(format string, args ...any) {
	_, _ = fmt.Fprintf(c.out, format, args...)
}
