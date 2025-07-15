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
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
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
		return runApply(dag)
	},
}

func init() {
	RootCmd.AddCommand(applyCmd)
}

func runApply(dag *planner.DAG) error {
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

	for i, node := range dag.Order {
		r := node.Resource
		id := planner.ResourceID(r)
		prefix := fmt.Sprintf("%2d/%d", i+1, len(dag.Order))
		if tty {
			sp.Prefix = prefix + " "
			sp.Suffix = " " + cyan(id)
			sp.Start()
		} else {
			fmt.Printf("%s applying %s\n", prefix, id)
		}

		err := applyResource(context.TODO(), dyn, mapper, r, prefix)

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

func applyResource(ctx context.Context, dyn dynamic.Interface, mapper meta.RESTMapper, r planner.ResourceType, prefix string) error {
	gv, err := schema.ParseGroupVersion(r.APIVersion)
	if err != nil {
		return err
	}
	gvk := gv.WithKind(r.Kind)
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}
	ns, _ := r.Metadata["namespace"].(string)
	watchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	w, watchErr := dyn.Resource(mapping.Resource).Namespace(ns).Watch(watchCtx, metav1.ListOptions{})
	if watchErr == nil {
		go func(id string, watcher watch.Interface) {
			for e := range watcher.ResultChan() {
				fmt.Printf("%s event %s: %s\n", prefix, id, e.Type)
			}
		}(planner.ResourceID(r), w)
	}
	err = kube.ApplyResource(ctx, dyn, mapper, planner.ConvertResourceType(r))
	cancel()
	return err
}
