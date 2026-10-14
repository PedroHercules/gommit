package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode"
)

const (
	npmLatestURL       = "https://registry.npmjs.org/gmit/latest"
	githubReleaseAPI   = "https://api.github.com/repos/PedroHercules/gommit/releases/tags/v"
	githubReleasesPage = "https://github.com/PedroHercules/gommit/releases"
	updateCheckTimeout = 2 * time.Second
	notesCacheTTL      = 24 * time.Hour
	notesRetryTTL      = 5 * time.Minute
)

var semverPattern = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z.-]+))?(?:\+[0-9A-Za-z.-]+)?$`)
var ansiEscapePattern = regexp.MustCompile(`\x1b(\[[0-?]*[ -/]*[@-~]|\][^\x07]*(\x07|\x1b\\))`)

type updateNotice struct {
	CurrentVersion string
	LatestVersion  string
	ReleaseURL     string
	Highlights     []string
}

type releaseNotesCache struct {
	Version    string    `json:"version"`
	ReleaseURL string    `json:"release_url"`
	Highlights []string  `json:"highlights"`
	Available  bool      `json:"available"`
	FetchedAt  time.Time `json:"fetched_at"`
}

type updateManager struct {
	currentVersion string
	client         *http.Client
	registryURL    string
	releaseAPIURL  string
	cacheDir       func() (string, error)
	install        func() error
}

func newUpdateManager(currentVersion string) *updateManager {
	return &updateManager{
		currentVersion: currentVersion,
		client:         &http.Client{},
		registryURL:    npmLatestURL,
		releaseAPIURL:  githubReleaseAPI,
		cacheDir:       func() (string, error) { return os.UserCacheDir() },
		install:        runNPMUpdate,
	}
}

func (m *updateManager) check() (updateNotice, error) {
	if m.currentVersion == "" || m.currentVersion == "dev" {
		return updateNotice{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
	defer cancel()

	latest, err := m.latestVersion(ctx)
	if err != nil {
		return updateNotice{}, err
	}
	newer, err := isNewerVersion(latest, m.currentVersion)
	if err != nil || !newer {
		return updateNotice{CurrentVersion: m.currentVersion, LatestVersion: latest}, err
	}

	notice := updateNotice{
		CurrentVersion: m.currentVersion,
		LatestVersion:  latest,
		ReleaseURL:     githubReleasesPage + "/tag/v" + latest,
	}
	if cached, ok := m.loadNotesCache(latest); ok {
		notice.ReleaseURL = cached.ReleaseURL
		notice.Highlights = cached.Highlights
		return notice, nil
	}

	notesAvailable := false
	if notes, notesErr := m.releaseNotes(ctx, latest); notesErr == nil {
		notesAvailable = true
		if notes.HTMLURL != "" {
			notice.ReleaseURL = notes.HTMLURL
		}
		notice.Highlights = extractReleaseHighlights(notes.Body, 3)
	}
	m.saveNotesCache(releaseNotesCache{
		Version:    latest,
		ReleaseURL: notice.ReleaseURL,
		Highlights: notice.Highlights,
		Available:  notesAvailable,
		FetchedAt:  time.Now(),
	})
	return notice, nil
}

func (m *updateManager) latestVersion(ctx context.Context) (string, error) {
	var result struct {
		Version string `json:"version"`
	}
	if err := m.getJSON(ctx, m.registryURL, &result); err != nil {
		return "", fmt.Errorf("query npm registry: %w", err)
	}
	if _, _, _, _, ok := parseSemver(result.Version); !ok {
		return "", fmt.Errorf("npm registry returned an invalid version")
	}
	return result.Version, nil
}

func (m *updateManager) releaseNotes(ctx context.Context, version string) (struct {
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
}, error) {
	var result struct {
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
	}
	endpoint := m.releaseAPIURL + url.PathEscape("v"+version)
	if err := m.getJSON(ctx, endpoint, &result); err != nil {
		return result, err
	}
	return result, nil
}

func (m *updateManager) getJSON(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gmit-update-check")
	client := m.client
	if client == nil {
		client = &http.Client{}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("request failed with HTTP %d", resp.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	return decoder.Decode(target)
}

func (m *updateManager) loadNotesCache(version string) (releaseNotesCache, bool) {
	path, err := m.notesCachePath()
	if err != nil {
		return releaseNotesCache{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return releaseNotesCache{}, false
	}
	var cache releaseNotesCache
	if json.Unmarshal(data, &cache) != nil || cache.Version != version {
		return releaseNotesCache{}, false
	}
	maxAge := notesCacheTTL
	if !cache.Available {
		maxAge = notesRetryTTL
	}
	if time.Since(cache.FetchedAt) > maxAge {
		return releaseNotesCache{}, false
	}
	return cache, true
}

func (m *updateManager) saveNotesCache(cache releaseNotesCache) {
	path, err := m.notesCachePath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return
	}
	data, err := json.Marshal(cache)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0600)
}

func (m *updateManager) notesCachePath() (string, error) {
	cacheDir := m.cacheDir
	if cacheDir == nil {
		cacheDir = func() (string, error) { return os.UserCacheDir() }
	}
	base, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "gmit", "release-notes.json"), nil
}

func (m *updateManager) update() error {
	if m.install == nil {
		return fmt.Errorf("npm update is unavailable")
	}
	return m.install()
}

func runNPMUpdate() error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("run `gmit update` through the npm-installed Gmit command so Windows can replace the running executable safely")
	}
	cmd := exec.Command("npm", "install", "--global", "gmit@latest")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm update failed: %w", err)
	}
	return nil
}

func (n updateNotice) printTo(w io.Writer) {
	if w == nil || n.LatestVersion == "" || n.ReleaseURL == "" {
		return
	}
	fmt.Fprintf(w, "! Gmit update available: %s → %s\n", n.CurrentVersion, n.LatestVersion)
	for _, highlight := range n.Highlights {
		fmt.Fprintf(w, "  • %s\n", highlight)
	}
	fmt.Fprintf(w, "  Release notes: %s\n  Run `gmit update` to install.\n\n", n.ReleaseURL)
}

func isNewerVersion(candidate, current string) (bool, error) {
	candidateMajor, candidateMinor, candidatePatch, candidatePre, candidateOK := parseSemver(candidate)
	currentMajor, currentMinor, currentPatch, currentPre, currentOK := parseSemver(current)
	if !candidateOK || !currentOK {
		return false, fmt.Errorf("cannot compare invalid semantic versions %q and %q", candidate, current)
	}
	if candidateMajor != currentMajor {
		return candidateMajor > currentMajor, nil
	}
	if candidateMinor != currentMinor {
		return candidateMinor > currentMinor, nil
	}
	if candidatePatch != currentPatch {
		return candidatePatch > currentPatch, nil
	}
	if candidatePre == currentPre {
		return false, nil
	}
	if candidatePre == "" {
		return true, nil
	}
	if currentPre == "" {
		return false, nil
	}
	return comparePrerelease(candidatePre, currentPre) > 0, nil
}

func parseSemver(version string) (major, minor, patch int, prerelease string, ok bool) {
	matches := semverPattern.FindStringSubmatch(version)
	if matches == nil {
		return 0, 0, 0, "", false
	}
	if _, err := fmt.Sscanf(matches[1], "%d", &major); err != nil {
		return 0, 0, 0, "", false
	}
	if _, err := fmt.Sscanf(matches[2], "%d", &minor); err != nil {
		return 0, 0, 0, "", false
	}
	if _, err := fmt.Sscanf(matches[3], "%d", &patch); err != nil {
		return 0, 0, 0, "", false
	}
	return major, minor, patch, matches[4], true
}

func comparePrerelease(left, right string) int {
	leftParts := strings.Split(left, ".")
	rightParts := strings.Split(right, ".")
	for i := 0; i < len(leftParts) && i < len(rightParts); i++ {
		leftNumber, leftErr := numericIdentifier(leftParts[i])
		rightNumber, rightErr := numericIdentifier(rightParts[i])
		switch {
		case leftErr == nil && rightErr == nil && leftNumber != rightNumber:
			if leftNumber > rightNumber {
				return 1
			}
			return -1
		case leftErr == nil && rightErr != nil:
			return -1
		case leftErr != nil && rightErr == nil:
			return 1
		case leftParts[i] != rightParts[i]:
			if leftParts[i] > rightParts[i] {
				return 1
			}
			return -1
		}
	}
	if len(leftParts) > len(rightParts) {
		return 1
	}
	if len(leftParts) < len(rightParts) {
		return -1
	}
	return 0
}

func numericIdentifier(identifier string) (int, error) {
	var value int
	if _, err := fmt.Sscanf(identifier, "%d", &value); err != nil || fmt.Sprintf("%d", value) != identifier {
		return 0, fmt.Errorf("not a numeric identifier")
	}
	return value, nil
}

func extractReleaseHighlights(body string, max int) []string {
	if max <= 0 {
		return nil
	}
	var highlights []string
	include := false
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			heading := strings.ToLower(strings.TrimSpace(strings.TrimLeft(trimmed, "#")))
			include = strings.Contains(heading, "feature") || strings.Contains(heading, "bug fix") || strings.Contains(heading, "breaking change")
			continue
		}
		if !include || !(strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- ")) {
			continue
		}
		item := sanitizeTerminalText(trimmed[2:])
		if item == "" {
			continue
		}
		highlights = append(highlights, truncateRunes(item, 150))
		if len(highlights) >= max {
			return highlights
		}
	}
	return highlights
}

func sanitizeTerminalText(value string) string {
	value = ansiEscapePattern.ReplaceAllString(value, "")
	var clean strings.Builder
	for _, character := range value {
		if unicode.IsControl(character) {
			if character == '\t' {
				clean.WriteByte(' ')
			}
			continue
		}
		clean.WriteRune(character)
	}
	return strings.TrimSpace(clean.String())
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}
