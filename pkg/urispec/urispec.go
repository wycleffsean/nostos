package urispec

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v5"
)

// Spec represents a URI specification. It can reference a local path or a
// git repository.
type Spec struct {
	Raw  string // original specification
	Type string // "path" or "git"
	Path string // local path or remote URL
}

// Parse converts a string specification into a Spec structure. It supports
// plain filesystem paths as well as git repositories via HTTPS or the
// "github:" shorthand.
func Parse(s string) Spec {
	if strings.HasPrefix(s, "github:") {
		repo := strings.TrimPrefix(s, "github:")
		return Spec{Raw: s, Type: "git", Path: fmt.Sprintf("https://github.com/%s.git", repo)}
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		if strings.HasSuffix(s, ".git") {
			return Spec{Raw: s, Type: "git", Path: s}
		}
	}
	return Spec{Raw: s, Type: "path", Path: s}
}

// LocalPath resolves the spec to a local filesystem path. For git
// repositories it clones the repo into the user's cache directory on the first
// use.
func (s Spec) LocalPath() (string, error) {
	switch s.Type {
	case "path":
		if s.Path == "" {
			return "", fmt.Errorf("empty path")
		}
		return filepath.Abs(s.Path)
	case "git":
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			cacheDir = os.TempDir()
		}
		dir := filepath.Join(cacheDir, "nostos")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		hash := fmt.Sprintf("%x", sha1.Sum([]byte(s.Path)))
		repoDir := filepath.Join(dir, hash)
		if _, err := os.Stat(repoDir); os.IsNotExist(err) {
			if _, err := git.PlainClone(repoDir, false, &git.CloneOptions{URL: s.Path}); err != nil {
				return "", err
			}
		}
		return repoDir, nil
	default:
		return "", fmt.Errorf("unknown spec type %s", s.Type)
	}
}
