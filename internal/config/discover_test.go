package config

import (
	"testing"
	"testing/fstest"
)

func TestFindProjectConfig(t *testing.T) {
	tests := []struct {
		name    string
		fs      fstest.MapFS
		start   string
		home    string
		want    string
		wantOK  bool
	}{
		{
			name: "found in cwd, yml wins",
			fs: fstest.MapFS{
				"home/user/proj/.jira-config.yml": {},
			},
			start:  "home/user/proj",
			home:   "home/user",
			want:   "home/user/proj/.jira-config.yml",
			wantOK: true,
		},
		{
			name: "found in parent",
			fs: fstest.MapFS{
				"home/user/proj/.jira-config.yml": {},
				"home/user/proj/sub/keep":         {},
			},
			start:  "home/user/proj/sub",
			home:   "home/user",
			want:   "home/user/proj/.jira-config.yml",
			wantOK: true,
		},
		{
			name: "yml beats yaml beats json in same dir",
			fs: fstest.MapFS{
				"a/.jira-config.yaml": {},
				"a/.jira-config.json": {},
				"a/.jira-config.yml":  {},
			},
			start:  "a",
			want:   "a/.jira-config.yml",
			wantOK: true,
		},
		{
			name: "yaml found when yml absent",
			fs: fstest.MapFS{
				"a/.jira-config.yaml": {},
				"a/.jira-config.json": {},
			},
			start:  "a",
			want:   "a/.jira-config.yaml",
			wantOK: true,
		},
		{
			name: "json found when yml/yaml absent",
			fs: fstest.MapFS{
				"a/.jira-config.json": {},
			},
			start:  "a",
			want:   "a/.jira-config.json",
			wantOK: true,
		},
		{
			name: "stops at home boundary",
			fs: fstest.MapFS{
				"home/.jira-config.yml":          {},
				"home/user/proj/sub/placeholder": {},
			},
			start:  "home/user/proj/sub",
			home:   "home/user",
			wantOK: false,
		},
		{
			name: "home dir itself is searched",
			fs: fstest.MapFS{
				"home/user/.jira-config.yml": {},
			},
			start:  "home/user/proj",
			home:   "home/user",
			want:   "home/user/.jira-config.yml",
			wantOK: true,
		},
		{
			name: "stops at .git boundary",
			fs: fstest.MapFS{
				"repo/.git/HEAD":                  {},
				"repo/sub/placeholder":            {},
				"outside-repo/.jira-config.yml":   {},
			},
			start:  "repo/sub",
			home:   "",
			wantOK: false,
		},
		{
			name: ".git in same dir as config — config still wins",
			fs: fstest.MapFS{
				"repo/.git/HEAD":          {},
				"repo/.jira-config.yml":   {},
			},
			start:  "repo",
			want:   "repo/.jira-config.yml",
			wantOK: true,
		},
		{
			name: "no file anywhere",
			fs: fstest.MapFS{
				"a/b/c/keep": {},
			},
			start:  "a/b/c",
			home:   "",
			wantOK: false,
		},
		{
			name: "no home: walks up to filesystem root",
			fs: fstest.MapFS{
				".jira-config.yml":     {},
				"a/b/c/d/e/placeholder": {},
			},
			start:  "a/b/c/d/e",
			home:   "",
			want:   ".jira-config.yml",
			wantOK: true,
		},
		{
			name: "start at root",
			fs: fstest.MapFS{
				".jira-config.yml": {},
			},
			start:  ".",
			home:   "",
			want:   ".jira-config.yml",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := FindProjectConfig(tt.fs, tt.start, tt.home)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v (path=%q)", ok, tt.wantOK, got)
			}
			if got != tt.want {
				t.Fatalf("path = %q, want %q", got, tt.want)
			}
		})
	}
}
