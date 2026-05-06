package pipelinedsl

import "github.com/samber/oops"

func pipelineDSLErrorf(format string, args ...any) error {
	return oops.In("pipelinedsl").Errorf("pipelinedsl: "+format, args...)
}

func wrapPipelineDSLErrorf(err error, format string, args ...any) error {
	return oops.In("pipelinedsl").Wrapf(err, "pipelinedsl: "+format, args...)
}
