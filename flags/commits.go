package flags

var commitsFlags = &CommitsFlags{}

type CommitsFlags struct {
	File string
}

func GetCommitsFlags() *CommitsFlags {
	return commitsFlags
}
