package flags

var gitRequestFlags = &GitRequestFlags{}

type GitRequestFlags struct {
	SourceBranch string
	TargetBranch string
}

func GetGitRequestFlags() *GitRequestFlags {
	return gitRequestFlags
}
