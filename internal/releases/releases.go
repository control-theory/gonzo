package releases

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

// Release represents a GitHub release.
type Release struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	URL         string `json:"url"`
}

// githubRelease is the raw GitHub API response shape (we only extract what we need).
type githubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
}

// Fetcher fetches and caches GitHub releases.
type Fetcher struct {
	releases []Release
	done     chan struct{}
	once     sync.Once
}

// NewFetcher creates a new releases fetcher.
func NewFetcher() *Fetcher {
	return &Fetcher{
		done: make(chan struct{}),
	}
}

// FetchInBackground starts a goroutine to fetch releases from the GitHub API.
// Non-blocking. Silently ignores errors.
func (f *Fetcher) FetchInBackground() {
	go func() {
		defer func() {
			f.once.Do(func() { close(f.done) })
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET",
			"https://api.github.com/repos/control-theory/gonzo/releases", nil)
		if err != nil {
			return
		}
		req.Header.Set("Accept", "application/vnd.github+json")

		resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return
		}

		var ghReleases []githubRelease
		if err := json.Unmarshal(body, &ghReleases); err != nil {
			return
		}

		releases := make([]Release, 0, len(ghReleases))
		for _, gh := range ghReleases {
			releases = append(releases, Release{
				TagName:     gh.TagName,
				Name:        gh.Name,
				Body:        gh.Body,
				PublishedAt: gh.PublishedAt,
				URL:         gh.HTMLURL,
			})
		}

		f.releases = releases
	}()
}

// GetReleases returns fetched releases (newest first), or nil if not ready/failed.
// On the first call, waits up to 2 seconds for the fetch to complete.
func (f *Fetcher) GetReleases() []Release {
	select {
	case <-f.done:
		return f.releases
	case <-time.After(2 * time.Second):
		return f.releases
	}
}

// GetReleasesNonBlocking returns whatever releases are available without waiting.
func (f *Fetcher) GetReleasesNonBlocking() []Release {
	select {
	case <-f.done:
		return f.releases
	default:
		return nil
	}
}

// GetRelease returns the release matching the given tag (e.g. "v0.3.1"), or nil.
func (f *Fetcher) GetRelease(tag string) *Release {
	rels := f.GetReleasesNonBlocking()
	for i := range rels {
		if rels[i].TagName == tag {
			return &rels[i]
		}
	}
	return nil
}
