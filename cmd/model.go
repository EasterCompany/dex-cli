package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/charmbracelet/lipgloss"
)

// Model manages the dex-cli model command
func Model(args []string) error {
	if len(args) == 0 {
		return showModelUsage()
	}

	switch args[0] {
	case "list":
		return listModels()
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("model name required for 'delete' command")
		}
		return deleteModel(args[1])
	default:
		return showModelUsage()
	}
}

func showModelUsage() error {
	fmt.Println("Manage models in ~/Dexter/models")
	fmt.Println()

	cmdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Padding(0, 2)
	fmt.Println(cmdStyle.Render("dex model list        # List all models"))
	fmt.Println(cmdStyle.Render("dex model delete <name> # Delete a model"))

	return nil
}

func listModels() error {
	modelsPath, err := config.ExpandPath(filepath.Join(config.DexterRoot, "models"))
	if err != nil {
		return fmt.Errorf("failed to expand models path: %w", err)
	}

	files, err := os.ReadDir(modelsPath)
	if err != nil {
		return fmt.Errorf("failed to read models directory: %w", err)
	}

	for _, file := range files {
		fmt.Println(file.Name())
	}

	return nil
}

func deleteModel(modelName string) error {
	modelsPath, err := config.ExpandPath(filepath.Join(config.DexterRoot, "models"))
	if err != nil {
		return fmt.Errorf("failed to expand models path: %w", err)
	}

	modelPath := filepath.Join(modelsPath, modelName)

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("model %s not found", modelName)
	}

	if err := os.Remove(modelPath); err != nil {
		return fmt.Errorf("failed to delete model %s: %w", modelName, err)
	}

	ui.PrintInfo(fmt.Sprintf("Model %s deleted successfully", modelName))

	return nil
}
