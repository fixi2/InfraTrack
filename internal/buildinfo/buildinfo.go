package buildinfo

// These values can be overridden at build time with:
// -ldflags "-X github.com/fixi2/InfraTrack/internal/buildinfo.Version=v0.2.0"
var Version = "dev"

func String() string {
	return Version
}
