package compiler

import "github.com/samber/oops"

func compilerErrorf(format string, args ...any) error {
	return oops.In("compiler").Errorf(format, args...)
}

func wrapCompilerErrorf(err error, format string, args ...any) error {
	return oops.In("compiler").Wrapf(err, format, args...)
}
