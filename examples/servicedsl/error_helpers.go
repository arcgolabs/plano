package servicedsl

import "github.com/samber/oops"

func serviceDSLErrorf(format string, args ...any) error {
	return oops.In("servicedsl").Errorf("servicedsl: "+format, args...)
}

func wrapServiceDSLErrorf(err error, format string, args ...any) error {
	return oops.In("servicedsl").Wrapf(err, "servicedsl: "+format, args...)
}
