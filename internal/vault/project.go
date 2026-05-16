// Package vault owns the on-disk vault data model. It is the only
// package that touches the filesystem for secret storage.
package vault

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Project identifies a unique vault location on disk.
type Project struct {
	Name string // human-readable, e.g. "myapp"
	ID   string // 8-char sha256 prefix used as the directory name
	Dir  string // absolute path to the per-project directory
}

// ResolveProject identifies the current project for vault lookup.
//
// It tries `git remote get-url origin` first so a repo's vault follows
// the remote identity (renaming the local folder doesn't break it). If
// the directory isn't a git repo or has no origin, it falls back to the
// folder name.
//
// vaultRoot is the parent directory of all project vaults (typically
// ~/.aden/). It is created on demand by callers that need to write.
func ResolveProject(vaultRoot string) (Project, error) {
	if vaultRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Project{}, fmt.Errorf("resolve home dir: %w", err)
		}
		vaultRoot = filepath.Join(home, ".aden")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return Project{}, fmt.Errorf("get working dir: %w", err)
	}

	name, identity := projectIdentity(cwd)
	sum := sha256.Sum256([]byte(identity))
	id := hex.EncodeToString(sum[:])[:8]

	return Project{
		Name: name,
		ID:   id,
		Dir:  filepath.Join(vaultRoot, fmt.Sprintf("%s-%s", name, id)),
	}, nil
}

// projectIdentity returns the human name and the identity string used
// to derive the project ID. The identity prefers `git remote origin`
// for stability across folder renames.
func projectIdentity(cwd string) (name, identity string) {
	if remote := gitOriginURL(cwd); remote != "" {
		return repoNameFromRemote(remote), remote
	}
	folder := filepath.Base(cwd)
	return sanitize(folder), folder
}

func gitOriginURL(cwd string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// repoNameFromRemote extracts a project name from a git remote URL.
//
// Examples:
//
//	git@github.com:user/myapp.git   -> myapp
//	https://github.com/user/myapp   -> myapp
//	ssh://git@host/team/svc.git     -> svc
func repoNameFromRemote(remote string) string {
	// Drop trailing slashes (some users paste URLs with them) and then
	// strip a trailing .git so both flavours normalise to the same name.
	r := strings.TrimRight(remote, "/")
	r = strings.TrimSuffix(r, ".git")
	if i := strings.LastIndexAny(r, "/:"); i >= 0 {
		r = r[i+1:]
	}
	return sanitize(r)
}

// sanitize lowercases and replaces filesystem-unfriendly characters so
// the project name is safe to use as a directory component.
func sanitize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "project"
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= '0' && r <= '9',
			r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}

// VaultPath returns the absolute path to vault.<env>.json for this
// project.
func (p Project) VaultPath(env string) string {
	return filepath.Join(p.Dir, fmt.Sprintf("vault.%s.json", env))
}

// ConfigPath returns the absolute path to config.json for this project.
func (p Project) ConfigPath() string {
	return filepath.Join(p.Dir, "config.json")
}
