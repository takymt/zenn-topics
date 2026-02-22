package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

const (
	sitemapIndexURL = "https://zenn.dev/sitemaps/_index.xml"
	topicPathPrefix = "/topics/"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, fetchTopics); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

type fetchTopicsFunc func(ctx context.Context) ([]string, error)

func run(ctx context.Context, args []string, stdout io.Writer, fetch fetchTopicsFunc) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: zenn-topics <query>")
	}

	query := strings.TrimSpace(args[0])
	if query == "" {
		return fmt.Errorf("query must not be empty")
	}

	topics, err := fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetch topics: %w", err)
	}

	for _, slug := range filterTopics(topics, query) {
		if _, err := fmt.Fprintln(stdout, slug); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}

	return nil
}

func fetchTopics(ctx context.Context) ([]string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	return fetchTopicsWithClient(ctx, client)
}

func fetchTopicsWithClient(ctx context.Context, client *http.Client) ([]string, error) {
	indexXML, err := fetchBytes(ctx, client, sitemapIndexURL)
	if err != nil {
		return nil, fmt.Errorf("fetch sitemap index: %w", err)
	}

	topicSitemapURLs, err := parseTopicSitemapIndex(indexXML)
	if err != nil {
		return nil, fmt.Errorf("parse sitemap index: %w", err)
	}

	var slugs []string
	for _, sitemapURL := range topicSitemapURLs {
		gzXML, err := fetchBytes(ctx, client, sitemapURL)
		if err != nil {
			return nil, fmt.Errorf("fetch topic sitemap %q: %w", sitemapURL, err)
		}

		parsed, err := parseTopicSlugsGzip(gzXML)
		if err != nil {
			return nil, fmt.Errorf("parse topic sitemap %q: %w", sitemapURL, err)
		}

		slugs = append(slugs, parsed...)
	}

	return slugs, nil
}

func fetchBytes(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

type sitemapIndex struct {
	Sitemaps []sitemapItem `xml:"sitemap"`
}

type sitemapItem struct {
	Loc string `xml:"loc"`
}

type urlSet struct {
	URLs []urlEntry `xml:"url"`
}

type urlEntry struct {
	Loc string `xml:"loc"`
}

func parseTopicSitemapIndex(data []byte) ([]string, error) {
	var idx sitemapIndex
	if err := xml.Unmarshal(data, &idx); err != nil {
		return nil, err
	}

	var urls []string
	for _, item := range idx.Sitemaps {
		if isTopicSitemapURL(item.Loc) {
			urls = append(urls, item.Loc)
		}
	}
	return urls, nil
}

func isTopicSitemapURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	base := path.Base(u.Path)
	return strings.HasPrefix(base, "topic") && strings.HasSuffix(base, ".xml.gz")
}

func parseTopicSlugsGzip(data []byte) ([]string, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = reader.Close()
	}()

	xmlBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return parseTopicSlugsXML(xmlBytes)
}

func parseTopicSlugsXML(data []byte) ([]string, error) {
	var set urlSet
	if err := xml.Unmarshal(data, &set); err != nil {
		return nil, err
	}

	var slugs []string
	for _, entry := range set.URLs {
		slug, ok := extractTopicSlug(entry.Loc)
		if !ok {
			continue
		}
		slugs = append(slugs, slug)
	}
	return slugs, nil
}

func extractTopicSlug(rawURL string) (string, bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}

	if !strings.HasPrefix(u.Path, topicPathPrefix) {
		return "", false
	}

	slug := strings.TrimPrefix(u.Path, topicPathPrefix)
	if slug == "" || strings.Contains(slug, "/") {
		return "", false
	}

	return slug, true
}

func filterTopics(slugs []string, query string) []string {
	queryLower := strings.ToLower(query)

	var out []string
	for _, slug := range slugs {
		if strings.Contains(strings.ToLower(slug), queryLower) {
			out = append(out, slug)
		}
	}
	return out
}
