package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	commit_services "github.com/PedroHercules/gommit/pkg/application/services/commit"
	pr_services "github.com/PedroHercules/gommit/pkg/application/services/pull-request"
	"github.com/PedroHercules/gommit/pkg/domain/entities"
	"github.com/PedroHercules/gommit/pkg/domain/repositories"
)

type fakeSecurityGit struct {
	repositories.GitRepository
	stagedDiff *entities.GitDiff
	prDiff     *entities.GitDiff
	fetchCalls int
}

func (f *fakeSecurityGit) GetStagedDiff() (*entities.GitDiff, error) { return f.stagedDiff, nil }
func (f *fakeSecurityGit) GetDiffBetweenBranches(string, string) (*entities.GitDiff, error) {
	return f.prDiff, nil
}
func (f *fakeSecurityGit) FetchRemote() error { f.fetchCalls++; return nil }

type fakePRPreviewCommandService struct {
	response *pr_services.GeneratePRPreviewResponse
	calls    int
}

func (f *fakePRPreviewCommandService) GeneratePRPreview(*pr_services.GeneratePRPreviewRequest) (*pr_services.GeneratePRPreviewResponse, error) {
	f.calls++
	return f.response, nil
}

func diffFixture(file, changedLines string) string {
	return fmt.Sprintf("diff --git a/%s b/%s\n--- a/%s\n+++ b/%s\n@@ -1,0 +1,%d @@\n%s", file, file, file, file, strings.Count(changedLines, "\n")+1, changedLines)
}

func fixtureAssignment(name, operator, value string) string { return name + operator + value }
func fixtureQuoted(value string) string                     { return `"` + value + `"` }
func randomCredentialFixture() string {
	return strings.Join([]string{"aB3$kL8!qR", "2@vN7#xM4%pC9Q"}, "")
}

func TestSensitiveAssignmentDetectorCoversCommonDeclarationForms(t *testing.T) {
	cases := []struct {
		name, file, line string
	}{
		{"dotenv underscore name", ".env.example", fixtureAssignment("OPENROUTER_API_KEY", "=", "synthetic-token-123456")},
		{"camel case source", "config.go", fixtureAssignment("const clientSecret", " = ", fixtureQuoted("synthetic-secret-123456"))},
		{"acronym camel case source", "config.go", fixtureAssignment("const APIKey", " = ", fixtureQuoted("synthetic-api-key-123456"))},
		{"typed source assignment", "settings.ts", fixtureAssignment("const serviceToken: string", " = ", fixtureQuoted("synthetic-token-123456"))},
		{"JSON property", "settings.json", fixtureAssignment(`"refreshToken"`, ": ", fixtureQuoted("synthetic-refresh-123456"))},
		{"YAML property", "settings.yaml", fixtureAssignment("database_password", ": ", fixtureQuoted("synthetic-db-pass-123456"))},
		{"TOML property", "settings.toml", fixtureAssignment("auth_token", " = ", fixtureQuoted("synthetic-auth-token-123456"))},
		{"generic key suffix", "settings.env", fixtureAssignment("GEMINI_KEY", "=", "synthetic-gemini-value-123456")},
		{"service connection string", "settings.json", fixtureAssignment(`"connectionString"`, ": ", fixtureQuoted(strings.Join([]string{"postgres://user", ":pass@db.example.test/app"}, "")))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report := scanSensitiveDiff(diffFixture(tc.file, "+"+tc.line+"\n"), []string{tc.file})
			if !report.hasFindings() {
				t.Fatalf("expected a finding for %s", tc.line)
			}
		})
	}
}

func TestSensitiveDetectorRecognizesCredentialShapes(t *testing.T) {
	googleToken := strings.Join([]string{"AIzaSyB9c_fake", "SyntheticGoogleKeyValue123456789"}, "")
	jwt := strings.Join([]string{"eyJhbGciOiJub25lIn0", "eyJzdWIiOiJzeW50aGV0aWMifQ", "eyJzaWciOiJub25lIn0"}, ".")
	header := strings.Join([]string{"Authorization:", " Bearer ", jwt}, "")
	privateMarker := strings.Join([]string{"-----BEGIN ", "PRIVATE KEY-----"}, "")
	credentialLink := strings.Join([]string{"https://alice", ":synthetic-password@db.example.test/path"}, "")
	lines := []string{
		"+" + fixtureAssignment("const marker", " = ", fixtureQuoted(googleToken)),
		"+" + header,
		"+" + privateMarker,
		"+" + fixtureAssignment("url", " = ", fixtureQuoted(credentialLink)),
	}
	for _, line := range lines {
		file := "config.txt"
		if strings.Contains(line, "const marker") {
			file = "config.go"
		}
		report := scanSensitiveDiff(diffFixture(file, line+"\n"), []string{file})
		if !report.hasFindings() {
			t.Errorf("expected detector to recognize a credential form in %s", line)
		}
	}
}

func TestEntropyDetectorUsesContextAndBoundaries(t *testing.T) {
	secretLike := randomCredentialFixture()
	cases := []struct {
		name, assignment string
		want             bool
	}{
		{"random long assigned value", fixtureAssignment("opaqueSetting", " = ", fixtureQuoted(secretLike)), true},
		{"short value below minimum", fixtureAssignment("opaqueSetting", " = ", fixtureQuoted("aB3$kL8!qR2@vN7")), false},
		{"repeated value low entropy", fixtureAssignment("opaqueSetting", " = ", fixtureQuoted("aaaaaaaaaaaaaaaaaaaaaaaa")), false},
		{"single character class", fixtureAssignment("opaqueSetting", " = ", fixtureQuoted("abcdefghijklmnopqrstuvwx")), false},
		{"hash name excluded", fixtureAssignment("buildHash", " = ", fixtureQuoted(secretLike)), false},
		{"secret name bypasses entropy threshold", fixtureAssignment("databasePassword", " = ", fixtureQuoted("short-value")), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report := scanSensitiveDiff(diffFixture("config.go", "+"+tc.assignment+"\n"), []string{"config.go"})
			if got := report.hasFindings(); got != tc.want {
				t.Fatalf("hasFindings() = %v, want %v; findings: %#v", got, tc.want, report.findings)
			}
		})
	}
}

func TestPlaceholdersAndBenignIdentifiersAvoidWarnings(t *testing.T) {
	lines := []string{
		"+" + fixtureAssignment("API_KEY", "=", "your_api_key_here"),
		"+" + fixtureAssignment("TOKEN", "=", "${SERVICE_TOKEN}"),
		"+" + fixtureAssignment("serviceMode", "=", "development"),
		"+" + fixtureAssignment("apiEndpoint", "=", "https://api.example.test"),
		"+" + fixtureAssignment("monkey", "=", "banana"),
		"+" + fixtureAssignment("clientMode", "=", "internal"),
		"+" + fixtureAssignment("buildHash", "=", fixtureQuoted(randomCredentialFixture())),
	}
	report := scanSensitiveDiff(diffFixture("settings.env", strings.Join(lines, "\n")+"\n"), []string{"settings.env"})
	if report.hasFindings() {
		t.Fatalf("benign settings and placeholders triggered warnings: %#v", report.findings)
	}
}

func TestGitHubWorkflowReferencesAndExpressionsAreNotSecrets(t *testing.T) {
	checkoutSHA := strings.Join([]string{"3d3c42e5aac5ba805825", "da76410c181273ba90b1"}, "")
	setupGoSHA := strings.Join([]string{"b7ad1dad31e06c5925ef", "5d2fc7ad053ef454303e"}, "")
	setupNodeSHA := strings.Join([]string{"949feb2413d6458794dc", "d2491c4babbbce0c15c1"}, "")
	workflow := strings.Join([]string{
		"+      - uses: actions/checkout@" + checkoutSHA + " # v7.0.1",
		"+      - uses: actions/setup-go@" + setupGoSHA + " # v7.0.0",
		"+      - uses: actions/setup-node@" + setupNodeSHA + " # v7.1.0",
		"+          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}",
	}, "\n") + "\n"
	file := ".github/workflows/ci-cd.yml"
	report := scanSensitiveDiff(diffFixture(file, workflow), []string{file})
	if report.hasFindings() {
		t.Fatalf("pinned GitHub Actions and a GitHub Actions secret reference triggered findings: %#v", report.findings)
	}

	knownToken := strings.Join([]string{"ghp_", "123456789012345678901234567890123456"}, "")
	literalLine := "+          API_TOKEN: \"" + knownToken + "\"\n"
	report = scanSensitiveDiff(diffFixture(file, literalLine), []string{file})
	if !report.hasFindings() {
		t.Fatal("a literal credential in a workflow should still be reported")
	}
}

func TestPlaceholderRecognitionUsesBoundariesInsteadOfSubstringMatches(t *testing.T) {
	for _, value := range []string{"", "example", "replace_me", "your-api-key-here", "${SERVICE_TOKEN}", "{{TOKEN}}", "${{ secrets.GITHUB_TOKEN }}", "<redacted>"} {
		if !isPlaceholder(value) {
			t.Errorf("%q should be recognized as a placeholder", value)
		}
	}
	for _, value := range []string{"prefix-example-suffix", "secret-example-token-123", "not-a-placeholder"} {
		if isPlaceholder(value) {
			t.Errorf("%q should not be ignored based on a substring", value)
		}
	}
	report := scanSensitiveDiff(diffFixture("settings.env", "+"+fixtureAssignment("SERVICE_TOKEN", "=", "secret-example-token-123")+"\n"), []string{"settings.env"})
	if !report.hasFindings() {
		t.Fatal("a non-placeholder secret value containing 'example' was ignored")
	}
}

func TestFilePathRulesAndIgnoreStatus(t *testing.T) {
	for _, file := range []string{".env", ".env.production", "keys/deploy.pem", ".npmrc"} {
		report := scanSensitiveDiff("", []string{file})
		if !report.hasFindings() {
			t.Errorf("expected path finding for %s", file)
		}
	}
	for _, file := range []string{".env.example", ".env.sample", "README.md"} {
		report := scanSensitiveDiff("", []string{file})
		if report.hasFindings() {
			t.Errorf("unexpected path-only warning for %s: %#v", file, report.findings)
		}
	}

	report := scanSensitiveDiff("", []string{".env"})
	checker := fixedIgnoreChecker{ignored: true}
	annotateIgnoreStatus(&report, checker)
	if report.findings[0].ignoreStatus == "" || !strings.Contains(report.findings[0].ignoreStatus, "staged or tracked") {
		t.Fatalf("ignore annotation = %q", report.findings[0].ignoreStatus)
	}
	annotateIgnoreStatus(&report, fixedIgnoreChecker{})
	if !strings.Contains(report.findings[0].ignoreStatus, "not covered") {
		t.Fatalf("non-ignored annotation = %q", report.findings[0].ignoreStatus)
	}
}

type fixedIgnoreChecker struct{ ignored bool }

func (f fixedIgnoreChecker) IsIgnored(string) (bool, error) { return f.ignored, nil }

func TestDiffParserReportsAddedAndRemovedLineNumbers(t *testing.T) {
	diff := "diff --git a/settings.env b/settings.env\n--- a/settings.env\n+++ b/settings.env\n@@ -10,2 +10,2 @@\n context\n-" + fixtureAssignment("PASSWORD", "=", "synthetic-old-password") + "\n+" + fixtureAssignment("PASSWORD", "=", "synthetic-new-password") + "\n"
	report := scanSensitiveDiff(diff, []string{"settings.env"})
	if len(report.findings) != 2 {
		t.Fatalf("findings = %#v, want removed and added password lines", report.findings)
	}
	if report.findings[0].detail != "removed line 11 · PASSWORD" || report.findings[1].detail != "line 11 · PASSWORD" {
		t.Fatalf("line details = %#v", report.findings)
	}
}

func TestSecurityWarningNeverPrintsValuesAndExplainsContinuation(t *testing.T) {
	const sampleValue = "not-a-real-secret"
	report := scanSensitiveDiff(diffFixture("config.env", "+"+fixtureAssignment("SERVICE_TOKEN", "=", sampleValue)+"\n"), []string{"config.env"})
	var output bytes.Buffer
	printSecurityReport(&output, report)
	for _, text := range []string{"Possible credentials or secrets", "Continue once", "Cancel", "full diff", "configured model provider"} {
		if !strings.Contains(output.String(), text) {
			t.Errorf("warning missing %q: %s", text, output.String())
		}
	}
	if strings.Contains(output.String(), sampleValue) {
		t.Fatalf("warning exposed a detected value: %s", output.String())
	}
}

func TestSecurityWarningFallsBackToTextConfirmationWhenFullScreenUIFails(t *testing.T) {
	cli := &CLI{
		reviewSensitive:          func(securityReport) (bool, error) { return false, errors.New("terminal too short") },
		confirmSensitiveFallback: func() (bool, error) { return true, nil },
	}
	if err := cli.handleSecurityReport(securityReport{findings: []securityFinding{{file: "settings.env", category: "Sensitive setting", detail: "line 1 · API_KEY"}}}, false); err != nil {
		t.Fatalf("fallback continue returned error: %v", err)
	}
	cli.confirmSensitiveFallback = func() (bool, error) { return false, nil }
	if err := cli.handleSecurityReport(securityReport{findings: []securityFinding{{file: "settings.env", category: "Sensitive setting", detail: "line 1 · API_KEY"}}}, false); err == nil || !strings.Contains(err.Error(), "cancelled before generation") {
		t.Fatalf("fallback cancel error = %v", err)
	}
}

func TestCommitCancelStopsBeforeGenerationAndContinueUsesNormalActions(t *testing.T) {
	diff := &entities.GitDiff{Content: diffFixture("settings.env", "+"+fixtureAssignment("SERVICE_TOKEN", "=", "synthetic-token-123456")+"\n"), Files: []string{"settings.env"}}
	service := &fakeCommitCommandService{
		preview: &commit_services.GenerateCommitPreviewEntity{Success: true, CommitMessage: "feat: add setting", Model: "test"},
		confirm: &commit_services.ConfirmCommitEntity{Success: true, CommitHash: "abc123"},
	}
	gitRepo := &fakeSecurityGit{stagedDiff: diff}
	cli := &CLI{commitService: service, securityGit: gitRepo, securityScanEnabled: func() (bool, error) { return true, nil }, reviewSensitive: func(securityReport) (bool, error) { return false, nil }}
	if err := cli.handleCommit([]string{"--commit"}); err == nil || !strings.Contains(err.Error(), "cancelled before generation") {
		t.Fatalf("cancel error = %v", err)
	}
	if service.previewCall != 0 || service.confirmCall != 0 {
		t.Fatalf("cancel called generation=%d commit=%d", service.previewCall, service.confirmCall)
	}

	service.previewCall, service.confirmCall = 0, 0
	cli.reviewSensitive = func(report securityReport) (bool, error) {
		if !report.hasFindings() {
			t.Fatal("warning review received no findings")
		}
		return true, nil
	}
	selectorCalled := false
	cli.selectCommitAction = func() (string, error) { selectorCalled = true; return commitOnlyAction, nil }
	if err := cli.handleCommit(nil); err != nil {
		t.Fatalf("continue flow: %v", err)
	}
	if service.previewCall != 1 || service.confirmCall != 1 || !selectorCalled || service.pushCall != 0 {
		t.Fatalf("normal flow: generation=%d commit=%d selector=%v push=%d", service.previewCall, service.confirmCall, selectorCalled, service.pushCall)
	}
}

func TestNonInteractiveSecurityWarningDoesNotBlockGeneration(t *testing.T) {
	diff := &entities.GitDiff{Content: diffFixture("settings.env", "+"+fixtureAssignment("SERVICE_TOKEN", "=", "synthetic-token-123456")+"\n"), Files: []string{"settings.env"}}
	service := &fakeCommitCommandService{preview: &commit_services.GenerateCommitPreviewEntity{Success: true, CommitMessage: "feat: preview", Model: "test"}}
	cli := &CLI{commitService: service, securityGit: &fakeSecurityGit{stagedDiff: diff}}
	cli.securityScanEnabled = func() (bool, error) { return true, nil }
	oldStdin := os.Stdin
	stdin, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	os.Stdin = stdin
	defer func() { os.Stdin = oldStdin; _ = stdin.Close() }()
	if err := cli.handleCommit([]string{"--dry-run"}); err != nil {
		t.Fatalf("non-interactive warning blocked preview: %v", err)
	}
	if service.previewCall != 1 {
		t.Fatalf("generation calls = %d, want 1", service.previewCall)
	}
}

func TestPRWarningCancelAndContinueFlows(t *testing.T) {
	diff := &entities.GitDiff{Content: diffFixture("settings.yaml", "+"+fixtureAssignment("serviceToken", ": ", "synthetic-token-123456")+"\n"), Files: []string{"settings.yaml"}}
	gitRepo := &fakeSecurityGit{prDiff: diff}
	preview := &fakePRPreviewCommandService{response: &pr_services.GeneratePRPreviewResponse{Success: true, PullRequest: &entities.PullRequest{Body: "# Test PR\n\nDescription"}}}
	cli := &CLI{prService: preview, securityGit: gitRepo, securityScanEnabled: func() (bool, error) { return true, nil }, reviewSensitive: func(securityReport) (bool, error) { return false, nil }}
	if err := cli.handlePr(nil); err == nil || !strings.Contains(err.Error(), "cancelled before generation") {
		t.Fatalf("cancel error = %v", err)
	}
	if preview.calls != 0 {
		t.Fatalf("cancel generated %d PR previews", preview.calls)
	}
	cli.reviewSensitive = func(securityReport) (bool, error) { return true, nil }
	if err := cli.handlePr(nil); err != nil {
		t.Fatalf("continue flow: %v", err)
	}
	if preview.calls != 1 {
		t.Fatalf("continue generated %d PR previews, want 1", preview.calls)
	}
	cli.reviewSensitive = nil
	oldStdin := os.Stdin
	stdin, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	os.Stdin = stdin
	defer func() { os.Stdin = oldStdin; _ = stdin.Close() }()
	if err := cli.handlePr(nil); err != nil {
		t.Fatalf("non-interactive PR warning blocked preview: %v", err)
	}
	if preview.calls != 2 {
		t.Fatalf("non-interactive continue generated %d previews, want 2", preview.calls)
	}
}

func TestSecurityScanOffSkipsWarningsForCommitAndPR(t *testing.T) {
	secretDiff := &entities.GitDiff{
		Content: diffFixture("settings.env", "+"+fixtureAssignment("SERVICE_TOKEN", "=", "synthetic-token-123456")+"\n"),
		Files:   []string{"settings.env"},
	}
	warningCalls := 0
	commit := &fakeCommitCommandService{preview: &commit_services.GenerateCommitPreviewEntity{Success: true, CommitMessage: "feat: preview", Model: "test"}}
	commitCLI := &CLI{
		commitService: commit,
		securityGit:   &fakeSecurityGit{stagedDiff: secretDiff},
		reviewSensitive: func(securityReport) (bool, error) {
			warningCalls++
			return false, nil
		},
	}
	if err := commitCLI.handleCommit([]string{"--dry-run"}); err != nil {
		t.Fatalf("commit with default-off scan failed: %v", err)
	}
	if commit.previewCall != 1 || warningCalls != 0 {
		t.Fatalf("commit preview calls=%d, warning calls=%d; want 1, 0", commit.previewCall, warningCalls)
	}

	preview := &fakePRPreviewCommandService{response: &pr_services.GeneratePRPreviewResponse{Success: true, PullRequest: &entities.PullRequest{Body: "# Test PR\n\nDescription"}}}
	prCLI := &CLI{
		prService:   preview,
		securityGit: &fakeSecurityGit{prDiff: secretDiff},
		reviewSensitive: func(securityReport) (bool, error) {
			warningCalls++
			return false, nil
		},
	}
	if err := prCLI.handlePr(nil); err != nil {
		t.Fatalf("PR preview with default-off scan failed: %v", err)
	}
	if preview.calls != 1 || warningCalls != 0 {
		t.Fatalf("PR preview calls=%d, warning calls=%d; want 1, 0", preview.calls, warningCalls)
	}
}

func TestSecurityCommandUpdatesAndReportsPreference(t *testing.T) {
	enabled := false
	cli := &CLI{
		securityScanEnabled: func() (bool, error) { return enabled, nil },
		saveSecurityScanEnabled: func(next bool) error {
			enabled = next
			return nil
		},
	}
	if err := cli.Run([]string{"gmit", "security", "on"}); err != nil || !enabled {
		t.Fatalf("security on: enabled=%v error=%v", enabled, err)
	}
	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	statusErr := cli.Run([]string{"gmit", "security", "status"})
	_ = writer.Close()
	os.Stdout = oldStdout
	statusOutput, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if statusErr != nil || readErr != nil || !strings.Contains(string(statusOutput), "Security scan is ON") {
		t.Fatalf("security status output=%q statusErr=%v readErr=%v", statusOutput, statusErr, readErr)
	}
	if err := cli.Run([]string{"gmit", "security", "off"}); err != nil || enabled {
		t.Fatalf("security off: enabled=%v error=%v", enabled, err)
	}
	if err := cli.Run([]string{"gmit", "security"}); err == nil {
		t.Fatal("security command without action should return usage")
	}
}

func TestSecurityPromptUsesDefaultForBlankAnswer(t *testing.T) {
	oldStdin := os.Stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("\n\n")); err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	os.Stdin = reader
	defer func() {
		os.Stdin = oldStdin
		_ = reader.Close()
	}()

	for _, defaultValue := range []bool{false, true} {
		got, err := promptYesNoDefault("", defaultValue)
		if err != nil || got != defaultValue {
			t.Fatalf("blank answer with default %v = %v, %v", defaultValue, got, err)
		}
	}
}

func TestSecuritySetupPromptUsesCurrentSettingAsDefault(t *testing.T) {
	var gotDefault bool
	cli := &CLI{confirmSecuritySetup: func(current bool) (bool, error) {
		gotDefault = current
		return current, nil
	}}
	for _, current := range []bool{false, true} {
		gotDefault = !current
		oldStdout := os.Stdout
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Stdout = writer
		got, askErr := cli.askSecuritySetup(current)
		_ = writer.Close()
		os.Stdout = oldStdout
		output, readErr := io.ReadAll(reader)
		_ = reader.Close()
		wantStatus := "OFF"
		if current {
			wantStatus = "ON"
		}
		if askErr != nil || readErr != nil || got != current || gotDefault != current || !strings.Contains(string(output), "Current security scan status: "+wantStatus) || !strings.Contains(string(output), "Press Enter to keep it") {
			t.Fatalf("askSecuritySetup(%v) = %v, %v; callback default=%v output=%q readErr=%v", current, got, askErr, gotDefault, output, readErr)
		}
	}
}

func TestSensitiveIdentifierBoundaries(t *testing.T) {
	for _, name := range []string{"apiKey", "APIKey", "GEMINI_KEY", "clientSecret", "refresh_token", "DATABASE_PASSWORD", "sessionId", "authorization", "privateKey", "signingKey", "encryptionKey", "connectionString", "passphrase", "cookie", "sessionTokens"} {
		if !isSensitiveIdentifier(name) {
			t.Errorf("%q should be sensitive", name)
		}
	}
	for _, name := range []string{"monkey", "apiEndpoint", "clientMode", "tokenType", "buildHash", "passwordRequired", "secretMode"} {
		if isSensitiveIdentifier(name) {
			t.Errorf("%q should not be sensitive", name)
		}
	}
}

func TestCuratedCorpusMaintainsAtLeastNinetyPercentPrecision(t *testing.T) {
	token := strings.Join([]string{"sk-or-v1-", "syntheticcredential123456789"}, "")
	googleToken := strings.Join([]string{"AIzaSyB9c_", "syntheticGoogleCredential123456789"}, "")
	cases := []struct {
		name, file, content string
		secret              bool
	}{
		{"dotenv api key", ".env.example", fixtureAssignment("OPENROUTER_API_KEY", "=", "synthetic-key-value-123456"), true},
		{"camel case password", "config.go", fixtureAssignment("const databasePassword", " = ", fixtureQuoted("synthetic-password-value")), true},
		{"JSON client secret", "settings.json", fixtureAssignment(`"client_secret"`, ": ", fixtureQuoted("synthetic-client-secret")), true},
		{"bearer header", "headers.conf", strings.Join([]string{"Authori", "zation:", " Bearer ", token}, ""), true},
		{"provider token shape", "settings.env", fixtureAssignment("providerCredential", "=", googleToken), true},
		{"credential URL", "settings.txt", fixtureAssignment("databaseUrl", "=", strings.Join([]string{"postgres://dbuser", ":synthetic-password@db.example.test/app"}, "")), true},
		{"private key marker", "deploy.txt", strings.Join([]string{"-----BEGIN ", "PRIVATE KEY-----"}, ""), true},
		{"random unclassified assignment", "config.go", fixtureAssignment("opaqueValue", " = ", fixtureQuoted(randomCredentialFixture())), true},
		{"local environment path", ".env", "ordinary=value", true},
		{"api endpoint", "config.env", fixtureAssignment("apiEndpoint", "=", "https://api.example.test"), false},
		{"service mode", "config.env", fixtureAssignment("serviceMode", "=", "production"), false},
		{"placeholder secret", ".env.example", fixtureAssignment("API_KEY", "=", "your_api_key_here"), false},
		{"hash value", "config.go", fixtureAssignment("buildHash", " = ", fixtureQuoted(randomCredentialFixture())), false},
		{"metadata token type", "settings.json", fixtureAssignment("tokenType", ": ", fixtureQuoted("bearer")), false},
		{"ordinary client mode", "settings.env", fixtureAssignment("clientMode", "=", "internal"), false},
		{"repetitive opaque value", "config.go", fixtureAssignment("opaqueValue", " = ", fixtureQuoted("aaaaaaaaaaaaaaaaaaaaaaaa")), false},
	}
	truePositive, falsePositive, falseNegative := 0, 0, 0
	for _, tc := range cases {
		report := scanSensitiveDiff(diffFixture(tc.file, "+"+tc.content+"\n"), []string{tc.file})
		found := report.hasFindings()
		switch {
		case tc.secret && found:
			truePositive++
		case !tc.secret && found:
			falsePositive++
		case tc.secret && !found:
			falseNegative++
		}
	}
	precision := float64(truePositive) / float64(truePositive+falsePositive)
	if precision < 0.90 {
		t.Fatalf("precision %.1f%% below 90%% (TP=%d FP=%d FN=%d)", 100*precision, truePositive, falsePositive, falseNegative)
	}
	t.Logf("curated corpus: precision %.1f%%, recall %.1f%% (TP=%d FP=%d FN=%d)", precision*100, float64(truePositive)/float64(truePositive+falseNegative)*100, truePositive, falsePositive, falseNegative)
}

func TestSecurityWarningScreenHidesValuesAndShowsChoices(t *testing.T) {
	screen := newTestPickerScreen(t, 90, 14)
	defer screen.Fini()
	state := securityWarningState{
		lines:  []string{"Possible credentials or secrets were found in the diff.", "Continuing sends the full diff to your configured model provider."},
		cursor: 1,
	}
	state.render(screen)
	content := screenContent(screen)
	for _, want := range []string{"Sensitive data review", "Continue once", "Cancel", "full diff"} {
		if !strings.Contains(content, want) {
			t.Errorf("security warning screen missing %q: %s", want, content)
		}
	}
	if state.cursor != 1 {
		t.Fatalf("default cursor = %d, want Cancel selected", state.cursor)
	}
}

func TestScannerSourceAndFixturesDoNotTriggerWarnings(t *testing.T) {
	for _, file := range []string{"security_scan.go", "security_scan_test.go"} {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		var diff strings.Builder
		fmt.Fprintf(&diff, "diff --git a/%s b/%s\n+++ b/%s\n@@ -0,0 +1,1 @@\n", file, file, file)
		for _, line := range strings.Split(strings.TrimSuffix(string(content), "\n"), "\n") {
			diff.WriteByte('+')
			diff.WriteString(line)
			diff.WriteByte('\n')
		}
		report := scanSensitiveDiff(diff.String(), []string{file})
		if report.hasFindings() {
			t.Errorf("scanner source/fixtures in %s triggered findings: %#v", file, report.findings)
		}
	}
}

func TestParseCommandsAllowSensitiveOverrideRemainsCompatible(t *testing.T) {
	_, _, _, allowSensitive, err := parseCommitArgs([]string{"--dry-run", "--allow-sensitive"})
	if err != nil || !allowSensitive {
		t.Fatalf("parseCommitArgs compatibility = %v, %v", allowSensitive, err)
	}
	options, err := parsePRArgs([]string{"--allow-sensitive"})
	if err != nil || !options.allowSensitive {
		t.Fatalf("parsePRArgs compatibility = %#v, %v", options, err)
	}
}
