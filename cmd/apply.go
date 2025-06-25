package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"

	"github.com/wycleffsean/nostos/pkg/kube"
	"github.com/wycleffsean/nostos/pkg/planner"
)

// applyCmd represents the apply command.
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply the computed changes to your cluster.",
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterPlan, err := planner.BuildPlanFromCluster(ignoreSystemNamespace, ignoreClusterScoped)
		if err != nil {
			return err
		}
		var desired []planner.ResourceType
		diff := planner.DiffResources(clusterPlan.Resources, desired)
		plan, err := planner.BuildPlanFromDiff(diff)
		if err != nil {
			return err
		}
		return runApply(plan)
	},
}

func init() {
	RootCmd.AddCommand(applyCmd)
}

func runApply(resources []planner.ResourceType) error {
	tty := isatty.IsTerminal(os.Stdout.Fd())
	if !tty {
		color.NoColor = true
	}
	sp := spinner.New(spinner.CharSets[14], 120*time.Millisecond)
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	config, err := kube.LoadKubeConfig()
	if err != nil {
		return err
	}
	dyn, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return err
	}
	groupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	for i, r := range resources {
		id := planner.ResourceID(r)
		prefix := fmt.Sprintf("%2d/%d", i+1, len(resources))
		if tty {
			sp.Prefix = prefix + " "
			sp.Suffix = " " + cyan(id)
			sp.Start()
		} else {
			fmt.Printf("%s applying %s\n", prefix, id)
		}
		err := kube.ApplyResource(context.TODO(), dyn, mapper, planner.ConvertResourceType(r))
		if tty {
			sp.Stop()
			if err != nil {
				fmt.Printf("%s %s %s\n", prefix, red("✗"), id)
			} else {
				fmt.Printf("%s %s %s\n", prefix, green("✔"), id)
			}
		} else {
			if err != nil {
				fmt.Printf("%s failed %s: %v\n", prefix, id, err)
			} else {
				fmt.Printf("%s applied %s\n", prefix, id)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}
