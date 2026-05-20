// Package cli wires the cobra root command and subcommands for
// agno-docs-pp-cli. Every leaf command supports --json for agent consumption
// and --db to override the default database path.
package cli

import (
	"github.com/spf13/cobra"
)

// Version is overridden via -ldflags at build time.
var Version = "dev"

// Flags holds globals shared by all subcommands.
type Flags struct {
	JSON bool
	DB   string
}

// NewRootCmd returns the root cobra command with all leaves attached.
func NewRootCmd() *cobra.Command {
	f := &Flags{}
	cmd := &cobra.Command{
		Use:   "agno-docs-pp-cli",
		Short: "Offline lookup CLI for Agno developer documentation",
		Long: `agno-docs-pp-cli indexes docs.agno.com/llms-full.txt into a local
SQLite/FTS5 database and serves agent-native lookup commands.

Run 'agno-docs-pp-cli sync' once to download and index the docs (~10s),
then use find/which/context/examples/sections from any agent or terminal.`,
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	cmd.PersistentFlags().BoolVar(&f.JSON, "json", false, "Emit JSON output for agent consumption")
	cmd.PersistentFlags().StringVar(&f.DB, "db", "", "Database path (default: ~/.local/share/agno-docs-pp-cli/data.db)")

	cmd.AddCommand(
		newSyncCmd(f),
		newFindCmd(f),
		newWhichCmd(f),
		newContextCmd(f),
		newExamplesCmd(f),
		newSectionsCmd(f),
		newDoctorCmd(f),
		newVersionCmd(),
	)
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println(Version)
			return nil
		},
	}
}
