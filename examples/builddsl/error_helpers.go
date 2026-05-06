package builddsl

import "github.com/samber/oops"

func buildDSLErrorf(format string, args ...any) error {
	return oops.In("builddsl").Errorf("builddsl: "+format, args...)
}

func wrapBuildDSLErrorf(err error, format string, args ...any) error {
	return oops.In("builddsl").Wrapf(err, "builddsl: "+format, args...)
}
