package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

const (
	oauth1File = "oauth1_token.json"
	oauth2File = "oauth2_token.json"
)

var profileNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

type Store struct {
	profile string
	dir     string
}

func NewStore(profile string) (*Store, error) {
	if profile == "" {
		profile = os.Getenv("GARMIN_PROFILE")
	}
	if profile == "" {
		profile = "default"
	}
	base, err := ConfigRoot()
	if err != nil {
		return nil, err
	}
	return NewStoreInDir(profile, base)
}

func NewStoreInDir(profile, base string) (*Store, error) {
	if profile == "" {
		profile = "default"
	}
	if !profileNamePattern.MatchString(profile) {
		return nil, fmt.Errorf("invalid profile %q: use letters, numbers, dot, dash, or underscore", profile)
	}
	dir := filepath.Join(base, profile)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Store{profile: profile, dir: dir}, nil
}

func ConfigRoot() (string, error) {
	if dir := os.Getenv("GARMIN_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "garmin"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "garmin"), nil
}

func (s *Store) Profile() string { return s.profile }
func (s *Store) Dir() string     { return s.dir }

func (s *Store) SaveTokens(oauth1 OAuth1Token, oauth2 OAuth2Token) error {
	if err := writeJSON0600(filepath.Join(s.dir, oauth1File), oauth1); err != nil {
		return err
	}
	return writeJSON0600(filepath.Join(s.dir, oauth2File), oauth2)
}

func (s *Store) LoadTokens() (OAuth1Token, OAuth2Token, bool, error) {
	var oauth1 OAuth1Token
	var oauth2 OAuth2Token
	if err := readJSON(filepath.Join(s.dir, oauth1File), &oauth1); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return oauth1, oauth2, false, nil
		}
		return oauth1, oauth2, false, err
	}
	if err := readJSON(filepath.Join(s.dir, oauth2File), &oauth2); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return oauth1, oauth2, false, nil
		}
		return oauth1, oauth2, false, err
	}
	return oauth1, oauth2, true, nil
}

func (s *Store) Clear() error {
	var joined error
	for _, name := range []string{oauth1File, oauth2File} {
		err := os.Remove(filepath.Join(s.dir, name))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			joined = errors.Join(joined, err)
		}
	}
	return joined
}

func writeJSON0600(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
