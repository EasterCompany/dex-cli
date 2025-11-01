package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/ui"
)

type updateStep struct {
	name   string
	status string // "pending", "running", "success", "error", "skipped"
	message string
	icon   string
}

// Update manages the dex-cli update process
func Update(args []string) error {
	if len(args) == 0 {
		return updateFull()
	}

	switch args[0] {
	case "check":
		return updateCheck()
	case "pull":
		return updatePull()
	case "build":
		return updateBuild()
	case "install":
		return updateInstall()
	default:
		return showUpdateUsage()
	}
}

func showUpdateUsage() error {
	fmt.Println(ui.RenderTitle("DEXTER UPDATE"))
	fmt.Println()

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	fmt.Println(helpStyle.Render("Update dex-cli to the latest version"))
	fmt.Println()

	cmdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Padding(0, 2)
	fmt.Println(cmdStyle.Render("dex update         # Full update (pull, build, install)"))
	fmt.Println(cmdStyle.Render("dex update check   # Check for updates"))
	fmt.Println(cmdStyle.Render("dex update pull    # Pull latest changes"))
	fmt.Println(cmdStyle.Render("dex update build   # Build dex-cli"))
	fmt.Println(cmdStyle.Render("dex update install # Install to ~/Dexter/bin"))

	return nil
}

// updateFull performs complete update: pull, build, install
func updateFull() error {
	fmt.Println(ui.RenderTitle("DEXTER UPDATE"))
	fmt.Println()

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true)
	fmt.Println(infoStyle.Render("Updating dex-cli to latest version..."))
	fmt.Println()

	steps := []updateStep{
		{"Check repository", "pending", "", "○"},
		{"Pull latest changes", "pending", "", "○"},
		{"Build binary", "pending", "", "○"},
		{"Install to ~/Dexter/bin", "pending", "", "○"},
		{"Verify PATH configuration", "pending", "", "○"},
	}

	// Step 1: Check repository
	steps[0].status = "running"
	renderUpdateSteps(steps)

	dexCliPath, err := config.ExpandPath("~/EasterCompany/dex-cli")
	if err != nil {
		steps[0].status = "error"
		steps[0].message = "Failed to expand path"
		steps[0].icon = "✗"
		renderUpdateSteps(steps)
		return err
	}

	if _, err := os.Stat(dexCliPath); os.IsNotExist(err) {
		steps[0].status = "error"
		steps[0].message = "~/EasterCompany/dex-cli not found"
		steps[0].icon = "✗"
		renderUpdateSteps(steps)
		return fmt.Errorf("dex-cli repository not found at %s", dexCliPath)
	}

	steps[0].status = "success"
	steps[0].message = "Repository found"
	steps[0].icon = "✓"
	renderUpdateSteps(steps)

	// Step 2: Pull latest changes
	steps[1].status = "running"
	renderUpdateSteps(steps)

	repoStatus, err := git.CheckRepoStatus(dexCliPath)
	if err != nil {
		steps[1].status = "error"
		steps[1].message = fmt.Sprintf("Failed to check status: %v", err)
		steps[1].icon = "✗"
		renderUpdateSteps(steps)
		return err
	}

	if !repoStatus.IsClean {
		steps[1].status = "skipped"
		steps[1].message = "Uncommitted changes detected"
		steps[1].icon = "⊘"
		renderUpdateSteps(steps)
		return fmt.Errorf("repository has uncommitted changes, please commit or stash them first")
	}

	if repoStatus.AheadOfRemote {
		steps[1].status = "skipped"
		steps[1].message = "Local commits ahead of remote"
		steps[1].icon = "↑"
		renderUpdateSteps(steps)
		return fmt.Errorf("repository has unpushed commits, please push them first")
	}

	if !repoStatus.BehindRemote {
		steps[1].status = "success"
		steps[1].message = "Already up to date"
		steps[1].icon = "✓"
		renderUpdateSteps(steps)
	} else {
		if err := git.Pull(dexCliPath); err != nil {
			steps[1].status = "error"
			steps[1].message = fmt.Sprintf("Pull failed: %v", err)
			steps[1].icon = "✗"
			renderUpdateSteps(steps)
			return err
		}

		steps[1].status = "success"
		steps[1].message = "Pulled latest changes"
		steps[1].icon = "✓"
		renderUpdateSteps(steps)
	}

	// Step 3: Build binary
	steps[2].status = "running"
	renderUpdateSteps(steps)

	if err := buildDexCLI(dexCliPath); err != nil {
		steps[2].status = "error"
		steps[2].message = fmt.Sprintf("Build failed: %v", err)
		steps[2].icon = "✗"
		renderUpdateSteps(steps)
		return err
	}

	steps[2].status = "success"
	steps[2].message = "Binary built successfully"
	steps[2].icon = "✓"
	renderUpdateSteps(steps)

	// Step 4: Install
	steps[3].status = "running"
	renderUpdateSteps(steps)

	if err := installDexCLI(dexCliPath); err != nil {
		steps[3].status = "error"
		steps[3].message = fmt.Sprintf("Install failed: %v", err)
		steps[3].icon = "✗"
		renderUpdateSteps(steps)
		return err
	}

	steps[3].status = "success"
	steps[3].message = "Installed to ~/Dexter/bin/dex"
	steps[3].icon = "✓"
	renderUpdateSteps(steps)

	// Step 5: Verify PATH configuration
	steps[4].status = "running"
	renderUpdateSteps(steps)

	status, err := verifyPathConfiguration()
	if status == "error" {
		steps[4].status = "error"
		steps[4].message = fmt.Sprintf("Failed to verify PATH: %v", err)
		steps[4].icon = "✗"
		renderUpdateSteps(steps)
		return err
	}

	if status == "warning" {
		steps[4].status = "skipped"
		steps[4].message = "PATH not configured. Please add ~/Dexter/bin to your PATH."
		steps[4].icon = "⚠"
		renderUpdateSteps(steps)
	} else {
		steps[4].status = "success"
		steps[4].message = "PATH configuration verified"
		steps[4].icon = "✓"
		renderUpdateSteps(steps)
	}

	// Success summary
	fmt.Println()
	renderUpdateSummary(true)

	return nil
}

func updateCheck() error {
	fmt.Println(ui.RenderTitle("CHECK FOR UPDATES"))
	fmt.Println()

	dexCliPath, err := config.ExpandPath("~/EasterCompany/dex-cli")
	if err != nil {
		return err
	}

	repoStatus, err := git.CheckRepoStatus(dexCliPath)
	if err != nil {
		return fmt.Errorf("failed to check repository status: %w", err)
	}

	if !repoStatus.IsClean {
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
		fmt.Println(warningStyle.Render("⊘ Repository has uncommitted changes"))
		return nil
	}

	if repoStatus.AheadOfRemote {
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
		fmt.Println(warningStyle.Render("↑ Repository has unpushed commits"))
		return nil
	}

	if repoStatus.BehindRemote {
		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("111"))
		fmt.Println(infoStyle.Render("⬇ Updates available!"))
		fmt.Println()
		fmt.Println(infoStyle.Render("Run 'dex update' to install"))
		return nil
	}

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	fmt.Println(successStyle.Render("✓ Already up to date"))
	return nil
}

func updatePull() error {
	fmt.Println(ui.RenderTitle("PULL LATEST CHANGES"))
	fmt.Println()

	dexCliPath, err := config.ExpandPath("~/EasterCompany/dex-cli")
	if err != nil {
		return err
	}

	repoStatus, err := git.CheckRepoStatus(dexCliPath)
	if err != nil {
		return fmt.Errorf("failed to check status: %w", err)
	}

	if !repoStatus.IsClean {
		return fmt.Errorf("repository has uncommitted changes")
	}

	if repoStatus.AheadOfRemote {
		return fmt.Errorf("repository has unpushed commits")
	}

	if !repoStatus.BehindRemote {
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		fmt.Println(successStyle.Render("✓ Already up to date"))
		return nil
	}

	if err := git.Pull(dexCliPath); err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	fmt.Println(successStyle.Render("✓ Pulled latest changes"))
	return nil
}

func updateBuild() error {
	fmt.Println(ui.RenderTitle("BUILD DEX-CLI"))
	fmt.Println()

	dexCliPath, err := config.ExpandPath("~/EasterCompany/dex-cli")
	if err != nil {
		return err
	}

	if err := buildDexCLI(dexCliPath); err != nil {
		return err
	}

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	fmt.Println(successStyle.Render("✓ Binary built successfully"))
	return nil
}

func updateInstall() error {
	fmt.Println(ui.RenderTitle("INSTALL DEX-CLI"))
	fmt.Println()

	dexCliPath, err := config.ExpandPath("~/EasterCompany/dex-cli")
	if err != nil {
		return err
	}

	if err := installDexCLI(dexCliPath); err != nil {
		return err
	}

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	fmt.Println(successStyle.Render("✓ Installed to ~/Dexter/bin/dex"))
	return nil
}

func buildDexCLI(dexCliPath string) error {
	// Get git commit hash
	commitCmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	commitCmd.Dir = dexCliPath
	commit, err := commitCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get git commit: %w", err)
	}

	// Get current date
	date := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	ldflags := fmt.Sprintf("-X main.commit=%s -X main.date=%s",
		strings.TrimSpace(string(commit)), date)

	cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", "dex-cli")
	cmd.Dir = dexCliPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	return nil
}

func installDexCLI(dexCliPath string) error {
	dexterBinPath, err := config.ExpandPath("~/Dexter/bin")
	if err != nil {
		return err
	}

	sourcePath := filepath.Join(dexCliPath, "dex-cli")
	destPath := filepath.Join(dexterBinPath, "dex")
	tempPath := filepath.Join(dexterBinPath, "dex.new")

	// Copy to temporary location first
	sourceFile, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read binary: %w", err)
	}

	if err := os.WriteFile(tempPath, sourceFile, 0755); err != nil {
		return fmt.Errorf("failed to write binary: %w", err)
	}

	// Rename (atomic operation that works even if dest is running)
	if err := os.Rename(tempPath, destPath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to install binary: %w", err)
	}

	return nil
}

func renderUpdateSteps(steps []updateStep) {
	// Clear previous output (move cursor up)
	fmt.Print("\033[s") // Save cursor position

	for i, step := range steps {
		var style lipgloss.Style
		var icon string

		switch step.status {
		case "pending":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
			icon = "○"
		case "running":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("111"))
			icon = "⋯"
		case "success":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
			icon = step.icon
		case "error":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			icon = step.icon
		case "skipped":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
			icon = step.icon
		}

		iconStyle := lipgloss.NewStyle().
			Foreground(style.GetForeground()).
			Bold(true)

		nameStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Width(30).
			Padding(0, 1)

		message := step.message
		if step.status == "running" {
			message = "..."
		}

		fmt.Printf("%s %s %s\n",
			iconStyle.Render(icon),
			nameStyle.Render(step.name),
			style.Render(message))

		// If this is the last step being rendered, don't print extra newline
		if i == len(steps)-1 {
			fmt.Print("\033[u") // Restore cursor position for next update
		}
	}

	// Add delay for visibility
	time.Sleep(500 * time.Millisecond)
}

func renderUpdateSummary(success bool) {
	summaryBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("42")).
		Padding(1, 2).
		Width(50)

	var summaryText strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("42"))

	summaryText.WriteString(titleStyle.Render("✓ UPDATE COMPLETE"))
	summaryText.WriteString("\n\n")

	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	summaryText.WriteString(infoStyle.Render("dex-cli has been updated successfully!"))
	summaryText.WriteString("\n\n")

	versionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("111")).
		Italic(true)
	summaryText.WriteString(versionStyle.Render("Run 'dex version' to see the new version"))

	fmt.Println(summaryBox.Render(summaryText.String()))
}

func verifyPathConfiguration() (string, error) {
	// Get the user's home directory.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "error", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Construct the full path to ~/Dexter/bin.
	dexterBinPath := filepath.Join(homeDir, "Dexter", "bin")

	// Get the PATH environment variable.
	path := os.Getenv("PATH")

	// Split the PATH into a slice of directories.
	paths := strings.Split(path, string(os.PathListSeparator))

	// Check if the ~/Dexter/bin path is already in the PATH.
	for _, p := range paths {
		if p == dexterBinPath {
			return "success", nil
		}
	}

	// If it's not in the PATH, return a message with instructions.
	return "warning", fmt.Errorf("directory %s not in PATH", dexterBinPath)
}
