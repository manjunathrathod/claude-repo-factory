package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/manjunathrathod/claude-repo-factory/internal/plugin"
)

func newLanguagesCommand(app *App) *cobra.Command {
	var asJSON bool
	var includePlanned bool

	cmd := &cobra.Command{
		Use:     "languages",
		Aliases: []string{"langs", "list"},
		Short:   "List the language plugins this build knows about",
		Long: "List the language plugins this build knows about, including the ones\n" +
			"that are on the roadmap but not yet available for generation.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			descriptors := app.Registry.Descriptors()
			if !includePlanned {
				filtered := descriptors[:0:0]
				for _, d := range descriptors {
					if d.Status == plugin.StatusStable {
						filtered = append(filtered, d)
					}
				}
				descriptors = filtered
			}

			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(descriptors)
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tSTATUS\tPROJECT TYPES\tSUMMARY")
			for _, d := range descriptors {
				types := strings.Join(d.ProjectTypeIDs(), ", ")
				if types == "" {
					types = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", d.ID, d.DisplayName, d.Status, types, d.Summary)
			}
			return w.Flush()
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "Emit the language list as JSON")
	cmd.Flags().BoolVar(&includePlanned, "all", true, "Include languages that are planned but not yet available")
	return cmd
}
