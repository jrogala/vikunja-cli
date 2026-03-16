package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectListCmd)
}

var projectCmd = &cobra.Command{
	Use:     "project",
	Aliases: []string{"p", "projects"},
	Short:   "Manage projects",
}

var projectListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "List all projects with ID, title, favorite and archived status.",
	RunE: func(_ *cobra.Command, _ []string) error {
		c := newClient()
		projects, err := c.GetProjects()
		if err != nil {
			return err
		}

		if outputJSON {
			return printJSON(projects)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tTITLE\tFAV\tARCHIVED")
		for _, p := range projects {
			fav := ""
			if p.IsFavorite {
				fav = "*"
			}
			archived := ""
			if p.IsArchived {
				archived = "yes"
			}
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", p.ID, p.Title, fav, archived)
		}
		_ = w.Flush()
		return nil
	},
}
