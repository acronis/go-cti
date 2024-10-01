package gitstorage

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
)

var (
	wsRe       = regexp.MustCompile(`\s+`)
	goImportRe = regexp.MustCompile("<meta name=\"go-import\" content=\"([^\"]+)")
)

// TODO: Maybe use go-git. But it doesn't have git archive...
func gitArchive(remote string, ref string, destination string) error {
	cmd := exec.Command("git", "archive", "--remote", remote, ref, "-o", destination)
	slog.Info("Executing", slog.String("command", cmd.String()))
	if _, err := cmd.Output(); err != nil {
		return fmt.Errorf("git archive: %w", err)
	}
	return nil
}
func gitLsRemote(remote string, ref string) (string, error) {
	cmd := exec.Command("git", "ls-remote", remote, ref)
	slog.Info("Executing", slog.String("command", cmd.String()))
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git ls-remote: %w", err)
	}
	refData := strings.Split(wsRe.ReplaceAllString(string(out), " "), " ")
	return refData[0], nil
}

func parseGoQuery(goQuery string) (string, string, string) {
	parts := strings.Split(goQuery, " ")
	return parts[0], parts[1], parts[2]
}

func discoverSource(source string) ([]byte, error) {
	// TODO: Better dependency path handling
	// Reuse the same resolution mechanism that go mod uses
	// https://go.dev/ref/mod#vcs-find
	url, err := url.Parse(source)
	if err != nil {
		return nil, err
	}
	query := url.Query()
	query.Add("go-get", "1")

	resp, err := http.Get(url.String() + "?" + query.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
