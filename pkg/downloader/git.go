package downloader

import (
	"log/slog"
	"os/exec"
	"strings"
)

// TODO: Maybe use go-git. But it doesn't have git archive...
func gitArchive(remote string, ref string, destination string) error {
	cmd := exec.Command("git", "archive", "--remote", remote, ref, "-o", destination)
	slog.Info("Executing", slog.String("command", cmd.String()))
	if _, err := cmd.Output(); err != nil {
		return err
	}
	return nil
}

func gitLsRemote(remote string, ref string) (string, error) {
	cmd := exec.Command("git", "ls-remote", remote, ref)
	slog.Info("Executing", slog.String("command", cmd.String()))
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	refData := strings.Split(wsRe.ReplaceAllString(string(out), " "), " ")
	return refData[0], nil
}
