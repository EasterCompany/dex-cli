package cmd

// Lint lints the code for all services
func Lint(args []string) error {
	return runOnAllServices("golangci-lint", []string{"run", "./..."}, "LINTING", true)
}
