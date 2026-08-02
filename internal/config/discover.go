package config

import (
	"io/fs"
	"path"
	"strings"
)

// ProjectConfigBase is the basename (without extension) looked up when
// walking up the directory tree to discover a project-local jira config.
const ProjectConfigBase = ".jira-config"

// FindProjectConfig walks up from startDir looking for ProjectConfigBase
// with one of the supported extensions (.yml, .yaml, .json — all formats
// viper can parse). Within a directory, extensions are tried in declaration
// order; the first hit wins. Search stops at homeDir, at a directory
// containing .git, or at the filesystem root.
func FindProjectConfig(fsys fs.FS, startDir, homeDir string) (string, bool) {
	candidates := []string{
		ProjectConfigBase + ".yml",
		ProjectConfigBase + ".yaml",
		ProjectConfigBase + ".json",
	}

	dir := path.Clean(strings.TrimPrefix(startDir, "/"))
	home := path.Clean(strings.TrimPrefix(homeDir, "/"))

	for {
		for _, name := range candidates {
			candidate := path.Join(dir, name)
			if _, err := fs.Stat(fsys, candidate); err == nil {
				return candidate, true
			}
		}

		if homeDir != "" && dir == home {
			return "", false
		}
		if _, err := fs.Stat(fsys, path.Join(dir, ".git")); err == nil {
			return "", false
		}

		parent := path.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
