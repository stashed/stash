package apiversion

import (
	"fmt"
	"regexp"
	"strconv"
)

type Version struct {
	X int
	Y string
	Z int
}

type InvalidVersion struct {
	v string
}

func (e InvalidVersion) Error() string {
	return fmt.Sprintf("invalid version %s", e.v)
}

var (
	re = regexp.MustCompile(`^v(\d+)(alpha|beta|rc)?(\d*)$`)
)

func NewVersion(s string) (*Version, error) {
	groups := re.FindStringSubmatch(s)
	if len(groups) == 0 {
		return nil, InvalidVersion{v: s}
	}

	var out Version

	x, err := strconv.Atoi(groups[1])
	if err != nil {
		return nil, err
	}
	out.X = x

	out.Y = groups[2]

	if groups[3] != "" {
		z, err := strconv.Atoi(groups[3])
		if err != nil {
			return nil, err
		}
		out.Z = z
	}

	return &out, nil
}

// Compare returns an integer comparing two version strings.
// The result will be 0 if v==other, -1 if v < other, and +1 if v > other.
func (v Version) Compare(other Version) int {
	diffX := v.X - other.X
	switch {
	case diffX > 0:
		return 1
	case diffX < 0:
		return -1
	}

	if v.Y != other.Y {
		if v.Y == "" {
			return 1
		} else if other.Y == "" {
			return -1
		} else if v.Y > other.Y {
			return 1
		} else {
			return -1
		}
	}

	diffZ := v.Z - other.Z
	switch {
	case diffZ > 0:
		return 1
	case diffZ < 0:
		return -1
	}
	return 0
}

// Compare returns an integer comparing two version strings.
// The result will be 0 if x==y, -1 if x < y, and +1 if x > y.
// An error is returned, if version string can't be parsed.
func Compare(x, y string) (int, error) {
	xv, err := NewVersion(x)
	if err != nil {
		return 0, err
	}
	yv, err := NewVersion(y)
	if err != nil {
		return 0, err
	}
	return xv.Compare(*yv), nil
}

// MustCompare returns an integer comparing two version strings.
// The result will be 0 if x==y, -1 if x < y, and +1 if x > y.
func MustCompare(x, y string) int {
	result, err := Compare(x, y)
	if err != nil {
		panic(err)
	}
	return result
}
