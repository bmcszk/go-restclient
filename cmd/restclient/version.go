package main

import (
	"regexp"
	"runtime/debug"
)

// trustedSemver matches an exact tagged version, e.g. v1.2.3 (never pseudoVersions).
var trustedSemver = regexp.MustCompile(`^v?\d+\.\d+\.\d+$`)

// buildVersion reports the CLI version from embedded build info: tagged module
// version for go-install builds, vcs revision for tree builds, "dev" otherwise.
func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if v := info.Main.Version; trustedSemver.MatchString(v) {
		return v
	}
	if rev, dirty := vcsInfo(info); rev != "" {
		return "devel-" + rev[:min(7, len(rev))] + dirty
	}
	return "dev"
}

// vcsInfo returns the embedded vcs revision and dirty flag, if any.
func vcsInfo(info *debug.BuildInfo) (rev, dirty string) {
	rev, dirty = "", ""
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-dirty"
			}
		default:
		}
	}
	return rev, dirty
}
