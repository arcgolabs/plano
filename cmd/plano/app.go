package main

import (
	"github.com/arcgolabs/dix"
	planomodule "github.com/arcgolabs/plano"
	"github.com/arcgolabs/plano/compiler"
	"github.com/spf13/cobra"
)

type compilerFactory func() *compiler.Compiler

func newCLIApp() *dix.App {
	return dix.New(
		"plano-cli",
		dix.Version(planomodule.Version),
		dix.Modules(cliModule()),
	)
}

func cliModule() dix.Module {
	return dix.NewModule(
		"cli",
		dix.Providers(
			dix.Provider0(newCompilerFactory),
			dix.Provider1(newCompilerRunner),
			dix.Provider1(buildRootCmd),
		),
	)
}

func newRootCmdE() (*cobra.Command, error) {
	runtime, err := newCLIApp().Build()
	if err != nil {
		return nil, err
	}
	return dix.ResolveAs[*cobra.Command](runtime.Container())
}

func newRootCmd() *cobra.Command {
	cmd, err := newRootCmdE()
	if err != nil {
		panic(err)
	}
	return cmd
}

func newCompilerFactory() compilerFactory {
	return func() *compiler.Compiler {
		return compiler.New(compiler.Options{})
	}
}
