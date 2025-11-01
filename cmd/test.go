package cmd

// Test runs the test suite for all services
func Test(args []string) error {
	return runOnAllServices("go", []string{"test", "-v", "./..."}, "RUNNING TESTS", true)
}
