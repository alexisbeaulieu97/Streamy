package main

// Blank imports ensure plugin init() registration runs for the CLI binary.
import (
	_ "github.com/alexisbeaulieu97/streamy/internal/plugins/command"
	_ "github.com/alexisbeaulieu97/streamy/internal/plugins/copy"
	_ "github.com/alexisbeaulieu97/streamy/internal/plugins/package"
	_ "github.com/alexisbeaulieu97/streamy/internal/plugins/repo"
	_ "github.com/alexisbeaulieu97/streamy/internal/plugins/symlink"
)
