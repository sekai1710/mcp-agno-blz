// Command agno-docs-pp-cli indexes the Agno developer documentation
// (docs.agno.com/llms-full.txt) into a local SQLite/FTS5 database and serves
// agent-native lookup commands: find, which, context, examples, sections.
package main

import (
	"fmt"
	"os"

	"github.com/sekai1710/agno-docs-pp-cli/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
