package cli

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

type securityFinding struct {
	file, category, detail string
	ignoreStatus           string
}

type securityReport struct{ findings []securityFinding }

var (
	// assignmentPattern handles dotenv, JSON/YAML/TOML-like key/value pairs,
	// and common quoted assignments in source code without depending on a
	// provider-specific environment variable name.
	assignmentPattern = regexp.MustCompile(`(?i)(?:^|[\s,{.\[])['"]?([a-z_][a-z0-9_.-]{0,80})['"]?(?:\s*:\s*[a-z_][a-z0-9_<>,\[\] |?]*)?\s*(?::=|=|:)\s*(?:"((?:\\.|[^"\\])*)"|'((?:\\.|[^'\\])*)'|` + "`([^`]*)`" + `|([^\s"',;]+))`)
	knownTokenPattern = regexp.MustCompile(`(?i)\b(?:sk-or-v1-[a-z0-9_-]{12,}|xai-[a-z0-9_-]{12,}|sk-[a-z0-9_-]{20,}|AIza[0-9A-Za-z_-]{30,}|xox[baprs]-[0-9A-Za-z-]{10,}|gh[pousr]_[a-z0-9]{20,}|github_pat_[a-z0-9_]{20,}|AKIA[0-9A-Z]{16}|eyJ[a-zA-Z0-9_-]{8,}\.[a-zA-Z0-9_-]{8,}\.[a-zA-Z0-9_-]{8,})\b`)
	privateKeyPattern = regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY-----`)
	credentialURL     = regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.-]*://[^\s/:@]+:[^\s/@]+@`)
	authHeader        = regexp.MustCompile(`(?i)\b(?:authorization|proxy-authorization)\s*[:=]\s*(?:bearer|basic)\s+[^\s,;]+`)
	hunkPattern       = regexp.MustCompile(`^@@ -([0-9]+)(?:,[0-9]+)? \+([0-9]+)(?:,[0-9]+)? @@`)
	placeholderExact  = regexp.MustCompile(`(?i)^(?:your(?:[-_ ](?:api[-_ ]?)?(?:key|secret|token|password))?|replace[-_ ]?me|change[-_ ]?me|example|sample|placeholder|dummy|redacted|none|null|todo|x{4,})(?:[-_ ](?:value|here|goes-here))?$`)
	placeholderEnv    = regexp.MustCompile(`^\$\{[^}]+\}$`)
	placeholderMust   = regexp.MustCompile(`^\{\{[^}]+\}\}$`)
	placeholderAngle  = regexp.MustCompile(`^<[^>]+>$`)
)

var benignEntropyNameTokens = map[string]bool{
	"checksum": true, "commit": true, "digest": true, "etag": true,
	"guid": true, "hash": true, "id": true, "revision": true,
	"uuid": true, "version": true,
}

func scanSensitiveDiff(diff string, files []string) securityReport {
	var report securityReport
	seen := map[string]bool{}
	add := func(f securityFinding) {
		key := f.file + "\x00" + f.category + "\x00" + f.detail
		if !seen[key] {
			seen[key] = true
			report.findings = append(report.findings, f)
		}
	}
	for _, file := range files {
		base := strings.ToLower(filepath.Base(filepath.ToSlash(file)))
		if isLocalEnvironmentFile(base) {
			add(securityFinding{file: file, category: "Local environment file", detail: "this file commonly contains local credentials"})
		}
		if isPrivateCredentialFile(file) {
			add(securityFinding{file: file, category: "Credential file", detail: "this path commonly contains private credentials"})
		}
	}

	currentFile := "changed file"
	oldLine, newLine := 0, 0
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			parts := strings.SplitN(line, " b/", 2)
			if len(parts) == 2 {
				currentFile = parts[1]
			}
			oldLine, newLine = 0, 0
			continue
		}
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			continue
		}
		if strings.HasPrefix(line, "@@") {
			if positions := hunkPattern.FindStringSubmatch(line); len(positions) == 3 {
				fmt.Sscanf(positions[1], "%d", &oldLine)
				fmt.Sscanf(positions[2], "%d", &newLine)
			}
			continue
		}
		if len(line) == 0 {
			continue
		}

		marker := line[0]
		if marker != '+' && marker != '-' && marker != ' ' {
			continue
		}
		content := line[1:]
		lineNo := newLine
		if marker == '-' {
			lineNo = oldLine
			oldLine++
		} else if marker == '+' {
			newLine++
		} else {
			oldLine++
			newLine++
			continue
		}
		if lineNo < 1 {
			lineNo = 1
		}
		lineDetail := fmt.Sprintf("line %d", lineNo)
		if marker == '-' {
			lineDetail = fmt.Sprintf("removed line %d", lineNo)
		}

		for range knownTokenPattern.FindAllString(content, -1) {
			add(securityFinding{file: currentFile, category: "Recognized credential format", detail: lineDetail})
		}
		if privateKeyPattern.MatchString(content) {
			add(securityFinding{file: currentFile, category: "Private key material", detail: lineDetail})
		}
		if credentialURL.MatchString(content) {
			add(securityFinding{file: currentFile, category: "Credential in URL", detail: lineDetail})
		}
		if authHeader.MatchString(content) {
			add(securityFinding{file: currentFile, category: "Authentication header", detail: lineDetail})
		}

		for _, match := range assignmentPattern.FindAllStringSubmatch(content, -1) {
			if len(match) < 6 {
				continue
			}
			name := match[1]
			value := firstNonEmpty(match[2], match[3], match[4], match[5])
			if isSourceFile(currentFile) && match[2] == "" && match[3] == "" && match[4] == "" {
				continue
			}
			if value == "" || isPlaceholder(value) {
				continue
			}
			if isSensitiveIdentifier(name) {
				add(securityFinding{file: currentFile, category: "Sensitive setting", detail: fmt.Sprintf("%s · %s", lineDetail, name)})
				continue
			}
			if hasSecretLikeEntropy(value) && !hasBenignEntropyName(name) {
				add(securityFinding{file: currentFile, category: "Unusual high-entropy value", detail: fmt.Sprintf("%s · %s resembles a random credential", lineDetail, name)})
			}
		}
	}
	return report
}

type gitIgnoreChecker interface {
	IsIgnored(path string) (bool, error)
}

func annotateIgnoreStatus(report *securityReport, repository any) {
	checker, ok := repository.(gitIgnoreChecker)
	if report == nil || !ok {
		return
	}
	for i := range report.findings {
		finding := &report.findings[i]
		if finding.category != "Local environment file" && finding.category != "Credential file" {
			continue
		}
		ignored, err := checker.IsIgnored(finding.file)
		if err != nil {
			finding.ignoreStatus = "Git ignore status unavailable"
		} else if ignored {
			finding.ignoreStatus = "matches .gitignore, but this file is staged or tracked"
		} else {
			finding.ignoreStatus = "not covered by .gitignore"
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func isLocalEnvironmentFile(base string) bool {
	if base == ".env" {
		return true
	}
	if !strings.HasPrefix(base, ".env.") {
		return false
	}
	for _, example := range []string{"example", "sample", "template"} {
		if strings.Contains(base, example) {
			return false
		}
	}
	return true
}

func isPrivateCredentialFile(file string) bool {
	base := strings.ToLower(filepath.Base(filepath.ToSlash(file)))
	ext := strings.ToLower(filepath.Ext(base))
	path := strings.ToLower(filepath.ToSlash(file))
	return ext == ".pem" || ext == ".key" || ext == ".p12" || ext == ".pfx" || ext == ".jks" || ext == ".keystore" ||
		base == "id_rsa" || base == "id_ed25519" || base == "credentials.json" || base == ".npmrc" || base == ".pypirc" ||
		strings.HasSuffix(path, "/.docker/config.json")
}

func isSensitiveIdentifier(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	// Split separators and camel case so apiKey, API_KEY, clientSecret and
	// refresh-token share the same rules without provider-specific names.
	tokens := identifierTokens(name)
	if len(tokens) > 1 {
		switch tokens[len(tokens)-1] {
		case "type", "mode", "name", "kind", "enabled", "required", "configured", "length", "count", "format", "algorithm", "expires", "url":
			return false
		}
	}
	contains := func(want string) bool {
		for _, token := range tokens {
			if token == want {
				return true
			}
		}
		return false
	}
	for _, token := range []string{"password", "passwords", "passwd", "passphrase", "pwd", "secret", "secrets", "token", "tokens", "credential", "credentials", "authorization", "bearer", "cookie", "cookies"} {
		if contains(token) {
			return true
		}
	}
	if contains("key") || contains("keys") {
		return true
	}
	for _, pair := range [][2]string{{"api", "key"}, {"auth", "token"}, {"auth", "key"}, {"client", "secret"}, {"private", "key"}, {"access", "key"}, {"access", "token"}, {"refresh", "token"}, {"signing", "key"}, {"encryption", "key"}, {"session", "token"}, {"session", "id"}, {"connection", "string"}} {
		if contains(pair[0]) && contains(pair[1]) {
			return true
		}
	}
	return false
}

func hasBenignEntropyName(name string) bool {
	for _, token := range identifierTokens(name) {
		if benignEntropyNameTokens[strings.ToLower(token)] {
			return true
		}
	}
	return false
}

func identifierTokens(name string) []string {
	runes := []rune(name)
	tokens := make([]string, 0, 4)
	current := make([]rune, 0, len(runes))
	flush := func() {
		if len(current) > 0 {
			tokens = append(tokens, strings.ToLower(string(current)))
			current = current[:0]
		}
	}
	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		if len(current) > 0 {
			previous := current[len(current)-1]
			nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			boundary := unicode.IsUpper(r) && (unicode.IsLower(previous) || unicode.IsDigit(previous) || (unicode.IsUpper(previous) && nextIsLower))
			if boundary {
				flush()
			}
		}
		current = append(current, r)
	}
	flush()
	return tokens
}

func hasSecretLikeEntropy(value string) bool {
	value = strings.TrimSpace(value)
	if len([]rune(value)) < 24 {
		return false
	}
	if strings.IndexFunc(value, func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '\r' }) >= 0 {
		return false
	}
	classes := 0
	var lower, upper, digit, symbol bool
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= 'A' && r <= 'Z':
			upper = true
		case r >= '0' && r <= '9':
			digit = true
		default:
			symbol = true
		}
	}
	for _, present := range []bool{lower, upper, digit, symbol} {
		if present {
			classes++
		}
	}
	if classes < 3 {
		return false
	}
	return shannonEntropy(value) >= 3.5
}

func isSourceFile(file string) bool {
	switch strings.ToLower(filepath.Ext(file)) {
	case ".go", ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs", ".py", ".java", ".kt", ".rs", ".c", ".h", ".cc", ".cpp", ".cs", ".php", ".rb", ".swift", ".sh":
		return true
	default:
		return false
	}
}

func shannonEntropy(value string) float64 {
	if value == "" {
		return 0
	}
	counts := make(map[rune]int)
	length := 0
	for _, r := range value {
		counts[r]++
		length++
	}
	var entropy float64
	for _, count := range counts {
		p := float64(count) / float64(length)
		entropy -= p * math.Log2(p)
	}
	return entropy
}

func isPlaceholder(value string) bool {
	v := strings.TrimSpace(strings.Trim(value, `"'`))
	if v == "" || placeholderEnv.MatchString(v) || placeholderMust.MatchString(v) || placeholderAngle.MatchString(v) {
		return true
	}
	return placeholderExact.MatchString(v)
}

func (r securityReport) hasFindings() bool { return len(r.findings) > 0 }
