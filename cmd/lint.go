package cmd

// Lint lints the code for all services
func Lint(args []string) error {
	return runOnAllServices("golint", []string{"-set_exit_status", "./..."}, "LINTING", true)
}
