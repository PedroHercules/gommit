package cli

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func jsonResponse(request *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    request,
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name      string
		candidate string
		current   string
		want      bool
		wantErr   bool
	}{
		{name: "patch update", candidate: "1.2.4", current: "1.2.3", want: true},
		{name: "same version", candidate: "1.2.3", current: "v1.2.3", want: false},
		{name: "stable follows prerelease", candidate: "1.2.3", current: "1.2.3-rc.1", want: true},
		{name: "prerelease is not newer than stable", candidate: "1.2.3-rc.2", current: "1.2.3", want: false},
		{name: "prerelease identifiers compare numerically", candidate: "1.2.3-rc.10", current: "1.2.3-rc.2", want: true},
		{name: "invalid version", candidate: "latest", current: "1.2.3", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isNewerVersion(tt.candidate, tt.current)
			if (err != nil) != tt.wantErr {
				t.Fatalf("isNewerVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("isNewerVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractReleaseHighlights(t *testing.T) {
	body := `
## 1.2.0 (2026-10-14)

### Features

* feat: add model search (abc123)
* feat: add update command (def456)

### Bug Fixes

* fix: handle empty provider response (ghi789)

### Chores

* chore: update dependencies (jkl012)
`
	got := extractReleaseHighlights(body, 3)
	want := []string{
		"feat: add model search (abc123)",
		"feat: add update command (def456)",
		"fix: handle empty provider response (ghi789)",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d highlights, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("highlight %d = %q, want %q", i, got[i], want[i])
		}
	}
	if got := extractReleaseHighlights(body, 0); got != nil {
		t.Fatalf("zero limit should return nil, got %#v", got)
	}
}

func TestExtractReleaseHighlightsRemovesTerminalControlCharacters(t *testing.T) {
	got := extractReleaseHighlights("### Features\n\n* \x1b[31mfeat: safe text\x1b[0m\n", 1)
	if len(got) != 1 || got[0] != "feat: safe text" {
		t.Fatalf("unexpected sanitized highlight: %#v", got)
	}
}

func TestUpdateManagerChecksRegistryAndCachesReleaseNotes(t *testing.T) {
	var registryCalls, releaseCalls int
	manager := newUpdateManager("1.1.0")
	manager.registryURL = "https://unit.test/gmit/latest"
	manager.releaseAPIURL = "https://unit.test/releases/tags/"
	manager.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/gmit/latest":
			registryCalls++
			return jsonResponse(r, http.StatusOK, `{"version":"1.2.0"}`), nil
		case "/releases/tags/v1.2.0":
			releaseCalls++
			return jsonResponse(r, http.StatusOK, `{"html_url":"https://github.test/releases/tag/v1.2.0","body":"### Features\n\n* feat: add update check\n"}`), nil
		default:
			return jsonResponse(r, http.StatusNotFound, `{"message":"not found"}`), nil
		}
	})}
	cacheDir := t.TempDir()
	manager.cacheDir = func() (string, error) { return cacheDir, nil }

	first, err := manager.check()
	if err != nil {
		t.Fatalf("first check failed: %v", err)
	}
	second, err := manager.check()
	if err != nil {
		t.Fatalf("second check failed: %v", err)
	}
	if first.LatestVersion != "1.2.0" || second.LatestVersion != "1.2.0" {
		t.Fatalf("latest versions = %q and %q, want 1.2.0", first.LatestVersion, second.LatestVersion)
	}
	if len(first.Highlights) != 1 || first.Highlights[0] != "feat: add update check" {
		t.Fatalf("unexpected highlights: %#v", first.Highlights)
	}
	if registryCalls != 2 {
		t.Errorf("registry calls = %d, want 2 checks", registryCalls)
	}
	if releaseCalls != 1 {
		t.Errorf("release notes calls = %d, want cached single request", releaseCalls)
	}
}

func TestUpdateManagerFallsBackToReleaseLinkWhenNotesUnavailable(t *testing.T) {
	manager := newUpdateManager("1.9.0")
	manager.registryURL = "https://unit.test/gmit/latest"
	manager.releaseAPIURL = "https://unit.test/releases/tags/"
	manager.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/gmit/latest" {
			return jsonResponse(r, http.StatusOK, `{"version":"2.0.0"}`), nil
		}
		return jsonResponse(r, http.StatusNotFound, `{"message":"not found"}`), nil
	})}
	manager.cacheDir = func() (string, error) { return t.TempDir(), nil }
	notice, err := manager.check()
	if err != nil {
		t.Fatalf("check failed when release notes unavailable: %v", err)
	}
	if notice.ReleaseURL != githubReleasesPage+"/tag/v2.0.0" {
		t.Fatalf("fallback release URL = %q", notice.ReleaseURL)
	}
	if len(notice.Highlights) != 0 {
		t.Fatalf("expected no highlights, got %#v", notice.Highlights)
	}
}

func TestUpdateCheckFailureDoesNotPrintNotice(t *testing.T) {
	manager := newUpdateManager("1.0.0")
	manager.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, io.ErrUnexpectedEOF
	})}
	var output bytes.Buffer
	cli := &CLI{updates: manager, updateOutput: &output}
	if notice := cli.checkForUpdate(); notice.LatestVersion != "" {
		t.Fatalf("failed update check returned a release: %#v", notice)
	}
	if output.Len() != 0 {
		t.Fatalf("failed update check printed a notice: %q", output.String())
	}
}

func TestRunShowsUpdateNoticeBeforeVersionCommand(t *testing.T) {
	manager := newUpdateManager("1.0.0")
	manager.registryURL = "https://unit.test/gmit/latest"
	manager.releaseAPIURL = "https://unit.test/releases/tags/"
	manager.cacheDir = func() (string, error) { return t.TempDir(), nil }
	manager.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/gmit/latest" {
			return jsonResponse(r, http.StatusOK, `{"version":"1.1.0"}`), nil
		}
		return jsonResponse(r, http.StatusOK, `{"html_url":"https://github.test/releases/tag/v1.1.0","body":"### Features\n\n* feat: improve models\n"}`), nil
	})}
	var output bytes.Buffer
	cli := &CLI{updates: manager, updateOutput: &output}
	if err := cli.Run([]string{"gmit", "version"}); err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	for _, want := range []string{"1.0.0 → 1.1.0", "feat: improve models", "https://github.test/releases/tag/v1.1.0"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("update notice missing %q: %q", want, output.String())
		}
	}
}

func TestReleaseNotesCacheUsesShortRetryForUnavailableNotes(t *testing.T) {
	cacheDir := t.TempDir()
	manager := newUpdateManager("1.0.0")
	manager.cacheDir = func() (string, error) { return cacheDir, nil }
	manager.saveNotesCache(releaseNotesCache{
		Version:   "1.1.0",
		FetchedAt: time.Now().Add(-notesRetryTTL - time.Second),
	})
	if _, ok := manager.loadNotesCache("1.1.0"); ok {
		t.Fatal("unavailable release notes should be retried after the short cache period")
	}
	manager.saveNotesCache(releaseNotesCache{
		Version:    "1.1.0",
		Available:  true,
		Highlights: []string{"feat: cached notes"},
		FetchedAt:  time.Now().Add(-notesCacheTTL + time.Hour),
	})
	if cached, ok := manager.loadNotesCache("1.1.0"); !ok || len(cached.Highlights) != 1 {
		t.Fatalf("valid release note cache not reused: %#v, ok=%v", cached, ok)
	}
}

func TestUpdateNoticePrintsVersionHighlightsAndLink(t *testing.T) {
	notice := updateNotice{
		CurrentVersion: "1.0.0",
		LatestVersion:  "1.1.0",
		ReleaseURL:     "https://github.com/PedroHercules/gommit/releases/tag/v1.1.0",
		Highlights:     []string{"feat: update available"},
	}
	var output bytes.Buffer
	notice.printTo(&output)
	for _, want := range []string{"1.0.0 → 1.1.0", "feat: update available", notice.ReleaseURL, "gmit update"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("update notice missing %q in %q", want, output.String())
		}
	}
}

func TestHandleUpdateRunsInstaller(t *testing.T) {
	called := false
	manager := newUpdateManager("1.0.0")
	manager.install = func() error {
		called = true
		return nil
	}
	cli := &CLI{updates: manager}
	if err := cli.handleUpdate(updateNotice{CurrentVersion: "1.0.0", LatestVersion: "1.1.0"}); err != nil {
		t.Fatalf("handleUpdate() error = %v", err)
	}
	if !called {
		t.Fatal("update installer was not called")
	}
}
