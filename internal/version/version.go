// Package version holds build-time version metadata.
package version

// Version is the application version. It may be overridden at build time:
//
//	go build -ldflags "-X fearless/internal/version.Version=v1.2.3"
var Version = "dev"
