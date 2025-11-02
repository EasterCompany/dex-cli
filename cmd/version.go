package cmd

import (
	"fmt"

	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/ui"
)

func Version(version, date string) {
	branch, commit := git.GetVersionInfo(".")
	versionString := fmt.Sprintf(
		"dex (build name) | dex-cli (build project name) | ~/Dexter/bin/dex (original build target location) | Easter Company (company author) | version %s (version label) | release 1 (release label) | commit: %s@%s (branch and commit info of source from build) | build: %s (any other build info??)",
		version, branch, commit, date,
	)
	ui.PrintInfo(versionString)
}
