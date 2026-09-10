package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/manjunathrathod/claude-repo-factory/internal/version"
)

func newVersionCommand() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print build information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			info := version.Get()
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), info.String())
			return err
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "Emit build information as JSON")
	return cmd
}
