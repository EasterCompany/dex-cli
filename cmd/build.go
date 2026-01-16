package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/EasterCompany/dex-cli/cache"
	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/release"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

var RunningVersion string

// waitForActiveProcesses checks Redis for 'process:info:*' keys and waits until they are gone.
func waitForActiveProcesses(ctx context.Context) error {
	redisClient, err := cache.GetLocalClient(ctx)
	if err != nil {
		// If Redis is unreachable, we can't check, so we continue but warn.
		ui.PrintWarning("Could not connect to Redis to check for active processes. Continuing...")
		return nil
	}
	defer func() { _ = redisClient.Close() }()

	spinFrame := 0
	for {
		// Register in queue (heartbeat style)
		queueInfo := map[string]interface{}{
			"channel_id": "system-cli-op",
			"state":      "Waiting...",
			"start_time": time.Now().Unix(),
			"pid":        os.Getpid(),
			"updated_at": time.Now().Unix(),
		}
		qBytes, _ := json.Marshal(queueInfo)
		_ = redisClient.Set(ctx, "process:queued:system-cli-op", qBytes, 15*time.Second).Err()

		// Check for active process keys
		keys, err := redisClient.Keys(ctx, "process:info:*").Result()
		if err != nil {
			return fmt.Errorf("failed to query active processes: %w", err)
		}

		// Filter out our own build process
		var activeKeys []string
		for _, k := range keys {
			// We ignore our own build op
			if strings.HasSuffix(k, ":system-cli-op") {
				continue
			}

			// We DO NOT ignore system-guardian, because that is a heavy resource-consuming task
			if strings.HasSuffix(k, ":system-guardian") {
				activeKeys = append(activeKeys, k)
				continue
			}

			activeKeys = append(activeKeys, k)
		}
		if len(activeKeys) == 0 {
			ui.ClearLine()
			_ = redisClient.Del(ctx, "process:queued:system-cli-op").Err()
			return nil
		}

		// Show waiting status
		label := fmt.Sprintf("Waiting for %d active process(es) to finish...", len(activeKeys))
		ui.PrintSpinner(label, spinFrame)
		spinFrame++

		select {
		case <-ctx.Done():
			_ = redisClient.Del(ctx, "process:queued:system-cli-op").Err()
			return ctx.Err()
		case <-time.After(1 * time.Second):
			// Check again
		}
	}
}

// getServiceVersion gets the current version for a service by checking multiple sources
// and returning the highest valid version found.
func getServiceVersion(def config.ServiceDefinition) (major, minor, patch int, err error) {
	var versions []*git.Version

	// 1. Try to read from data.json (The Truth for distributed releases)
	feDef := config.GetServiceDefinition("easter-company")
	feSource, err := config.ExpandPath(feDef.Source)
	if err == nil {
		dataPath := filepath.Join(feSource, "bin", "data.json")
		rd, err := release.LoadReleaseData(dataPath)
		if err == nil && rd != nil {
			if svcInfo, ok := rd.Services[def.ID]; ok && svcInfo.Current != "" {
				v, err := git.Parse(svcInfo.Current)
				if err == nil {
					versions = append(versions, v)
				}
			}
		}
	}

	// 2. Try Git Tags (The Truth for source control)
	sourcePath, err := config.ExpandPath(def.Source)
	if err == nil {
		if tag, err := git.GetLatestTag(sourcePath); err == nil {
			if v, err := git.Parse(tag); err == nil {
				versions = append(versions, v)
			}
		}
	}

	// 3. Try Installed Binary (The Truth for the current machine)
	binVersionStr := utils.GetBinaryVersion(def)
	if binVersionStr != "" && binVersionStr != "N/A" && binVersionStr != "unknown" {
		if v, err := git.Parse(binVersionStr); err == nil {
			versions = append(versions, v)
		}
	}

	// If no versions found, default to 0.0.0
	if len(versions) == 0 {
		return 0, 0, 0, nil
	}

	// Find the highest version
	highest := versions[0]
	for i := 1; i < len(versions); i++ {
		if versions[i].Compare(highest) > 0 {
			highest = versions[i]
		}
	}

	major, _ = strconv.Atoi(highest.Major)
	minor, _ = strconv.Atoi(highest.Minor)
	patch, _ = strconv.Atoi(highest.Patch)
	return major, minor, patch, nil
}

// hasUncommittedChanges checks if a service has uncommitted changes
func hasUncommittedChanges(def config.ServiceDefinition) bool {
	sourcePath, err := config.ExpandPath(def.Source)
	if err != nil {
		return false
	}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = sourcePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) != ""
}

// getHighestMajorMinor returns the highest major.minor version across all buildable services
func getHighestMajorMinor(services []config.ServiceDefinition) (int, int, error) {
	maxMajor := 0
	maxMinor := 0

	for _, s := range services {
		if !s.IsBuildable() {
			continue
		}

		major, minor, _, err := getServiceVersion(s)
		if err != nil {
			return 0, 0, err
		}

		if major > maxMajor || (major == maxMajor && minor > maxMinor) {
			maxMajor = major
			maxMinor = minor
		}
	}

	return maxMajor, maxMinor, nil
}

// verifyDeveloperAccess checks if this is a developer environment with source code access
func verifyDeveloperAccess() error {
	dexCliDir := fmt.Sprintf("%s/EasterCompany/dex-cli", os.Getenv("HOME"))
	if _, err := os.Stat(dexCliDir); os.IsNotExist(err) {
		return fmt.Errorf("build command is only available for Easter Company developers with source code access")
	}
	return nil
}

// buildFrontendService executes the build.sh script for a frontend service like easter.company
func buildFrontendService(ctx context.Context, def config.ServiceDefinition, log func(message string), major, minor, patch int) (bool, error) {
	sourcePath, err := config.ExpandPath(def.Source)
	if err != nil {
		return false, fmt.Errorf("failed to expand source path for %s: %w", def.ShortName, err)
	}

	buildScriptPath := filepath.Join(sourcePath, "scripts", "build.sh")

	// 0. Install Dependencies (Bun)
	if _, err := os.Stat(filepath.Join(sourcePath, "package.json")); err == nil {
		log(fmt.Sprintf("[%s] Installing dependencies with Bun...", def.ShortName))
		installCmd := exec.CommandContext(ctx, "bun", "install")
		installCmd.Dir = sourcePath
		if out, err := installCmd.CombinedOutput(); err != nil {
			return false, fmt.Errorf("bun install failed: %w\n%s", err, string(out))
		}
	}

	// 0.2. Type Check (TypeScript)
	if _, err := os.Stat(filepath.Join(sourcePath, "tsconfig.json")); err == nil {
		log(fmt.Sprintf("[%s] Checking types with TypeScript...", def.ShortName))
		tscCmd := exec.CommandContext(ctx, "bun", "run", "tsc", "--noEmit")
		tscCmd.Dir = sourcePath
		if out, err := tscCmd.CombinedOutput(); err != nil {
			return false, fmt.Errorf("typescript check failed: %w\n%s", err, string(out))
		}
	}

	// 0. Format Code (Prettier)
	if _, err := exec.LookPath("prettier"); err == nil {
		log(fmt.Sprintf("[%s] Formatting source code with Prettier...", def.ShortName))
		// We format the source directory (where JS/CSS/HTML lives)
		fmtCmd := exec.CommandContext(ctx, "prettier", "--write", "source")
		fmtCmd.Dir = sourcePath
		if out, err := fmtCmd.CombinedOutput(); err != nil {
			log(fmt.Sprintf("[%s] Warning: Prettier failed: %v\n%s", def.ShortName, err, string(out)))
			// We warn but proceed, or should we fail?
			// The user requested strict tooling. Failing on format error (if it's a syntax error that prettier can't parse) is good.
			// If it's just "I formatted it", it returns 0.
			// So error here means Prettier couldn't parse the code.
			// Let's fail the build if formatting fails, as that usually indicates syntax errors.
			return false, fmt.Errorf("prettier formatting failed (syntax error?): %w\n%s", err, string(out))
		}
	} else {
		log(fmt.Sprintf("[%s] Warning: 'prettier' not found, skipping formatting.", def.ShortName))
	}

	// 0.5. Lint Code
	log(fmt.Sprintf("[%s] Linting source code...", def.ShortName))
	lintFailed := false

	// ESLint
	if _, err := exec.LookPath("eslint"); err == nil {
		// Run on project directory (includes config files)
		lintCmd := exec.CommandContext(ctx, "eslint", ".")
		lintCmd.Dir = sourcePath
		if out, err := lintCmd.CombinedOutput(); err != nil {
			log(fmt.Sprintf("[%s] ESLint failed: %v\n%s", def.ShortName, err, string(out)))
			lintFailed = true
		}
	}

	// Stylelint
	if _, err := exec.LookPath("stylelint"); err == nil {
		lintCmd := exec.CommandContext(ctx, "stylelint", "source/**/*.css")
		lintCmd.Dir = sourcePath
		if out, err := lintCmd.CombinedOutput(); err != nil {
			log(fmt.Sprintf("[%s] Stylelint failed: %v\n%s", def.ShortName, err, string(out)))
			lintFailed = true
		}
	}

	// HTMLHint
	if _, err := exec.LookPath("htmlhint"); err == nil {
		lintCmd := exec.CommandContext(ctx, "htmlhint", "source/**/*.html")
		lintCmd.Dir = sourcePath
		if out, err := lintCmd.CombinedOutput(); err != nil {
			log(fmt.Sprintf("[%s] HTMLHint failed: %v\n%s", def.ShortName, err, string(out)))
			lintFailed = true
		}
	}

	if lintFailed {
		return false, fmt.Errorf("linting failed, check logs for details")
	}

	// 0.8. Run Tests (Vitest)
	vitestConfig := filepath.Join(sourcePath, "vitest.config.js")
	if _, err := os.Stat(vitestConfig); err == nil {
		log(fmt.Sprintf("[%s] Running tests with Vitest...", def.ShortName))
		testCmd := exec.CommandContext(ctx, "bun", "run", "vitest", "run")
		testCmd.Dir = sourcePath
		if out, err := testCmd.CombinedOutput(); err != nil {
			log(fmt.Sprintf("[%s] Tests failed: %v\n%s", def.ShortName, err, string(out)))
			return false, fmt.Errorf("tests failed")
		}
		log(fmt.Sprintf("[%s] Tests passed!", def.ShortName))
	}

	// Construct full version string for frontend
	branch, commit := git.GetVersionInfo(sourcePath)
	buildDate := time.Now().Format("2006-01-02")
	arch := runtime.GOARCH
	shortVersionStr := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	fullVersionStr := fmt.Sprintf("%s.%s.%s.%s.%s", shortVersionStr, branch, commit, buildDate, arch)

	log(fmt.Sprintf("[%s] Running frontend build script: %s (Version: %s)", def.ShortName, buildScriptPath, fullVersionStr))

	cmd := exec.CommandContext(ctx, "bash", buildScriptPath)
	cmd.Dir = sourcePath // Execute the script from the service's source directory
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("DEX_BUILD_VERSION=%s", fullVersionStr))

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to build frontend service %s: %w", def.ShortName, err)
	}

	return true, nil
}

// verifyGitHubAccess performs a one-time check to verify GitHub access to EasterCompany org
func verifyGitHubAccess() error {
	cacheFile := fmt.Sprintf("%s/.cache/dex-cli/github-access-verified", os.Getenv("HOME"))

	// Check if already verified
	if _, err := os.Stat(cacheFile); err == nil {
		return nil
	}

	ui.PrintInfo("Verifying GitHub access to EasterCompany organization...")

	// Try SSH first (preferred for devs)
	sshCmd := exec.Command("ssh", "-T", "git@github.com")
	sshOutput, sshErr := sshCmd.CombinedOutput()

	hasSSH := sshErr == nil || strings.Contains(string(sshOutput), "successfully authenticated")

	// Try HTTPS as fallback
	httpsCmd := exec.Command("git", "ls-remote", "--exit-code", "https://github.com/EasterCompany/dex-cli.git", "HEAD")
	httpsErr := httpsCmd.Run()

	if !hasSSH && httpsErr != nil {
		return fmt.Errorf("cannot access GitHub EasterCompany organization - check your SSH keys or network connection")
	}

	// Cache the verification
	_ = os.MkdirAll(fmt.Sprintf("%s/.cache/dex-cli", os.Getenv("HOME")), 0755)
	if err := os.WriteFile(cacheFile, []byte(time.Now().Format(time.RFC3339)), 0644); err != nil {
		// Non-fatal, just skip caching
		ui.PrintWarning("Could not cache GitHub access verification")
	}

	if hasSSH {
		ui.PrintSuccess("GitHub SSH access verified")
	} else {
		ui.PrintSuccess("GitHub HTTPS access verified")
	}

	return nil
}

func Build(args []string) error {
	startTime := time.Now()

	// Setup signal handling for graceful cleanup on Ctrl+C
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Wipe Redis to ensure a clean state
	if err := utils.WipeRedis(ctx); err != nil {
		ui.PrintWarning(fmt.Sprintf("Failed to wipe Redis: %v", err))
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Emit initial status to register process
	utils.SendEvent("system.cli.status", map[string]interface{}{
		"status":  "online",
		"message": "build in progress...",
	})

	// EMIT START EVENT
	utils.SendEvent("system.cli.command", map[string]interface{}{
		"command":    "build",
		"args":       args,
		"state":      "started",
		"start_time": startTime.Format(time.RFC3339),
	})

	// Run build in a goroutine so we can listen for signals
	errChan := make(chan error, 1)
	go func() {
		errChan <- runBuild(ctx, args)
	}()

	var err error
	select {
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal %v, cancelling build...\n", sig)
		cancel()
		err = fmt.Errorf("build cancelled by user")
	case err = <-errChan:
		// Build finished naturally
	}

	duration := time.Since(startTime)
	status := "success"
	if err != nil {
		status = "failure"
	}

	// EMIT END EVENT
	utils.SendEvent("system.cli.command", map[string]interface{}{
		"command":  "build",
		"args":     args,
		"state":    "completed",
		"status":   status,
		"duration": duration.String(),
		"error":    fmt.Sprintf("%v", err),
	})

	// Emit final status to clear process (EventTypeCLIStatus)
	utils.SendEvent("system.cli.status", map[string]interface{}{
		"status":  status,
		"message": fmt.Sprintf("build %s", status),
	})

	return err
}

func runBuild(ctx context.Context, args []string) error {
	// Check for --source flag to build from source via go run
	buildFromSource := false
	var filteredArgsSource []string
	for _, arg := range args {
		if arg == "--source" {
			buildFromSource = true
		} else {
			filteredArgsSource = append(filteredArgsSource, arg)
		}
	}

	if buildFromSource {
		ui.PrintInfo("Building from source via 'go run'...")

		// Get dex-cli source path
		cliService := config.GetServiceDefinition("dex-cli")
		sourcePath, err := config.ExpandPath(cliService.Source)
		if err != nil {
			return fmt.Errorf("failed to expand dex-cli source path: %w", err)
		}

		// Construct command: go run main.go build [args...] (without --source)
		cmdArgs := append([]string{"run", "main.go", "build"}, filteredArgsSource...)
		cmd := exec.CommandContext(ctx, "go", cmdArgs...)
		cmd.Dir = sourcePath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		// Pass through environment
		cmd.Env = os.Environ()

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("build from source failed: %w", err)
		}
		return nil
	}

	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			ui.PrintHeader("Build Command Help")
			ui.PrintInfo("Usage: dex build [major|minor|patch] [-f|--force]")
			fmt.Println()
			ui.PrintInfo("Arguments:")
			ui.PrintInfo("  major, minor, patch   Increment the version number accordingly.")
			ui.PrintInfo("                        If omitted, defaults to 'patch' for modified services only.")
			fmt.Println()
			ui.PrintInfo("Flags:")
			ui.PrintInfo("  -f, --force           Force rebuild of all services even if no changes are detected.")
			fmt.Println()
			ui.PrintInfo("Description:")
			ui.PrintInfo("  Builds and installs CLI and services from source.")
			ui.PrintInfo("  This command requires developer access to the source code.")
			return nil
		}
	}

	// Verify this is a developer environment

	if err := verifyDeveloperAccess(); err != nil {

		return err

	}

	// Verify GitHub access (one-time check)

	if err := verifyGitHubAccess(); err != nil {

		return err

	}

	logFile, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	log := func(message string) {
		_, _ = fmt.Fprintln(logFile, message)
	}

	// Check for --force flag
	forceRebuild := false
	var filteredArgs []string
	for _, arg := range args {
		if arg == "--force" || arg == "-f" {
			forceRebuild = true
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}
	args = filteredArgs

	// Check for active processes before starting build (unless forced)
	if !forceRebuild {
		if err := waitForActiveProcesses(ctx); err != nil {
			return err
		}
	}

	// Validate arguments
	if len(args) > 1 {
		return fmt.Errorf("build command accepts at most 1 argument (major, minor, or patch)")
	}

	var requestedIncrement string
	if len(args) == 1 {
		requestedIncrement = args[0]
		if requestedIncrement != "major" && requestedIncrement != "minor" && requestedIncrement != "patch" {
			return fmt.Errorf("invalid argument '%s': must be 'major', 'minor', or 'patch'", requestedIncrement)
		}
	} else {
		requestedIncrement = "auto"
	}

	log("Build command called...")
	ui.PrintHeader("Building All Services from Local Source")
	allServices := config.GetAllServices()

	// ---
	// THE LAW OF VERSION: Determine versioning strategy
	// ---
	var servicesWithChanges []config.ServiceDefinition
	for _, s := range allServices {
		if s.IsBuildable() && hasUncommittedChanges(s) {
			servicesWithChanges = append(servicesWithChanges, s)
		}
	}

	var incrementType string
	var buildAllServices bool
	var targetMajorAll, targetMinorAll, targetPatchAll int

	switch requestedIncrement {
	case "major":
		// LAW 3: Major increment - force ALL services to same major version
		ui.PrintInfo("Major release: incrementing ALL services to same major version")
		highestMajor, _, err := getHighestMajorMinor(allServices)
		if err != nil {
			return err
		}
		targetMajorAll = highestMajor + 1
		targetMinorAll = 0
		targetPatchAll = 0
		incrementType = "major"
		buildAllServices = true

	case "minor":
		// LAW 2: Minor increment - force ALL services to same minor version
		ui.PrintInfo("Minor release: incrementing ALL services to same minor version")
		highestMajor, highestMinor, err := getHighestMajorMinor(allServices)
		if err != nil {
			return err
		}
		targetMajorAll = highestMajor
		targetMinorAll = highestMinor + 1
		targetPatchAll = 0
		incrementType = "minor"
		buildAllServices = true

	case "patch", "auto":
		// If force rebuild is specified, build ALL services
		if forceRebuild {
			ui.PrintInfo("Force rebuild: building all services")
			servicesWithChanges = []config.ServiceDefinition{}
			for _, s := range allServices {
				if s.IsBuildable() {
					servicesWithChanges = append(servicesWithChanges, s)
				}
			}
		} else if len(servicesWithChanges) == 0 {
			ui.PrintWarning("No uncommitted changes detected in any service")
			return nil
		}

		if len(servicesWithChanges) == 1 {
			ui.PrintInfo(fmt.Sprintf("Building %s with patch increment", servicesWithChanges[0].ShortName))
		} else {
			ui.PrintInfo(fmt.Sprintf("Building %d services with individual patch increments", len(servicesWithChanges)))
		}

		incrementType = "patch"
		buildAllServices = false
	}

	// ---
	// 1. Capture "before" state: Get versions and sizes by executing binaries
	// ---
	ui.PrintInfo("Capturing current versions...")
	oldVersions := make(map[string]string)
	oldSizes := make(map[string]int64)
	for _, s := range allServices {
		if s.IsBuildable() {
			// Single source of truth: execute binary with 'version' argument
			oldVersions[s.ID] = utils.GetBinaryVersion(s)
			oldSizes[s.ID] = utils.GetBinarySize(s)
		}
	}

	// ---
	// 2. Build services based on versioning strategy
	// ---
	type buildTask struct {
		service                               config.ServiceDefinition
		targetMajor, targetMinor, targetPatch int
	}

	var buildTasks []buildTask

	// Determine which services to build and their target versions
	for _, s := range allServices {
		if !s.IsBuildable() {
			continue
		}

		// Check if source code exists
		sourcePath, err := config.ExpandPath(s.Source)
		if err != nil || sourcePath == "" {
			continue
		}
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			continue
		}

		// Skip if not building all services and this service has no changes
		if !buildAllServices {
			hasChanges := false
			for _, changed := range servicesWithChanges {
				if changed.ID == s.ID {
					hasChanges = true
					break
				}
			}
			if !hasChanges {
				continue
			}
		}

		var targetMajor, targetMinor, targetPatch int

		if buildAllServices {
			// Use the shared version for all services
			targetMajor = targetMajorAll
			targetMinor = targetMinorAll
			targetPatch = targetPatchAll
		} else {
			// Individual patch increment
			baseMajor, baseMinor, basePatch, err := getServiceVersion(s)
			if err != nil {
				return fmt.Errorf("failed to get version for %s: %w", s.ShortName, err)
			}
			targetMajor = baseMajor
			targetMinor = baseMinor
			targetPatch = basePatch + 1
		}

		buildTasks = append(buildTasks, buildTask{
			service:     s,
			targetMajor: targetMajor,
			targetMinor: targetMinor,
			targetPatch: targetPatch,
		})
	}

	// ---
	// 2. Build Phase: Build each service
	// ---
	ui.PrintHeader("Build Phase")
	var builtServices []config.ServiceDefinition

	for i, task := range buildTasks {
		// Check for cancellation between tasks
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if i > 0 {
			fmt.Println()
		}

		s := task.service
		ui.PrintInfo(fmt.Sprintf("%s%s%s", ui.ColorCyan, fmt.Sprintf("# Building %s", s.ShortName), ui.ColorReset))

		baseMajor, baseMinor, basePatch, err := getServiceVersion(s)
		if err != nil {
			return fmt.Errorf("failed to get version for %s: %w", s.ShortName, err)
		}

		ui.PrintInfo(fmt.Sprintf("Incrementing version: %d.%d.%d -> %d.%d.%d (%s)",
			baseMajor, baseMinor, basePatch, task.targetMajor, task.targetMinor, task.targetPatch, incrementType))

		var built bool
		var buildErr error
		serviceStartTime := time.Now()
		if s.Type == "fe" { // Check if it's a frontend service
			built, buildErr = buildFrontendService(ctx, s, log, task.targetMajor, task.targetMinor, task.targetPatch)
		} else {
			built, buildErr = utils.RunUnifiedBuildPipeline(ctx, s, log, task.targetMajor, task.targetMinor, task.targetPatch)
		}

		serviceDuration := time.Since(serviceStartTime)

		if buildErr != nil {
			// EMIT NOTIFICATION ON FAILURE
			utils.SendEvent("system.notification.generated", map[string]interface{}{
				"title":    fmt.Sprintf("Build Failed: %s", s.ShortName),
				"priority": "critical",
				"category": "build",
				"body":     fmt.Sprintf("Build failure in service '%s'. Error: %v", s.ShortName, buildErr),
			})
			return buildErr
		}

		if built {
			builtServices = append(builtServices, s)
			ui.PrintSuccess(fmt.Sprintf("Successfully built %s!", s.ShortName))

			// EMIT EVENT: system.build.completed
			utils.SendEvent("system.build.completed", map[string]interface{}{
				"service_name": s.ShortName,
				"version":      fmt.Sprintf("%d.%d.%d", task.targetMajor, task.targetMinor, task.targetPatch),
				"duration":     serviceDuration.String(),
				"status":       "success",
			})
		}
	}

	// ---
	// 3. Install Phase: Install each built service
	// ---
	if len(builtServices) > 0 {
		fmt.Println()
		ui.PrintHeader("Install Phase")

		for _, s := range builtServices {
			// Frontend services don't have a systemd service to install
			if s.SystemdName == "" {
				ui.PrintInfo(fmt.Sprintf("Skipping systemd installation for frontend service %s", s.ShortName))
				continue
			}
			if err := utils.InstallSystemdService(s); err != nil {
				return err
			}
			ui.PrintSuccess(fmt.Sprintf("Successfully installed %s!", s.ShortName))
		}
	}

	// ---
	// 4. Capture "after" state: Get versions by executing newly-built binaries
	// ---
	fmt.Println()
	ui.PrintInfo("Capturing new versions...")
	newVersions := make(map[string]string)
	newSizes := make(map[string]int64)
	for _, s := range builtServices {
		// Single source of truth: execute newly-built binary with 'version' argument
		newVersions[s.ID] = utils.GetBinaryVersion(s)
		newSizes[s.ID] = utils.GetBinarySize(s)
	}

	// ---
	// 5. Git Phase: ALL git operations for ALL built services
	// ---
	if len(builtServices) > 0 {
		fmt.Println()
		ui.PrintHeader("Git Phase")

		for _, task := range buildTasks {
			// Only do git operations for services that were actually built
			wasBuilt := false
			for _, built := range builtServices {
				if built.ID == task.service.ID {
					wasBuilt = true
					break
				}
			}
			if !wasBuilt {
				continue
			}

			if err := gitAddCommitPush(task.service, incrementType, task.targetMajor, task.targetMinor, task.targetPatch); err != nil {
				return err
			}
		}
	}

	// ---
	// 6. Publish to easter.company (major, minor, AND patch)
	// ---
	if incrementType == "major" || incrementType == "minor" || incrementType == "patch" {
		fmt.Println()
		ui.PrintHeader("Publish Phase")

		// Get short version (e.g., "2.1.0")
		shortVersion := fmt.Sprintf("%d.%d.%d", targetMajorAll, targetMinorAll, targetPatchAll)

		// LAW: If shortVersion is 0.0.0 (individual patch builds), find the highest version among built services
		if shortVersion == "0.0.0" {
			for _, s := range builtServices {
				v := newVersions[s.ID]
				if v != "" && v != "N/A" && v != "unknown" {
					// We take the first non-empty version from built services as representative
					// Since we captured newVersions right before this, it should be accurate.
					// We only need the major.minor.patch part.
					parts := strings.Split(v, ".")
					if len(parts) >= 3 {
						shortVersion = strings.Join(parts[:3], ".")
						break
					}
				}
			}
		}

		// Publish binaries and update data.json
		fullVersion := ""
		for _, s := range builtServices {
			if s.ShortName == "cli" {
				fullVersion = newVersions[s.ID]
				break
			}
		}
		if fullVersion == "" && len(builtServices) > 0 {
			fullVersion = newVersions[builtServices[0].ID]
		}

		if err := release.PublishRelease(fullVersion, shortVersion, incrementType, builtServices); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to publish release: %v", err))
			ui.PrintWarning("Binaries are built and committed, but not published to easter.company")
		} else {
			ui.PrintSuccess(fmt.Sprintf("Release %s published to https://easter.company", shortVersion))
		}
	}

	// ---
	// 7. Summary
	// ---
	fmt.Println()
	ui.PrintHeader("Summary")
	time.Sleep(1 * time.Second)

	var summaryData []utils.SummaryInfo
	for _, s := range allServices {
		if s.IsBuildable() {
			oldVersionStr := oldVersions[s.ID]
			newVersionStr := oldVersionStr // Default to old version if not built
			oldSize := oldSizes[s.ID]
			newSize := oldSize // Default to old size if not built

			// Check if this service was built
			wasBuilt := false
			for _, built := range builtServices {
				if built.ID == s.ID {
					wasBuilt = true
					newVersionStr = newVersions[s.ID]
					newSize = newSizes[s.ID]
					break
				}
			}

			// Get the latest commit message from the repository
			var commitNote string
			if wasBuilt {
				repoPath, err := config.ExpandPath(s.Source)
				if err == nil {
					_, latestCommit := git.GetVersionInfo(repoPath)
					if latestCommit != "" && latestCommit != "unknown" {
						commitNote, _ = git.GetCommitMessage(repoPath, latestCommit)
					} else {
						commitNote = "N/A"
					}
				} else {
					commitNote = "N/A"
				}
			} else {
				commitNote = "N/A"
			}

			summaryData = append(summaryData, utils.SummaryInfo{
				Service:       s,
				OldVersion:    oldVersionStr,
				NewVersion:    newVersionStr,
				OldSize:       oldSize,
				NewSize:       newSize,
				ChangeSummary: commitNote,
			})
		}
	}

	utils.PrintSummaryTable(summaryData)
	fmt.Println()

	// ---
	// 8. Ollama Model Sync (runs after all builds and installs are complete)
	// ---
	// This uses the *newly-built* dex-cli binary to ensure its own models are in sync.
	if len(builtServices) > 0 {
		fmt.Println()
		ui.PrintHeader("Ollama Model Sync")
		ui.PrintInfo("Ensuring custom Ollama models are up-to-date...")

		dexCliPath := fmt.Sprintf("%s/Dexter/bin/dex", os.Getenv("HOME"))
		if _, err := os.Stat(dexCliPath); os.IsNotExist(err) {
			ui.PrintWarning("Newly built dex-cli not found, skipping model sync.")
		} else {
			// Run with --no-event to prevent recursion
			modelSyncCmd := exec.Command(dexCliPath, "ollama", "pull", "--no-event")
			modelSyncCmd.Stdout = os.Stdout
			modelSyncCmd.Stderr = os.Stderr
			if err := modelSyncCmd.Run(); err != nil {
				ui.PrintWarning(fmt.Sprintf("Ollama model sync failed: %v", err))
				ui.PrintInfo("This may be because the Ollama service is not running. You can run 'dex ollama pull' manually later.")
			} else {
				ui.PrintSuccess("Ollama models are up-to-date.")
			}
		}
	}

	ui.PrintSuccess("Build complete.")

	// ---
	// 9. Run release script if version increment was requested (post-build actions)
	// ---
	if incrementType != "" && len(builtServices) > 0 {
		fmt.Println()
		ui.PrintHeader("Post-Build Actions")
		releaseScript := fmt.Sprintf("%s/EasterCompany/easter.company/scripts/release_dex-cli.sh", os.Getenv("HOME"))

		// Check if release script exists
		if _, err := os.Stat(releaseScript); err == nil {
			ui.PrintInfo("Running release script...")
			releaseCmd := exec.Command(releaseScript)
			releaseCmd.Stdout = os.Stdout
			releaseCmd.Stderr = os.Stderr
			if err := releaseCmd.Run(); err != nil {
				ui.PrintWarning(fmt.Sprintf("Release script failed: %v", err))
				ui.PrintInfo("Version tags have been created and pushed successfully.")
			} else {
				ui.PrintSuccess("Release published successfully!")
			}
		} else {
			ui.PrintInfo("No release script found, skipping publish step.")
			ui.PrintInfo("Version tags have been created and pushed successfully.")
		}
	}

	return nil
}

func gitAddCommitPush(def config.ServiceDefinition, incrementType string, major, minor, patch int) error {
	sourcePath, err := config.ExpandPath(def.Source)
	if err != nil {
		return fmt.Errorf("failed to expand source path: %w", err)
	}

	ui.PrintInfo(fmt.Sprintf("[%s] Adding, committing, and pushing changes...", def.ShortName))

	// Add all changes
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = sourcePath
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed for %s:\n%s", def.ShortName, string(output))
	}

	// Check if there are changes to commit
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = sourcePath
	statusOutput, err := statusCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git status failed for %s:\n%s", def.ShortName, string(statusOutput))
	}

	// If no changes, skip commit and push
	if strings.TrimSpace(string(statusOutput)) == "" {
		ui.PrintInfo(fmt.Sprintf("[%s] No changes to commit", def.ShortName))
		return nil
	}

	// Get the diff for commit message generation
	diffCmd := exec.Command("git", "diff", "--cached")
	diffCmd.Dir = sourcePath
	diffOutput, err := diffCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git diff failed for %s:\n%s", def.ShortName, string(diffOutput))
	}

	// Generate commit message using the Ollama model
	commitMsg := utils.GenerateCommitMessage(string(diffOutput))

	// Commit with generated message
	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	commitCmd.Dir = sourcePath
	if output, err := commitCmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(output), "nothing to commit") {
			return fmt.Errorf("git commit failed for %s:\n%s", def.ShortName, string(output))
		}
	}

	// Push changes
	pushCmd := exec.Command("git", "push")
	pushCmd.Dir = sourcePath
	if output, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed for %s:\n%s", def.ShortName, string(output))
	}

	// Create and push tag for ALL builds (to track version history)
	tagName := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	ui.PrintInfo(fmt.Sprintf("[%s] Creating tag %s...", def.ShortName, tagName))

	// Create tag
	tagCmd := exec.Command("git", "tag", tagName)
	tagCmd.Dir = sourcePath
	if output, err := tagCmd.CombinedOutput(); err != nil {
		// If tag already exists, that's okay
		if !strings.Contains(string(output), "already exists") {
			return fmt.Errorf("git tag failed for %s:\n%s", def.ShortName, string(output))
		}
		ui.PrintWarning(fmt.Sprintf("[%s] Tag %s already exists", def.ShortName, tagName))
	} else {
		// Push tag
		pushTagCmd := exec.Command("git", "push", "--tags")
		pushTagCmd.Dir = sourcePath
		if output, err := pushTagCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git push --tags failed for %s:\n%s", def.ShortName, string(output))
		}

		ui.PrintSuccess(fmt.Sprintf("[%s] Tag %s created and pushed", def.ShortName, tagName))
	}

	return nil
}
