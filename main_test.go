package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"strings"
	"testing"
)

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
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d; got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
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

	want := []string{"go", "ローカルllm"}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d; got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
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

	want := []string{"nextjs", "golang"}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d; got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestFilterTopics(t *testing.T) {
	t.Parallel()

	slugs := []string{"Rust", "golang", "GoRouter", "ローカルllm"}
	got := filterTopics(slugs, "go")
	want := []string{"golang", "GoRouter"}

	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d; got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRunPrintsMatches(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	fetch := func(context.Context) ([]string, error) {
		return []string{"rust", "golang", "GoRouter"}, nil
	}

	err := run(context.Background(), []string{"go"}, &out, fetch)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	want := "golang\nGoRouter\n"
	if got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestRunMissingQuery(t *testing.T) {
	t.Parallel()

	err := run(context.Background(), nil, &bytes.Buffer{}, func(context.Context) ([]string, error) {
		t.Fatal("fetch should not be called")
		return nil, nil
	})
	if err == nil {
		t.Fatal("run() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "usage: zenn-topics <query>") {
		t.Fatalf("error = %q, want usage message", err)
	}
}

func TestRunFetchError(t *testing.T) {
	t.Parallel()

	err := run(context.Background(), []string{"go"}, &bytes.Buffer{}, func(context.Context) ([]string, error) {
		return nil, errors.New("boom")
	})
	if err == nil {
		t.Fatal("run() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "fetch topics") {
		t.Fatalf("error = %q, want wrapped fetch error", err)
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
