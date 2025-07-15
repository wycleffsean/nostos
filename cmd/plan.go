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
		clusterPlan, err := planner.BuildPlanFromCluster(ignoreSystemNamespace, ignoreClusterScoped)
		if err != nil {
			return err
		}
		odysseyPlan, err := planner.BuildPlanFromOdyssey(ignoreSystemNamespace, ignoreClusterScoped)
		if err != nil {
			return err
		}
		desired := odysseyPlan.Resources

		diff := planner.DiffResources(clusterPlan.Resources, desired)
		list, err := planner.BuildPlanFromDiff(diff)
		if err != nil {
			return err
		}
		dag, err := planner.BuildDAG(list)
		if err != nil {
			return err
		}
		printPlan(dag, planColor || isatty.IsTerminal(os.Stdout.Fd()))
		return nil
	},
}

func init() {
	planCmd.Flags().BoolVar(&planColor, "color", false, "force color output")
	RootCmd.AddCommand(planCmd)
}

func printPlan(dag *planner.DAG, useColor bool) {
	if !useColor {
		color.NoColor = true
	}
	blue := color.New(color.FgCyan).SprintFunc()
	visited := make(map[string]bool)
	for i, root := range dag.Roots {
		printNode(root, "", i == len(dag.Roots)-1, visited, blue)
	}
}

func printNode(n *planner.Node, prefix string, last bool, visited map[string]bool, colorize func(a ...interface{}) string) {
	connector := ""
	if prefix != "" {
		if last {
			connector = "└─ "
		} else {
			connector = "├─ "
		}
	}
	fmt.Printf("%s%s%s\n", prefix, connector, colorize(planner.ResourceID(n.Resource)))
	newPrefix := prefix
	if prefix != "" {
		if last {
			newPrefix += "   "
		} else {
			newPrefix += "│  "
		}
	}
	visited[n.ID] = true
	for i, c := range n.Children {
		if visited[c.ID] {
			continue
		}
		printNode(c, newPrefix, i == len(n.Children)-1, visited, colorize)
	}
}
