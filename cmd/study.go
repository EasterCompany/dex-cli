package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Study manages architectural research papers
func Study(args []string) error {
	if len(args) == 0 {
		ui.PrintHeader("Study Command Help")
		ui.PrintInfo("Usage: dex study [command] [args...]")
		fmt.Println()
		ui.PrintInfo("Commands:")
		ui.PrintInfo("  add <title>  Scaffold a new research paper")
		ui.PrintInfo("  list         List all existing research papers")
		return nil
	}

	switch args[0] {
	case "add":
		if len(args) < 2 {
			return fmt.Errorf("usage: dex study add \"Paper Title\"")
		}
		return studyAdd(args[1:])
	case "list":
		return studyList()
	default:
		return fmt.Errorf("unknown study subcommand: %s", args[0])
	}
}

func studyAdd(args []string) error {
	title := args[0]
	slug := strings.ToLower(strings.ReplaceAll(title, " ", "-"))
	// Remove special chars from slug
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, slug)

	// Get paths
	feDef := config.GetServiceDefinition("easter-company")
	feSource, err := config.ExpandPath(feDef.Source)
	if err != nil {
		return err
	}

	studiesDir := filepath.Join(feSource, "static", "docs", "studies")
	filePath := filepath.Join(studiesDir, slug+".md")

	// Check if already exists
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("study already exists: %s", slug)
	}

	// Create content
	dateStr := time.Now().Format("02 January 2006")
	content := fmt.Sprintf(`# %s

**Date:** %s  
**Authors:** Dexter Fabricator (Autonomous Protocol), Owen Easter (Easter Company)  
**Classification:** Architectural Study • AI Engineering

---

## Abstract
[Enter a high-fidelity summary of the research findings here. This text will be used for SEO and the preview card on the archive page.]

## 1. Introduction
[Introduction to the problem and the proposed solution.]

## 2. Methodology
[Technical details of the implementation or experiment.]

## 3. Analysis
[Data, results, and architectural implications.]

## 4. Conclusion
[Summary of outcomes and next steps.]

---
**References:**
- Dexter System Architecture (v7.1.4)
`, title, dateStr)

	// Write file
	if err := os.MkdirAll(studiesDir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return err
	}

	ui.PrintSuccess(fmt.Sprintf("Scaffolded new study: %s", slug))
	ui.PrintInfo(fmt.Sprintf("Path: %s", filePath))
	ui.PrintInfo("Run 'dex build' after editing to update the website.")

	return nil
}

func studyList() error {
	feDef := config.GetServiceDefinition("easter-company")
	feSource, err := config.ExpandPath(feDef.Source)
	if err != nil {
		return err
	}

	studiesDir := filepath.Join(feSource, "static", "docs", "studies")
	files, err := os.ReadDir(studiesDir)
	if err != nil {
		return err
	}

	ui.PrintHeader("ARCHITECTURAL STUDIES ARCHIVE")
	table := ui.NewTable([]string{"Slug", "Format"})
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
			table.AddRow([]string{strings.TrimSuffix(f.Name(), ".md"), "Markdown"})
		}
	}
	table.Render()

	return nil
}
