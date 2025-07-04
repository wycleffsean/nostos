package cmd

import (
	"fmt"
	"io"
	"os"

	"go.lsp.dev/uri"

	"github.com/spf13/cobra"

	"github.com/wycleffsean/nostos/lang"
	"github.com/wycleffsean/nostos/pkg/types"
	"github.com/wycleffsean/nostos/vm"
)

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Evaluate NostOS code from stdin",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		_, items := lang.NewStringLexer(string(data))
		p := lang.NewParser(items, uri.URI("stdin"))
		ast := p.Parse()
		if perrs := lang.CollectParseErrors(ast); len(perrs) > 0 {
			return perrs[0]
		}
		res, err := vm.EvalWithDir(ast, ".", uri.URI("stdin"))
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
