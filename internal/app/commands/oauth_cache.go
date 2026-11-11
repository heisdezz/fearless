package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"fearless/internal/core/engine/auth"
	"fearless/internal/core/events"
	"fearless/internal/core/persistence"
)

// OAuthTokenCache defines persistent or in-memory caching of acquired OAuth2 tokens.
type OAuthTokenCache interface {
	Get(credentialsID string) (auth.Token, bool)
	Set(credentialsID string, token auth.Token) error
	Delete(credentialsID string) error
	Clear() error
	Keys() []string
}

// MemoryOAuthTokenCache provides a concurrent in-memory token cache.
type MemoryOAuthTokenCache struct {
	mu     sync.RWMutex
	tokens map[string]auth.Token
}

// NewMemoryOAuthTokenCache creates a fresh in-memory token cache.
func NewMemoryOAuthTokenCache() *MemoryOAuthTokenCache {
	return &MemoryOAuthTokenCache{
		tokens: make(map[string]auth.Token),
	}
}

// Get returns the cached token for credentialsID, if present.
func (c *MemoryOAuthTokenCache) Get(credentialsID string) (auth.Token, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	tok, ok := c.tokens[credentialsID]
	return tok, ok
}

// Set stores a token for credentialsID.
func (c *MemoryOAuthTokenCache) Set(credentialsID string, token auth.Token) error {
	if credentialsID == "" {
		return errors.New("empty credentialsID")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tokens[credentialsID] = token
	return nil
}

// Delete removes credentialsID from the cache. Idempotent.
func (c *MemoryOAuthTokenCache) Delete(credentialsID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.tokens, credentialsID)
	return nil
}

// Clear purges all tokens from the cache. Idempotent.
func (c *MemoryOAuthTokenCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tokens = make(map[string]auth.Token)
	return nil
}

// Keys returns all active credentials IDs sorted lexicographically.
func (c *MemoryOAuthTokenCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]string, 0, len(c.tokens))
	for k := range c.tokens {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// FileOAuthTokenCache provides a disk-persisted, atomic-written token cache.
type FileOAuthTokenCache struct {
	mu     sync.RWMutex
	path   string
	tokens map[string]auth.Token
}

// NewFileOAuthTokenCache initializes a file-backed token cache at path.
// If the file exists, it loads stored tokens.
func NewFileOAuthTokenCache(path string) (*FileOAuthTokenCache, error) {
	if path == "" {
		return nil, errors.New("empty file token cache path")
	}

	c := &FileOAuthTokenCache{
		path:   path,
		tokens: make(map[string]auth.Token),
	}

	data, err := os.ReadFile(path)
	if err == nil {
		var stored map[string]auth.Token
		if jErr := json.Unmarshal(data, &stored); jErr == nil && stored != nil {
			c.tokens = stored
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read oauth token cache %s: %w", path, err)
	}

	return c, nil
}

// Get returns the cached token for credentialsID, if present.
func (c *FileOAuthTokenCache) Get(credentialsID string) (auth.Token, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	tok, ok := c.tokens[credentialsID]
	return tok, ok
}

// Set stores a token and atomically writes the updated cache to disk.
func (c *FileOAuthTokenCache) Set(credentialsID string, token auth.Token) error {
	if credentialsID == "" {
		return errors.New("empty credentialsID")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tokens[credentialsID] = token
	return c.saveLocked()
}

// Delete removes credentialsID and writes to disk. Idempotent.
func (c *FileOAuthTokenCache) Delete(credentialsID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.tokens[credentialsID]; !exists {
		return nil
	}
	delete(c.tokens, credentialsID)
	return c.saveLocked()
}

// Clear purges all tokens and writes to disk. Idempotent.
func (c *FileOAuthTokenCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.tokens) == 0 {
		return nil
	}
	c.tokens = make(map[string]auth.Token)
	return c.saveLocked()
}

// Keys returns all active credentials IDs sorted lexicographically.
func (c *FileOAuthTokenCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]string, 0, len(c.tokens))
	for k := range c.tokens {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (c *FileOAuthTokenCache) saveLocked() error {
	data, err := json.MarshalIndent(c.tokens, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal oauth tokens: %w", err)
	}
	return persistence.WriteAtomic(c.path, data, 0o600)
}

// ClearOAuth2Cache clears cached OAuth2 tokens from memory and disk.
// If cmd.CredentialsID is non-empty, only that specific token is cleared.
// If cmd.CredentialsID is empty, all cached tokens are cleared.
// Emits events.CredentialsUpdated for all affected credentials IDs.
// Calling ClearOAuth2Cache on empty or non-existent tokens is idempotent.
func (s *PreferencesService) ClearOAuth2Cache(ctx context.Context, cmd ClearOAuthCacheCmd) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if s.oauthCache == nil {
		return nil
	}

	if cmd.CredentialsID != "" {
		if err := s.oauthCache.Delete(cmd.CredentialsID); err != nil {
			return fmt.Errorf("clear oauth cache for %s: %w", cmd.CredentialsID, err)
		}
		if s.bus != nil {
			s.bus.Publish(events.CredentialsUpdated{
				CredentialsID: cmd.CredentialsID,
			})
		}
		return nil
	}

	// Clear all tokens
	keys := s.oauthCache.Keys()
	if err := s.oauthCache.Clear(); err != nil {
		return fmt.Errorf("clear all oauth tokens: %w", err)
	}

	if s.bus != nil {
		if len(keys) == 0 {
			s.bus.Publish(events.CredentialsUpdated{CredentialsID: ""})
		} else {
			for _, k := range keys {
				s.bus.Publish(events.CredentialsUpdated{CredentialsID: k})
			}
		}
	}

	return nil
}

// GetToken returns a token from the cache.
func (s *PreferencesService) GetToken(credentialsID string) (auth.Token, bool) {
	if s.oauthCache == nil {
		return auth.Token{}, false
	}
	return s.oauthCache.Get(credentialsID)
}

// SetToken stores a token in the cache and emits events.CredentialsUpdated.
func (s *PreferencesService) SetToken(credentialsID string, token auth.Token) error {
	if s.oauthCache == nil {
		return errors.New("oauth cache not configured")
	}
	if err := s.oauthCache.Set(credentialsID, token); err != nil {
		return err
	}
	if s.bus != nil {
		s.bus.Publish(events.CredentialsUpdated{
			CredentialsID: credentialsID,
		})
	}
	return nil
}

// DeleteToken removes a token from the cache and emits events.CredentialsUpdated.
func (s *PreferencesService) DeleteToken(credentialsID string) error {
	return s.ClearOAuth2Cache(context.Background(), ClearOAuthCacheCmd{
		CredentialsID: credentialsID,
	})
}

// DefaultOAuthCachePath determines the default path for disk-based token cache.
func DefaultOAuthCachePath() (string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		home := os.Getenv("HOME")
		if home == "" {
			return "", fmt.Errorf("unable to determine user config directory: %w", err)
		}
		cfgDir = filepath.Join(home, ".config")
	}
	return filepath.Join(cfgDir, "fearless", "oauth_tokens.json"), nil
}
