// Package update finds out when a newer taskpoet has been released, and
// installs it.
package update

import (
	"fmt"
	"regexp"
	"strconv"
)

// Version is a plain MAJOR.MINOR.PATCH release version
type Version struct {
	Major, Minor, Patch int
}

var versionRE = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// ParseVersion reads "v2.1.0" or "2.1.0". Anything else is an error, and that
// includes what 'git describe' makes of a commit after a tag, "v2.1.0-3-gabc",
// and "dev": builds like that are not releases, so there is nothing to compare.
func ParseVersion(s string) (Version, error) {
	m := versionRE.FindStringSubmatch(s)
	if m == nil {
		return Version{}, fmt.Errorf("%q is not a release version", s)
	}
	var parts [3]int
	for i := range parts {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return Version{}, fmt.Errorf("%q is not a release version: %w", s, err)
		}
		parts[i] = n
	}
	return Version{parts[0], parts[1], parts[2]}, nil
}

// After is true if v is a later release than o
func (v Version) After(o Version) bool {
	if v.Major != o.Major {
		return v.Major > o.Major
	}
	if v.Minor != o.Minor {
		return v.Minor > o.Minor
	}
	return v.Patch > o.Patch
}

// String is the version without the v, like the version in an archive name
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Tag is the git tag of the release
func (v Version) Tag() string {
	return "v" + v.String()
}
