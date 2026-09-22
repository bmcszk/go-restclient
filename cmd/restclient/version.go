package main

import (
	"regexp"
	"runtime/debug"
)

// trustedSemver matches an exact tagged version, e.g. v1.2.3 (never pseudoVersions).
var trustedSemver = regexp.MustCompile(`^v?\d+\.\d+\.\d+$`)

// resolveVersion returns the effective CLI version: ldflags-injected value wins,
// then a tagged go-install module version, then vcs revision, then "dev".
func resolveVersion() string {
	if version != "dev" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	if v := info.Main.Version; trustedSemver.MatchString(v) {
		return v
	}
	vcs := vcsInfo(info)
	if vcs.revision != "" {
		return "devel-" + vcs.revision[:min(7, len(vcs.revision))] + vcs.dirty
	}
	return version
}

// vcsInfo extracts revision and dirty flag from build settings.
func vcsInfo(info *debug.BuildInfo) (vcs struct {
	revision, dirty string
}) {
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			vcs.revision = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				vcs.dirty = "-dirty"
			}
		default:
		}
	}
	return vcs
}
