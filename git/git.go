package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GetVersionInfo returns the git version information for a given path.
func GetVersionInfo(path string) (string, string) {
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = path
	branch, err := branchCmd.Output()
	if err != nil {
		branch = []byte("unknown")
	}

	commitCmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	commitCmd.Dir = path
	commit, err := commitCmd.Output()
	if err != nil {
		commit = []byte("unknown")
	}

	return strings.TrimSpace(string(branch)), strings.TrimSpace(string(commit))
}

// RepoStatus represents the state of a git repository
type RepoStatus struct {
	Exists         bool
	IsClean        bool
	HasUncommitted bool
	HasUnpushed    bool
	CurrentBranch  string
	BehindRemote   bool
	AheadOfRemote  bool
	CanSafelyPull  bool
}

// CheckRepoStatus checks the status of a git repository
func CheckRepoStatus(repoPath string) (*RepoStatus, error) {
	status := &RepoStatus{}

	// Check if directory exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		status.Exists = false
		return status, nil
	}
	status.Exists = true

	// Check if it's a git repository
	gitDir := fmt.Sprintf("%s/.git", repoPath)
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("%s exists but is not a git repository", repoPath)
	}

	// Get current branch
	branchCmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	branchOutput, err := branchCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}
	status.CurrentBranch = strings.TrimSpace(string(branchOutput))

	// Check for uncommitted changes
	statusCmd := exec.Command("git", "-C", repoPath, "status", "--porcelain")
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to check git status: %w", err)
	}

	statusStr := strings.TrimSpace(string(statusOutput))
	status.HasUncommitted = len(statusStr) > 0
	status.IsClean = !status.HasUncommitted

	// Fetch remote to get latest info (silently)
	fetchCmd := exec.Command("git", "-C", repoPath, "fetch", "--quiet")
	_ = fetchCmd.Run() // Ignore errors, we'll handle it in the next steps

	// Check if behind remote
	behindCmd := exec.Command("git", "-C", repoPath, "rev-list", "--count", "HEAD..@{u}")
	behindOutput, err := behindCmd.Output()
	if err == nil {
		behindCount := strings.TrimSpace(string(behindOutput))
		status.BehindRemote = behindCount != "0"
	}

	// Check if ahead of remote
	aheadCmd := exec.Command("git", "-C", repoPath, "rev-list", "--count", "@{u}..HEAD")
	aheadOutput, err := aheadCmd.Output()
	if err == nil {
		aheadCount := strings.TrimSpace(string(aheadOutput))
		status.AheadOfRemote = aheadCount != "0"
		status.HasUnpushed = status.AheadOfRemote
	}

	// Can safely pull if: clean working tree, not ahead of remote
	status.CanSafelyPull = status.IsClean && !status.AheadOfRemote

	return status, nil
}

// Clone clones a git repository
func Clone(repoURL, destPath string) error {
	fmt.Printf("  Cloning %s...\n", repoURL)

	cmd := exec.Command("git", "clone", repoURL, destPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// Pull pulls the latest changes from remote
func Pull(repoPath string) error {
	fmt.Printf("  Pulling latest changes...\n")

	cmd := exec.Command("git", "-C", repoPath, "pull", "--ff-only")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull repository: %w", err)
	}

	return nil
}

// SwitchBranch switches to a different branch
func SwitchBranch(repoPath, branch string) error {
	cmd := exec.Command("git", "-C", repoPath, "checkout", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to switch to branch %s: %w", branch, err)
	}

	return nil
}

// CheckoutCommit checks out a specific commit
func CheckoutCommit(repoPath, commit string) error {
	cmd := exec.Command("git", "-C", repoPath, "checkout", commit)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout commit %s: %w", commit, err)
	}

	return nil
}
