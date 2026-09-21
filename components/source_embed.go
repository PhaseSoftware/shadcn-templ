package components

import "embed"

// SourceFiles supplies component source to the docs pages and registry API.
// This file is repo-only and is not listed in any registry item.
//
//go:embed all:*
var SourceFiles embed.FS
