package web

import "embed"

// nolint: gochecknoglobals
//go:embed **/*
var files embed.FS

// Files returns a filesystem with static files.
func Files() embed.FS {
	return files
}
