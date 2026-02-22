package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	sitemapIndexURL  = "https://zenn.dev/sitemaps/_index.xml"
	topicPathPrefix  = "/topics/"
	defaultCacheTTL  = 12 * time.Hour
	defaultCacheFile = "topics.json"
)

type fetchTopicsFunc func(ctx context.Context) ([]string, error)

type topicCache interface {
	Load(now time.Time, ttl time.Duration) ([]string, bool, error)
	Save(now time.Time, slugs []string) error
}

type runDeps struct {
	fetch    fetchTopicsFunc
	cache    topicCache
	now      func() time.Time
	cacheTTL time.Duration
}

type cliOptions struct {
	Query   string
	Refresh bool
}

type diskTopicCache struct {
	path string
}

type topicsCacheFile struct {
	FetchedAt time.Time `json:"fetched_at"`
	Slugs     []string  `json:"slugs"`
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, fetchTopics); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer, fetch fetchTopicsFunc) error {
	cachePath, err := defaultTopicCachePath()
	if err != nil {
		return fmt.Errorf("resolve cache path: %w", err)
	}

	deps := runDeps{
		fetch:    fetch,
		cache:    diskTopicCache{path: cachePath},
		now:      time.Now,
		cacheTTL: defaultCacheTTL,
	}

	return runWithDeps(ctx, args, stdout, deps)
}

func runWithDeps(ctx context.Context, args []string, stdout io.Writer, deps runDeps) error {
	opts, err := parseCLIArgs(args)
	if err != nil {
		return err
	}

	topics, err := loadTopics(ctx, deps, opts.Refresh)
	if err != nil {
		return fmt.Errorf("load topics: %w", err)
	}

	for _, slug := range filterTopics(topics, opts.Query) {
		if _, err := fmt.Fprintln(stdout, slug); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}

	return nil
}

func parseCLIArgs(args []string) (cliOptions, error) {
	var opts cliOptions
	var positionals []string

	for _, arg := range args {
		switch arg {
		case "--refresh":
			opts.Refresh = true
		default:
			if strings.HasPrefix(arg, "-") {
				return cliOptions{}, fmt.Errorf("unknown option: %s", arg)
			}
			positionals = append(positionals, arg)
		}
	}

	if len(positionals) != 1 {
		return cliOptions{}, fmt.Errorf("usage: zenn-topics [--refresh] <query>")
	}

	opts.Query = strings.TrimSpace(positionals[0])
	if opts.Query == "" {
		return cliOptions{}, fmt.Errorf("query must not be empty")
	}

	return opts, nil
}

func loadTopics(ctx context.Context, deps runDeps, refresh bool) ([]string, error) {
	if deps.fetch == nil {
		return nil, fmt.Errorf("fetch function is nil")
	}
	if deps.now == nil {
		deps.now = time.Now
	}
	if deps.cacheTTL <= 0 {
		deps.cacheTTL = defaultCacheTTL
	}

	now := deps.now()

	if deps.cache != nil && !refresh {
		slugs, hit, err := deps.cache.Load(now, deps.cacheTTL)
		if err != nil {
			return nil, fmt.Errorf("read cache: %w", err)
		}
		if hit {
			return slugs, nil
		}
	}

	slugs, err := deps.fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch topics: %w", err)
	}

	if deps.cache != nil {
		if err := deps.cache.Save(now, slugs); err != nil {
			return nil, fmt.Errorf("write cache: %w", err)
		}
	}

	return slugs, nil
}

func defaultTopicCachePath() (string, error) {
	root, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("empty cache directory")
	}
	return filepath.Join(root, "zenn-topics", defaultCacheFile), nil
}

func (c diskTopicCache) Load(now time.Time, ttl time.Duration) ([]string, bool, error) {
	if c.path == "" {
		return nil, false, fmt.Errorf("empty cache path")
	}

	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	var payload topicsCacheFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, false, err
	}
	if payload.FetchedAt.IsZero() {
		return nil, false, fmt.Errorf("missing fetched_at")
	}

	if now.Sub(payload.FetchedAt) > ttl {
		return nil, false, nil
	}

	return append([]string(nil), payload.Slugs...), true, nil
}

func (c diskTopicCache) Save(now time.Time, slugs []string) error {
	if c.path == "" {
		return fmt.Errorf("empty cache path")
	}

	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}

	payload := topicsCacheFile{
		FetchedAt: now.UTC(),
		Slugs:     append([]string(nil), slugs...),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(c.path, data, 0o644)
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
