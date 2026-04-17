package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/massonsky/buffalo/internal/version"
	"github.com/massonsky/buffalo/pkg/errors"
)

const (
	// GitHubAPIURL is the base URL for GitHub API.
	GitHubAPIURL = "https://api.github.com"

	// DefaultOwner is the default GitHub repository owner.
	DefaultOwner = "massonsky"

	// DefaultRepo is the default GitHub repository name.
	DefaultRepo = "buffalo"
)

// Checker checks for Buffalo updates.
type Checker struct {
	client     *http.Client
	owner      string
	repo       string
	apiBaseURL string
}

// CheckerOption is a functional option for Checker.
type CheckerOption func(*Checker)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) CheckerOption {
	return func(c *Checker) {
		c.client = client
	}
}

// WithRepository sets a custom GitHub repository.
func WithRepository(owner, repo string) CheckerOption {
	return func(c *Checker) {
		c.owner = owner
		c.repo = repo
	}
}

// WithAPIBaseURL sets a custom API base URL (for testing).
func WithAPIBaseURL(url string) CheckerOption {
	return func(c *Checker) {
		c.apiBaseURL = url
	}
}

// NewChecker creates a new update checker.
func NewChecker(opts ...CheckerOption) *Checker {
	c := &Checker{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		owner:      DefaultOwner,
		repo:       DefaultRepo,
		apiBaseURL: GitHubAPIURL,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// CheckForUpdates checks if a newer version is available.
func (c *Checker) CheckForUpdates(ctx context.Context) (*UpgradeCheck, error) {
	latest, err := c.GetLatestRelease(ctx)
	if err != nil {
		return nil, err
	}

	currentVersion := version.Version
	latestVersion := strings.TrimPrefix(latest.Version, "v")

	updateAvailable := compareVersions(latestVersion, currentVersion) > 0

	check := &UpgradeCheck{
		CurrentVersion:  currentVersion,
		LatestVersion:   latestVersion,
		UpdateAvailable: updateAvailable,
		LatestRelease:   latest,
		MigrationSteps:  []MigrationStep{}, // Will be populated by migrator
	}

	return check, nil
}

// GetLatestRelease fetches the latest release from GitHub.
func (c *Checker) GetLatestRelease(ctx context.Context) (*ReleaseInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", c.apiBaseURL, c.owner, c.repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrIO, "failed to create request")
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", fmt.Sprintf("Buffalo/%s", version.Version))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrIO, "failed to fetch latest release")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New(errors.ErrNotFound, "no releases found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.New(errors.ErrIO, fmt.Sprintf("GitHub API error: %s - %s", resp.Status, string(body)))
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidInput, "failed to parse release info")
	}

	return &release, nil
}

// GetReleases fetches all releases from GitHub.
func (c *Checker) GetReleases(ctx context.Context, includePrerelease bool) ([]*ReleaseInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases", c.apiBaseURL, c.owner, c.repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrIO, "failed to create request")
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", fmt.Sprintf("Buffalo/%s", version.Version))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrIO, "failed to fetch releases")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.New(errors.ErrIO, fmt.Sprintf("GitHub API error: %s - %s", resp.Status, string(body)))
	}

	var releases []*ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidInput, "failed to parse releases")
	}

	// Filter out prereleases and drafts if not requested
	if !includePrerelease {
		var filtered []*ReleaseInfo
		for _, r := range releases {
			if !r.Prerelease && !r.Draft {
				filtered = append(filtered, r)
			}
		}
		releases = filtered
	}

	return releases, nil
}

// GetRelease fetches a specific release by tag.
func (c *Checker) GetRelease(ctx context.Context, tag string) (*ReleaseInfo, error) {
	// Ensure tag has 'v' prefix for GitHub
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}

	url := fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s", c.apiBaseURL, c.owner, c.repo, tag)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrIO, "failed to create request")
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", fmt.Sprintf("Buffalo/%s", version.Version))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrIO, "failed to fetch release")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New(errors.ErrNotFound, fmt.Sprintf("release %s not found", tag))
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.New(errors.ErrIO, fmt.Sprintf("GitHub API error: %s - %s", resp.Status, string(body)))
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidInput, "failed to parse release info")
	}

	return &release, nil
}

// GetAssetForPlatform finds the appropriate asset for the current platform.
func (c *Checker) GetAssetForPlatform(release *ReleaseInfo) (*ReleaseAsset, error) {
	os := runtime.GOOS
	arch := runtime.GOARCH

	// Map arch names
	archNames := map[string][]string{
		"amd64": {"amd64", "x86_64", "x64"},
		"arm64": {"arm64", "aarch64"},
		"386":   {"386", "i386", "x86"},
	}

	// Build possible asset name patterns
	var patterns []string
	for _, archName := range archNames[arch] {
		patterns = append(patterns,
			fmt.Sprintf("buffalo_%s_%s", os, archName),
			fmt.Sprintf("buffalo-%s-%s", os, archName),
			fmt.Sprintf("buffalo_%s_%s.tar.gz", os, archName),
			fmt.Sprintf("buffalo-%s-%s.tar.gz", os, archName),
			fmt.Sprintf("buffalo_%s_%s.zip", os, archName),
			fmt.Sprintf("buffalo-%s-%s.zip", os, archName),
		)

		// Windows-specific patterns
		if os == "windows" {
			patterns = append(patterns,
				fmt.Sprintf("buffalo_%s_%s.exe", os, archName),
				fmt.Sprintf("buffalo-%s-%s.exe", os, archName),
			)
		}
	}

	// Find matching asset
	for _, asset := range release.Assets {
		nameLower := strings.ToLower(asset.Name)
		for _, pattern := range patterns {
			if strings.Contains(nameLower, strings.ToLower(pattern)) {
				return &asset, nil
			}
		}
	}

	return nil, errors.New(errors.ErrNotFound,
		fmt.Sprintf("no binary found for %s/%s in release %s", os, arch, release.Version))
}

// GetChangelogBetweenVersions returns changelog for versions between from and to.
func (c *Checker) GetChangelogBetweenVersions(ctx context.Context, fromVersion, toVersion string) (string, error) {
	releases, err := c.GetReleases(ctx, false)
	if err != nil {
		return "", err
	}

	var relevantReleases []*ReleaseInfo
	for _, r := range releases {
		ver := strings.TrimPrefix(r.Version, "v")
		if compareVersions(ver, fromVersion) > 0 && compareVersions(ver, toVersion) <= 0 {
			relevantReleases = append(relevantReleases, r)
		}
	}

	// Sort by version descending (newest first)
	sort.Slice(relevantReleases, func(i, j int) bool {
		vi := strings.TrimPrefix(relevantReleases[i].Version, "v")
		vj := strings.TrimPrefix(relevantReleases[j].Version, "v")
		return compareVersions(vi, vj) > 0
	})

	var changelog strings.Builder
	for _, r := range relevantReleases {
		changelog.WriteString(fmt.Sprintf("## %s\n", r.Version))
		if r.Name != "" && r.Name != r.Version {
			changelog.WriteString(fmt.Sprintf("**%s**\n", r.Name))
		}
		changelog.WriteString(fmt.Sprintf("Released: %s\n\n", r.PublishedAt.Format("2006-01-02")))
		changelog.WriteString(r.Body)
		changelog.WriteString("\n\n---\n\n")
	}

	return changelog.String(), nil
}

// compareVersions compares two semantic versions.
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func compareVersions(v1, v2 string) int {
	// Remove 'v' prefix if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Split into parts
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare each part
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int

		if i < len(parts1) {
			// Handle pre-release suffixes (e.g., "1-beta")
			part := strings.Split(parts1[i], "-")[0]
			_, _ = fmt.Sscanf(part, "%d", &n1)
		}

		if i < len(parts2) {
			part := strings.Split(parts2[i], "-")[0]
			_, _ = fmt.Sscanf(part, "%d", &n2)
		}

		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}

	// Check for pre-release (pre-release < release)
	hasPrerelease1 := strings.Contains(v1, "-")
	hasPrerelease2 := strings.Contains(v2, "-")

	if hasPrerelease1 && !hasPrerelease2 {
		return -1
	}
	if !hasPrerelease1 && hasPrerelease2 {
		return 1
	}

	return 0
}

// CompareVersions is exported for external use.
func CompareVersions(v1, v2 string) int {
	return compareVersions(v1, v2)
}
