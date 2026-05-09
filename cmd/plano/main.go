// Package main implements the plano CLI.
package main

import "os"

func main() {
	cmd, err := newRootCmdE()
	if err != nil {
		if writeErr := writeString(os.Stderr, err.Error()+"\n"); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
	if err := cmd.Execute(); err != nil {
		if writeErr := writeString(cmd.ErrOrStderr(), err.Error()+"\n"); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}
