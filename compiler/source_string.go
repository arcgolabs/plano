package compiler

import (
	"context"

	"github.com/arcgolabs/plano/diag"
)

func (c *Compiler) BindString(ctx context.Context, filename, src string) (*Binding, diag.Diagnostics) {
	return c.BindSource(ctx, filename, []byte(src))
}

func (c *Compiler) BindStringDetailed(ctx context.Context, filename, src string) BindResult {
	return c.BindSourceDetailed(ctx, filename, []byte(src))
}

func (c *Compiler) CheckString(ctx context.Context, filename, src string) (*CheckInfo, diag.Diagnostics) {
	return c.CheckSource(ctx, filename, []byte(src))
}

func (c *Compiler) CheckStringDetailed(ctx context.Context, filename, src string) CheckResult {
	return c.CheckSourceDetailed(ctx, filename, []byte(src))
}

func (c *Compiler) CompileString(ctx context.Context, filename, src string) (*Document, diag.Diagnostics) {
	return c.CompileSource(ctx, filename, []byte(src))
}

func (c *Compiler) CompileStringDetailed(ctx context.Context, filename, src string) Result {
	return c.CompileSourceDetailed(ctx, filename, []byte(src))
}
