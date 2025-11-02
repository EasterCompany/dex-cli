package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Model manages the dex-cli model command
func Model(args []string) error {
	logFile, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	log := func(message string) {
		_, _ = fmt.Fprintln(logFile, message)
	}

	log(fmt.Sprintf("Model command called with args: %v", args))

	if len(args) == 0 {
		return showModelUsage(log)
	}

	switch args[0] {
	case "list":
		return listModels(log)
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("model name required for 'delete' command")
		}
		return deleteModel(args[1], log)
	default:
		return showModelUsage(log)
	}
}

func showModelUsage(log func(string)) error {
	log("Displaying model command usage.")
	fmt.Println("Manage models in ~/Dexter/models")
	fmt.Println()
	fmt.Println("  dex model list        # List all models")
	fmt.Println("  dex model delete <name> # Delete a model")
	return nil
}

func listModels(log func(string)) error {
	log("Listing models.")
	modelsPath, err := config.ExpandPath(filepath.Join(config.DexterRoot, "models"))
	if err != nil {
		return fmt.Errorf("failed to expand models path: %w", err)
	}

	files, err := os.ReadDir(modelsPath)
	if err != nil {
		return fmt.Errorf("failed to read models directory: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No models found.")
		log("No models found.")
		return nil
	}

	table := ui.NewTable([]string{"Model Name"})
	for _, file := range files {
		table.AddRow([]string{file.Name()})
		log(fmt.Sprintf("Found model: %s", file.Name()))
	}
	table.Render()

	return nil
}

func deleteModel(modelName string, log func(string)) error {
	log(fmt.Sprintf("Attempting to delete model: %s", modelName))
	modelsPath, err := config.ExpandPath(filepath.Join(config.DexterRoot, "models"))
	if err != nil {
		return fmt.Errorf("failed to expand models path: %w", err)
	}

	modelPath := filepath.Join(modelsPath, modelName)

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		log(fmt.Sprintf("Model %s not found at %s", modelName, modelPath))
		return fmt.Errorf("model %s not found", modelName)
	}

	if err := os.Remove(modelPath); err != nil {
		log(fmt.Sprintf("Failed to delete model %s: %v", modelName, err))
		return fmt.Errorf("failed to delete model %s: %w", modelName, err)
	}

	fmt.Printf("Model %s deleted successfully\n", modelName)
	log(fmt.Sprintf("Model %s deleted successfully", modelName))

	return nil
}
