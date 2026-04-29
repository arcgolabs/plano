package main

import (
	planomodule "github.com/arcgolabs/plano"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the plano release and API compatibility versions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			value := struct {
				Version               string `json:"version"               yaml:"version"`
				PublicAPIVersion      string `json:"publicApiVersion"      yaml:"publicApiVersion"`
				ArtifactSchemaVersion string `json:"artifactSchemaVersion" yaml:"artifactSchemaVersion"`
			}{
				Version:               planomodule.Version,
				PublicAPIVersion:      planomodule.PublicAPIVersion,
				ArtifactSchemaVersion: planomodule.ArtifactSchemaVersion,
			}
			return writeValue(cmd.OutOrStdout(), value, formatJSON)
		},
	}
}
