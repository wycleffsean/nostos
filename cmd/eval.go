package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.lsp.dev/uri"

	"github.com/spf13/cobra"

	"github.com/wycleffsean/nostos/lang"
	"github.com/wycleffsean/nostos/pkg/types"
	"github.com/wycleffsean/nostos/pkg/workspace"
	"github.com/wycleffsean/nostos/vm"
)

var evalCmd = &cobra.Command{
	Use:   "eval [file]",
	Short: "Evaluate NostOS code from stdin or a file",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			data    []byte
			err     error
			baseDir string
			u       uri.URI
		)

		if len(args) > 0 {
			path := args[0]
			data, err = os.ReadFile(path)
			if err != nil {
				return err
			}
			baseDir = filepath.Dir(path)
			u = uri.File(path)
			workspace.Set(baseDir)
		} else {
			data, err = io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			baseDir = workspace.Dir()
			u = uri.URI("stdin")
		}

		_, items := lang.NewStringLexer(string(data))
		p := lang.NewParser(items, u)
		ast := p.Parse()
		if perrs := lang.CollectParseErrors(ast); len(perrs) > 0 {
			return perrs[0]
		}
		res, err := vm.EvalWithDir(ast, baseDir, u)
		if err != nil {
			return err
		}
		fmt.Print(types.InspectValue(res))
		return nil
	},
}

func init() {
	RootCmd.AddCommand(evalCmd)
}
