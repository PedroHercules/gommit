# Contributing to Gmit

Thanks for helping improve Gmit. Issues and pull requests are reviewed by the maintainer; submitting one does not guarantee that it will be accepted or merged.

## Before opening an issue

- Search existing issues to avoid duplicates.
- For bugs, include the Gmit version, operating system and architecture, steps to reproduce, expected behavior, and actual behavior. Remove API keys, private diffs, and other sensitive data from logs.
- For feature requests or new provider integrations, describe the problem and intended behavior first. Please wait for maintainer feedback before investing in a substantial implementation.
- Small documentation fixes can be submitted directly as a pull request.
- Do not report security vulnerabilities in public issues. Follow [SECURITY.md](SECURITY.md).

## Pull requests

- Do not push directly to `main`. Create a branch and open a pull request against `main`.
- Keep each pull request focused. Link the related issue when one exists and explain the user-visible change.
- Add or update tests when behavior changes. Run `gofmt` on changed Go files and `go test ./...` before submitting.
- Use Conventional Commit titles such as `feat: add provider support` or `fix: handle empty responses`; semantic-release uses these messages to generate version numbers and GitHub release notes.
- Never include credentials, API keys, personal data, or private repository contents in code, tests, screenshots, or logs.
- Resolve review conversations and update the pull request when requested. The required checks and maintainer/code-owner approval must pass before merge.

The maintainer makes the final decision on scope, design, prioritization, and merging. Changes may be declined when they are duplicates, out of scope, insufficiently explained, unsafe, or inconsistent with the project direction. Please keep discussion constructive and respectful.

## Other ways to help

You can improve documentation, report reproducible bugs, suggest features, or review open discussions. Start with an issue for changes that affect behavior or project direction; no code contribution is required to help.
