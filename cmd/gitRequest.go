package cmd

import (
	"github.com/spf13/cobra"
	"github.com/zricethezav/gitleaks/v8/flags"
)

var (
	gitRequestCmd = &cobra.Command{
		Use:   "git-request",
		Short: "Detect in Merge-Request or Pull-Request",
		Run:   runDetectGitRequest,
	}
)

func init() {
	detectCmd.AddCommand(gitRequestCmd)
	gitRequestFlags := flags.GetGitRequestFlags()
	gitRequestCmd.Flags().StringVar(&gitRequestFlags.SourceBranch, "source-branch", "", "Source Branch")
	gitRequestCmd.Flags().StringVar(&gitRequestFlags.TargetBranch, "target-branch", "", "Target Branch")
}

func runDetectGitRequest(cmd *cobra.Command, args []string) {
	runDetect(cmd, args)
}
