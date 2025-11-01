package cmd

// Format formats and lints the code for all services
func Format(args []string) error {
	return runOnAllServices("gofmt", []string{"-w", "."}, "FORMATTING & LINTING", true)
}
