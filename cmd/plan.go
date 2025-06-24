package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/wycleffsean/nostos/pkg/planner"
)

var planColor bool

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Generate an execution plan",
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterPlan, err := planner.BuildPlanFromCluster()
		if err != nil {
			return err
		}
		// TODO: load desired resources from user files once parser is implemented
		var desired []planner.ResourceType

		diff := planner.DiffResources(clusterPlan.Resources, desired)
		plan, err := planner.BuildPlanFromDiff(diff)
		if err != nil {
			return err
		}
		printPlan(plan, planColor || isatty.IsTerminal(os.Stdout.Fd()))
		return nil
	},
}

func init() {
	planCmd.Flags().BoolVar(&planColor, "color", false, "force color output")
	RootCmd.AddCommand(planCmd)
}

func printPlan(resources []planner.ResourceType, useColor bool) {
	if !useColor {
		color.NoColor = true
	}
	blue := color.New(color.FgCyan).SprintFunc()
	for i, r := range resources {
		fmt.Printf("%2d. %s\n", i+1, blue(planner.ResourceID(r)))
	}
}
