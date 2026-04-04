// Package integration provides compile-time interface compliance checks.
// This file ensures that concrete implementations satisfy their respective
// interfaces. If an interface is updated or a method signature changes,
// the compiler will immediately report an error here, rather than at
// runtime or at the point of injection.
//
// No code in this file is ever executed - it exists solely to provide
// early warning when contracts between packages are broken.
//
// To add a new check, use the pattern:
//
//	var _ <package>.<Interface> = (*<impl>.ConcreteType)(nil)
package integration

import (
	"golangwebcrawler/cmd/crawler/internal/crawler"
	"golangwebcrawler/cmd/crawler/internal/fetcher"
	"golangwebcrawler/cmd/crawler/internal/parser"
)

var (
	_ crawler.Fetcher = (*fetcher.HTTPFetcher)(nil)
	_ crawler.Parser  = (*parser.HTTPParser)(nil)
)
