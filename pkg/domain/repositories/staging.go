package repositories

// FileStatus describes the index and worktree state of a repository-relative path.
type FileStatus struct {
	Path, OriginalPath string
	Index, Worktree    byte
}

// StagingRepository exposes the Git index operations used by the file selector.
type StagingRepository interface {
	ListChanges() ([]FileStatus, error)
	StageFiles([]string) error
	StageAll() error
	UnstageFile(FileStatus) error
}
