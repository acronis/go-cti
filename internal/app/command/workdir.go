package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	workingDirFlag = "working-dir"
)

func AddWorkDirFlag(cmd *cobra.Command) {
	cwd, _ := os.Getwd()

	cmd.PersistentFlags().StringP(workingDirFlag, "w", cwd, "define working directory")
}

func GetWorkingDir(cmd *cobra.Command) (string, error) {
	baseDir, err := cmd.Flags().GetString(workingDirFlag)
	if err != nil {
		return "", fmt.Errorf("get base-dir flag: %w", err)
	}
	return baseDir, nil
}
