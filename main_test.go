package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseCLIArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		want    cliOptions
		wantErr string
	}{
		{
			name: "query only",
			args: []string{"go"},
			want: cliOptions{Query: "go"},
		},
		{
			name: "refresh before query",
			args: []string{"--refresh", "go"},
			want: cliOptions{Query: "go", Refresh: true},
		},
		{
			name: "refresh after query",
			args: []string{"go", "--refresh"},
			want: cliOptions{Query: "go", Refresh: true},
		},
		{
			name: "verbose short flag",
			args: []string{"-v", "go"},
			want: cliOptions{Query: "go", Verbose: true},
		},
		{
			name: "help long flag",
			args: []string{"--help"},
			want: cliOptions{ShowHelp: true},
		},
		{
			name: "version short flag",
			args: []string{"-V"},
			want: cliOptions{ShowVersion: true},
		},
		{
			name:    "unknown option",
			args:    []string{"--bad", "go"},
			wantErr: "unknown option",
		},
		{
			name:    "missing query",
			args:    nil,
			wantErr: "usage: zenn-topics [options] <query>",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseCLIArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("parseCLIArgs() error = nil, want %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want substring %q", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseCLIArgs() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("parseCLIArgs() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDiskTopicCacheLoadHit(t *testing.T) {
	t.Parallel()

	cachePath := filepath.Join(t.TempDir(), "topics.json")
	cache := diskTopicCache{path: cachePath}
	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)

	if err := cache.Save(now.Add(-30*time.Minute), []string{"go", "golang"}); err != nil {
		t.Fatalf("cache.Save() error = %v", err)
	}

	got, hit, err := cache.Load(now, time.Hour)
	if err != nil {
		t.Fatalf("cache.Load() error = %v", err)
	}
	if !hit {
		t.Fatal("cache.Load() hit = false, want true")
	}

	want := []string{"go", "golang"}
	assertStringSliceEqual(t, got, want)
}

func TestLoadTopicsFetchesAndWritesCacheOnMiss(t *testing.T) {
	t.Parallel()

	cachePath := filepath.Join(t.TempDir(), "cache", "topics.json")
	cache := diskTopicCache{path: cachePath}
	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)

	fetchCalls := 0
	deps := runDeps{
		fetch: func(context.Context) ([]string, error) {
			fetchCalls++
			return []string{"nextjs", "golang"}, nil
		},
		cache:    cache,
		now:      func() time.Time { return now },
		cacheTTL: time.Hour,
	}

	got, err := loadTopics(context.Background(), deps, false)
	if err != nil {
		t.Fatalf("loadTopics() error = %v", err)
	}
	if fetchCalls != 1 {
		t.Fatalf("fetchCalls = %d, want 1", fetchCalls)
	}
	assertStringSliceEqual(t, got, []string{"nextjs", "golang"})

	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("os.ReadFile(cachePath) error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("json.Unmarshal(cache file) error = %v", err)
	}
	if _, ok := payload["fetched_at"]; !ok {
		t.Fatalf("cache JSON missing fetched_at: %v", payload)
	}
	if _, ok := payload["slugs"]; !ok {
		t.Fatalf("cache JSON missing slugs: %v", payload)
	}
}

func TestLoadTopicsFetchesOnExpiredCache(t *testing.T) {
	t.Parallel()

	cachePath := filepath.Join(t.TempDir(), "topics.json")
	cache := diskTopicCache{path: cachePath}
	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)
	if err := cache.Save(now.Add(-2*time.Hour), []string{"old"}); err != nil {
		t.Fatalf("cache.Save() error = %v", err)
	}

	fetchCalls := 0
	deps := runDeps{
		fetch: func(context.Context) ([]string, error) {
			fetchCalls++
			return []string{"new"}, nil
		},
		cache:    cache,
		now:      func() time.Time { return now },
		cacheTTL: time.Hour,
	}

	got, err := loadTopics(context.Background(), deps, false)
	if err != nil {
		t.Fatalf("loadTopics() error = %v", err)
	}
	if fetchCalls != 1 {
		t.Fatalf("fetchCalls = %d, want 1", fetchCalls)
	}
	assertStringSliceEqual(t, got, []string{"new"})

	cached, hit, err := cache.Load(now, time.Hour)
	if err != nil {
		t.Fatalf("cache.Load() error = %v", err)
	}
	if !hit {
		t.Fatal("cache.Load() hit = false, want true after refresh")
	}
	assertStringSliceEqual(t, cached, []string{"new"})
}

func TestLoadTopicsRefreshBypassesFreshCache(t *testing.T) {
	t.Parallel()

	cachePath := filepath.Join(t.TempDir(), "topics.json")
	cache := diskTopicCache{path: cachePath}
	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)
	if err := cache.Save(now.Add(-5*time.Minute), []string{"cached"}); err != nil {
		t.Fatalf("cache.Save() error = %v", err)
	}

	fetchCalls := 0
	deps := runDeps{
		fetch: func(context.Context) ([]string, error) {
			fetchCalls++
			return []string{"fresh"}, nil
		},
		cache:    cache,
		now:      func() time.Time { return now },
		cacheTTL: time.Hour,
	}

	got, err := loadTopics(context.Background(), deps, true)
	if err != nil {
		t.Fatalf("loadTopics(refresh=true) error = %v", err)
	}
	if fetchCalls != 1 {
		t.Fatalf("fetchCalls = %d, want 1", fetchCalls)
	}
	assertStringSliceEqual(t, got, []string{"fresh"})

	cached, hit, err := cache.Load(now, time.Hour)
	if err != nil {
		t.Fatalf("cache.Load() error = %v", err)
	}
	if !hit {
		t.Fatal("cache.Load() hit = false, want true")
	}
	assertStringSliceEqual(t, cached, []string{"fresh"})
}

func TestParseTopicSitemapIndex(t *testing.T) {
	t.Parallel()

	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap><loc>https://zenn.dev/sitemaps/static.xml</loc></sitemap>
  <sitemap><loc>https://zenn.dev/sitemaps/article1.xml.gz</loc></sitemap>
  <sitemap><loc>https://zenn.dev/sitemaps/topic1.xml.gz</loc></sitemap>
  <sitemap><loc>https://zenn.dev/sitemaps/topic2.xml.gz</loc></sitemap>
</sitemapindex>`)

	got, err := parseTopicSitemapIndex(xmlData)
	if err != nil {
		t.Fatalf("parseTopicSitemapIndex() error = %v", err)
	}

	want := []string{
		"https://zenn.dev/sitemaps/topic1.xml.gz",
		"https://zenn.dev/sitemaps/topic2.xml.gz",
	}
	assertStringSliceEqual(t, got, want)
}

func TestParseTopicSlugsXML(t *testing.T) {
	t.Parallel()

	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://zenn.dev/topics/go</loc></url>
  <url><loc>https://zenn.dev/articles/not-a-topic</loc></url>
  <url><loc>https://zenn.dev/topics/ローカルllm</loc></url>
</urlset>`)

	got, err := parseTopicSlugsXML(xmlData)
	if err != nil {
		t.Fatalf("parseTopicSlugsXML() error = %v", err)
	}

	assertStringSliceEqual(t, got, []string{"go", "ローカルllm"})
}

func TestParseTopicSlugsGzip(t *testing.T) {
	t.Parallel()

	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://zenn.dev/topics/nextjs</loc></url>
  <url><loc>https://zenn.dev/topics/golang</loc></url>
</urlset>`)

	got, err := parseTopicSlugsGzip(mustGzip(t, xmlData))
	if err != nil {
		t.Fatalf("parseTopicSlugsGzip() error = %v", err)
	}

	assertStringSliceEqual(t, got, []string{"nextjs", "golang"})
}

func TestFilterTopics(t *testing.T) {
	t.Parallel()

	slugs := []string{"Rust", "golang", "GoRouter", "ローカルllm"}
	got := filterTopics(slugs, "go")
	want := []string{"golang", "GoRouter"}

	assertStringSliceEqual(t, got, want)
}

func TestRunWithDepsPrintsMatches(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	fetch := func(context.Context) ([]string, error) {
		return []string{"rust", "golang", "GoRouter"}, nil
	}

	err := runWithDeps(context.Background(), []string{"go"}, &out, runDeps{fetch: fetch})
	if err != nil {
		t.Fatalf("runWithDeps() error = %v", err)
	}

	got := out.String()
	want := "golang\nGoRouter\n"
	if got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestRunWithDepsPrintsHelp(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := runWithDeps(context.Background(), []string{"--help"}, &out, runDeps{})
	if err != nil {
		t.Fatalf("runWithDeps() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Usage:") {
		t.Fatalf("help output missing Usage: %q", got)
	}
	if !strings.Contains(got, "--verbose") {
		t.Fatalf("help output missing --verbose: %q", got)
	}
	if !strings.Contains(got, "--version") {
		t.Fatalf("help output missing --version: %q", got)
	}
}

func TestRunWithDepsPrintsVersion(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := runWithDeps(context.Background(), []string{"--version"}, &out, runDeps{})
	if err != nil {
		t.Fatalf("runWithDeps() error = %v", err)
	}

	got := out.String()
	want := version + "\n"
	if got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestRunWithDepsVerboseWritesStderr(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	var errOut bytes.Buffer
	err := runWithDepsIO(context.Background(), []string{"--verbose", "go"}, &out, &errOut, runDeps{
		fetch: func(context.Context) ([]string, error) {
			return []string{"go"}, nil
		},
	})
	if err != nil {
		t.Fatalf("runWithDepsIO() error = %v", err)
	}
	if out.String() != "go\n" {
		t.Fatalf("stdout = %q, want %q", out.String(), "go\n")
	}
	if !strings.Contains(errOut.String(), "verbose: fetching topics from network") {
		t.Fatalf("stderr = %q, want verbose fetch message", errOut.String())
	}
}

func TestRunWithDepsPrintsNoMatchesMessage(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := runWithDeps(context.Background(), []string{"nomatch"}, &out, runDeps{
		fetch: func(context.Context) ([]string, error) {
			return []string{"go", "rust"}, nil
		},
	})
	if err != nil {
		t.Fatalf("runWithDeps() error = %v", err)
	}

	got := out.String()
	want := "No topics matched query: nomatch\n"
	if got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestRunWithDepsMissingQuery(t *testing.T) {
	t.Parallel()

	err := runWithDeps(context.Background(), nil, &bytes.Buffer{}, runDeps{
		fetch: func(context.Context) ([]string, error) {
			t.Fatal("fetch should not be called")
			return nil, nil
		},
	})
	if err == nil {
		t.Fatal("runWithDeps() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "usage: zenn-topics [options] <query>") {
		t.Fatalf("error = %q, want usage message", err)
	}
}

func TestRunWithDepsLoadError(t *testing.T) {
	t.Parallel()

	err := runWithDeps(context.Background(), []string{"go"}, &bytes.Buffer{}, runDeps{
		fetch: func(context.Context) ([]string, error) {
			return nil, errors.New("boom")
		},
		cache:    nil,
		now:      func() time.Time { return time.Now() },
		cacheTTL: time.Hour,
	})
	if err == nil {
		t.Fatal("runWithDeps() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "load topics") {
		t.Fatalf("error = %q, want wrapped load error", err)
	}
}

func TestRunWithDepsRefreshFlagUsesFetch(t *testing.T) {
	t.Parallel()

	cachePath := filepath.Join(t.TempDir(), "topics.json")
	cache := diskTopicCache{path: cachePath}
	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)
	if err := cache.Save(now, []string{"cached"}); err != nil {
		t.Fatalf("cache.Save() error = %v", err)
	}

	fetchCalls := 0
	var out bytes.Buffer
	err := runWithDeps(context.Background(), []string{"--refresh", "fresh"}, &out, runDeps{
		fetch: func(context.Context) ([]string, error) {
			fetchCalls++
			return []string{"fresh"}, nil
		},
		cache:    cache,
		now:      func() time.Time { return now },
		cacheTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("runWithDeps() error = %v", err)
	}
	if fetchCalls != 1 {
		t.Fatalf("fetchCalls = %d, want 1", fetchCalls)
	}
	if out.String() != "fresh\n" {
		t.Fatalf("stdout = %q, want %q", out.String(), "fresh\n")
	}
}

func assertStringSliceEqual(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d; got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func mustGzip(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}
