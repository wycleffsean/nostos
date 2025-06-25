package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/wycleffsean/nostos/pkg/planner"
)

var diffColor bool

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show differences between cluster and desired resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterPlan, err := planner.BuildPlanFromCluster(ignoreSystemNamespace, ignoreClusterScoped)
		if err != nil {
			return err
		}

		// TODO: load desired resources from user files once parser is implemented
		var desired []planner.ResourceType

		diff := planner.DiffResources(clusterPlan.Resources, desired)
		printDiff(diff, diffColor || isatty.IsTerminal(os.Stdout.Fd()))
		return nil
	},
}

func init() {
	diffCmd.Flags().BoolVar(&diffColor, "color", false, "force color output")
	RootCmd.AddCommand(diffCmd)
}

func printDiff(d planner.DiffResult, useColor bool) {
	if !useColor {
		color.NoColor = true
	}
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	magenta := color.New(color.FgMagenta).SprintFunc()

	for _, r := range d.ToCreate {
		fmt.Printf("%s %s\n", green("+"), planner.ResourceID(r))
	}
	for _, u := range d.ToUpdate {
		fmt.Printf("%s %s\n", yellow("~"), planner.ResourceID(u.Desired))
		diffText := planner.DiffString(u.Current, u.Desired)
		fmt.Print(diffText)
	}
	for _, r := range d.Unmanaged {
		fmt.Printf("%s %s\n", magenta("?"), planner.ResourceID(r))
	}
}
