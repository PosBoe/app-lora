// Package version holds the release version of the binary.
package version

// Version is set at link time with -ldflags "-X .../pkg/version.Version=v1.2.3".
var Version = "dev"
