package main

import "github.com/samber/oops"

func cliErrorf(format string, args ...any) error {
	return oops.In("cli").Errorf(format, args...)
}

func wrapCLIErrorf(err error, format string, args ...any) error {
	return oops.In("cli").Wrapf(err, format, args...)
}
