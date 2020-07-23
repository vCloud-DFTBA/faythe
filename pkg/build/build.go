package build

import "github.com/prometheus/common/version"

// Version information passed to Faythe version package
// Package path as used by linker changes based on vendoring being used or not,
var (
	Version   string
	Revision  string
	Branch    string
	BuildUser string
	BuildDate string
)

func init() {
	version.Version = Version
	version.Revision = Revision
	version.Branch = Branch
	version.BuildUser = BuildUser
	version.BuildDate = BuildDate
}
