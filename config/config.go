package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/EasterCompany/dex-cli/ui"
)

const (
	DexterRoot        = "~/Dexter"
	DexterBin         = "~/Dexter/bin"
	DexterLogs        = "~/Dexter/logs"
	EasterCompanyRoot = "~/EasterCompany"
)

var RequiredDexterDirs = []string{
	"bin",
	"config",
	"data",
	"logs",
	"models",
	"run",
}

// GetDexterPath returns the absolute path to the ~/Dexter directory.
func GetDexterPath() (string, error) {
	return ExpandPath(DexterRoot)
}

// ExpandPath expands ~ to the user's home directory
func ExpandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	if path == "~" {
		return homeDir, nil
	}

	return filepath.Join(homeDir, path[2:]), nil
}

// EnsureDirectoryStructure creates required directories if they don't exist
func EnsureDirectoryStructure() error {
	// Ensure ~/Dexter exists
	dexterPath, err := ExpandPath(DexterRoot)
	if err != nil {
		return fmt.Errorf("failed to expand Dexter root path: %w", err)
	}

	if err := os.MkdirAll(dexterPath, 0o755); err != nil {
		return fmt.Errorf("failed to create Dexter directory: %w", err)
	}

	// Ensure all required subdirectories exist
	for _, dir := range RequiredDexterDirs {
		dirPath := filepath.Join(dexterPath, dir)
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", dir, err)
		}
	}

	// Ensure linter/formatter configs exist in ~/EasterCompany (ONLY if directory exists)
	if err := EnsureLinterConfigs(); err != nil {
		return fmt.Errorf("failed to ensure linter configs: %w", err)
	}

	// Ensure log files exist for all services (including Upstash/OS)
	if err := EnsureServiceLogFiles(); err != nil {
		return fmt.Errorf("failed to ensure service log files: %w", err)
	}

	return nil
}

// EnsureLinterConfigs produces/overwrites standard config files in ~/EasterCompany
func EnsureLinterConfigs() error {
	root, err := ExpandPath(EasterCompanyRoot)
	if err != nil {
		return err
	}

	// Only proceed if the directory actually exists (prevent unwanted creation in user mode)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	}

	configs := map[string]string{
		".htmlhintrc": `{
  "tagname-lowercase": true,
  "attr-lowercase": true,
  "attr-value-double-quotes": true,
  "doctype-first": false,
  "tag-pair": true,
  "spec-char-escape": true,
  "id-unique": true,
  "src-not-empty": true,
  "attr-no-duplication": true,
  "title-require": true
}
`,
		".prettierrc": `{
  "tabWidth": 2,
  "useTabs": false,
  "semi": true,
  "singleQuote": true,
  "trailingComma": "es5",
  "printWidth": 100,
  "bracketSpacing": true,
  "arrowParens": "always",
  "endOfLine": "lf"
}
`,
		".stylelintrc.json": `{
  "rules": {
    "color-no-invalid-hex": true,
    "block-no-empty": true,
    "unit-no-unknown": true,
    "property-no-unknown": [
      true,
      {
        "ignoreProperties": ["text-fill-color"]
      }
    ],
    "selector-pseudo-class-no-unknown": true
  }
}
`,
		"eslint.config.mjs": `export default [
  {
    ignores: [
      '**/dist/',
      '**/bin/',
      '**/node_modules/',
      '**/oceaster.github.io/',
      '**/*.min.js',
      'easter.company/static/',
      '**/*.ts',
    ],
  },
  {
    files: ['**/*.js'],
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: 'module',
      globals: {
        browser: true,
        node: true,
        es2021: true,
      },
    },
    rules: {
      'no-unused-vars': 'warn',
      'no-undef': 'warn',
    },
  },
];
`,
	}

	for filename, content := range configs {
		path := filepath.Join(root, filename)
		// ALWAYS overwrite to ensure latest standards are enforced
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	return nil
}

// EnsureServiceLogFiles creates empty log files for all services if they don't exist
func EnsureServiceLogFiles() error {
	for _, def := range serviceDefinitions {
		logPath, err := ExpandPath(def.GetLogPath())
		if err != nil {
			continue
		}

		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			// Create an empty log file
			f, err := os.Create(logPath)
			if err != nil {
				continue
			}
			_ = f.Close()
		}
	}
	return nil
}

// EnsureConfigFiles creates and validates all config files.
func EnsureConfigFiles() error {
	// Service Map
	userMap, err := LoadServiceMapConfig()
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Creating default service-map.json...")
			if err := SaveServiceMapConfig(DefaultServiceMapConfig()); err != nil {
				return fmt.Errorf("failed to save default service-map.json: %w", err)
			}
		} else {
			return fmt.Errorf("failed to load service-map.json: %w", err)
		}
	} else {
		// File exists, check if it needs healing (missing new services)
		if healed := healServiceMapConfig(userMap, DefaultServiceMapConfig()); healed {
			fmt.Println("Healing service-map.json: Added missing services.")
			if err := SaveServiceMapConfig(userMap); err != nil {
				return fmt.Errorf("failed to save healed service-map.json: %w", err)
			}
		}
	}

	// Options - with healing
	userOpts, err := LoadOptionsConfig()
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, create a new one with all defaults
			fmt.Println("Creating default options.json...")
			if err := SaveOptionsConfig(DefaultOptionsConfig()); err != nil {
				return fmt.Errorf("failed to save default options.json: %w", err)
			}
		} else {
			// Other error loading the file
			return fmt.Errorf("failed to load options.json: %w", err)
		}
	} else {
		// File exists, check if it needs healing
		if healed := healOptionsConfig(userOpts, DefaultOptionsConfig()); healed {
			fmt.Println("Healing options.json: Added missing default values.")
			if err := SaveOptionsConfig(userOpts); err != nil {
				return fmt.Errorf("failed to save healed options.json: %w", err)
			}
		}
	}

	// Server Map
	_, serverMapErr := LoadServerMapConfig()
	if serverMapErr != nil {
		if os.IsNotExist(serverMapErr) {
			fmt.Println("Creating default server-map.json...")
			if err := SaveServerMapConfig(DefaultServerMapConfig()); err != nil {
				return fmt.Errorf("failed to save default server-map.json: %w", err)
			}
		} else {
			return fmt.Errorf("failed to load server-map.json: %w", serverMapErr)
		}
	}

	return nil
}

// healServiceMapConfig adds missing services from default to user config.
// It modifies userMap directly. Returns true if changes were made.
func healServiceMapConfig(userMap *ServiceMapConfig, defaultMap *ServiceMapConfig) bool {
	healed := false
	if userMap.Services == nil {
		userMap.Services = make(map[string][]ServiceEntry)
	}

	for category, defaultServices := range defaultMap.Services {
		if _, exists := userMap.Services[category]; !exists {
			userMap.Services[category] = defaultServices
			healed = true
			continue
		}

		// Check for missing individual services within the category
		for _, defSvc := range defaultServices {
			found := false
			for _, userSvc := range userMap.Services[category] {
				if userSvc.ID == defSvc.ID {
					found = true
					break
				}
			}
			if !found {
				userMap.Services[category] = append(userMap.Services[category], defSvc)
				healed = true
			}
		}
	}
	return healed
}

// healOptionsConfig merges the default config into the user's config to add missing fields.
// It modifies the userOpts object directly. Returns true if changes were made.
func healOptionsConfig(userOpts *OptionsConfig, defaultOpts *OptionsConfig) bool {
	// Check top-level fields
	if userOpts.Editor == "" {
		userOpts.Editor = defaultOpts.Editor
	}
	if userOpts.Theme == "" {
		userOpts.Theme = defaultOpts.Theme
	}

	// Check Discord options
	if userOpts.Discord.Token == "" {
		userOpts.Discord.Token = defaultOpts.Discord.Token
	}
	if userOpts.Discord.ServerID == "" {
		userOpts.Discord.ServerID = defaultOpts.Discord.ServerID
	}
	if userOpts.Discord.DebugChannelID == "" {
		userOpts.Discord.DebugChannelID = defaultOpts.Discord.DebugChannelID
	}

	// Check Services
	if userOpts.Services == nil {
		userOpts.Services = make(map[string]map[string]interface{})
	}
	for svcName, defConfig := range defaultOpts.Services {
		if _, exists := userOpts.Services[svcName]; !exists {
			userOpts.Services[svcName] = defConfig
		} else {
			// Merge nested map (ensure keys exist if missing)
			for k, v := range defConfig {
				if _, ok := userOpts.Services[svcName][k]; !ok {
					userOpts.Services[svcName][k] = v
				}
			}
		}
	}

	// Server Map
	_, serverMapErr := LoadServerMapConfig()
	if serverMapErr != nil {
		if os.IsNotExist(serverMapErr) {
			fmt.Println("Creating default server-map.json...")
			if err := SaveServerMapConfig(DefaultServerMapConfig()); err != nil {
				ui.PrintInfo("failed to save default server-map.json")
				return false
			}
		} else {
			ui.PrintInfo("failed to load server-map.json")
			return false
		}
	}

	return true
}

// LogFile returns a file handle to the dex-cli log file.
func LogFile() (*os.File, error) {
	logPath, err := ExpandPath(filepath.Join(DexterRoot, "logs", "dex-cli.log"))
	if err != nil {
		return nil, fmt.Errorf("failed to expand log file path: %w", err)
	}

	// Ensure the directory exists.
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open the file in append mode, create it if it doesn't exist.
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return file, nil
}

// Log writes a message to the dex-cli log file.
func Log(message string) {
	f, err := LogFile()
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = fmt.Fprintln(f, message)
}

// IsDevMode checks if the EasterCompany source directory exists.
func IsDevMode() bool {
	// Check if the source code directory exists
	path, err := ExpandPath("~/EasterCompany/dex-cli")
	if err != nil {
		return false
	}
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
