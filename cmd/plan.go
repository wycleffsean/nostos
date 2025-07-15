package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"

	"github.com/wycleffsean/nostos/pkg/kube"
	"github.com/wycleffsean/nostos/pkg/planner"
)

var (
	planColor bool
	planWatch bool
)

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

		tty := isatty.IsTerminal(os.Stdout.Fd())
		if planWatch && tty {
			return watchPlan(dag)
		}

		printPlan(dag, planColor || tty)
		return nil
	},
}

func init() {
	planCmd.Flags().BoolVar(&planColor, "color", false, "force color output")
	planCmd.Flags().BoolVar(&planWatch, "watch", false, "watch resource status")
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

type resourceStatus struct {
	ready  bool
	detail string
	err    error
}

func watchPlan(dag *planner.DAG) error {
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

	statuses := make(map[string]resourceStatus)
	mu := sync.Mutex{}

	for _, node := range dag.Order {
		id := node.ID
		statuses[id] = resourceStatus{detail: "pending"}
		go func(n *planner.Node) {
			for {
				ready, detail, err := checkResource(context.TODO(), dyn, mapper, n.Resource)
				mu.Lock()
				statuses[n.ID] = resourceStatus{ready: ready, detail: detail, err: err}
				mu.Unlock()
				if ready || err != nil {
					return
				}
				time.Sleep(2 * time.Second)
			}
		}(node)
	}

	spinnerChars := []string{"|", "/", "-", "\\"}
	i := 0
	for {
		mu.Lock()
		allReady := true
		copyStatus := make(map[string]resourceStatus, len(statuses))
		for k, v := range statuses {
			copyStatus[k] = v
			if !v.ready && v.err == nil {
				allReady = false
			}
		}
		mu.Unlock()

		fmt.Print("\033[H\033[2J")
		blue := color.New(color.FgCyan).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()
		visited := make(map[string]bool)
		printNodeStatus(dag.Roots, "", visited, copyStatus, spinnerChars[i%len(spinnerChars)], blue, green, red)

		if allReady {
			return nil
		}

		i++
		time.Sleep(500 * time.Millisecond)
	}
}

func printNodeStatus(nodes []*planner.Node, prefix string, visited map[string]bool, statuses map[string]resourceStatus, spinner string, cyan, green, red func(a ...interface{}) string) {
	for idx, n := range nodes {
		if visited[n.ID] {
			continue
		}
		last := idx == len(nodes)-1
		connector := ""
		if prefix != "" {
			if last {
				connector = "└─ "
			} else {
				connector = "├─ "
			}
		}
		status := statuses[n.ID]
		symbol := spinner
		if status.err != nil {
			symbol = red("✗")
		} else if status.ready {
			symbol = green("✔")
		}
		fmt.Printf("%s%s[%s] %s %s\n", prefix, connector, symbol, cyan(planner.ResourceID(n.Resource)), status.detail)
		newPrefix := prefix
		if prefix != "" {
			if last {
				newPrefix += "   "
			} else {
				newPrefix += "│  "
			}
		}
		visited[n.ID] = true
		printNodeStatus(n.Children, newPrefix, visited, statuses, spinner, cyan, green, red)
	}
}

func checkResource(ctx context.Context, dyn dynamic.Interface, mapper meta.RESTMapper, r planner.ResourceType) (bool, string, error) {
	gv, err := schema.ParseGroupVersion(r.APIVersion)
	if err != nil {
		return false, "gv error", err
	}
	gvk := gv.WithKind(r.Kind)
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gv.Version)
	if err != nil {
		return false, "map error", err
	}
	ns, _ := r.Metadata["namespace"].(string)
	name, _ := r.Metadata["name"].(string)

	u, err := dyn.Resource(mapping.Resource).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, "get error", err
	}

	if strings.EqualFold(r.Kind, "Pod") {
		var pod corev1.Pod
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &pod); err != nil {
			return false, "convert", err
		}
		ready := true
		readyCount := 0
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				readyCount++
			} else {
				ready = false
			}
		}
		detail := fmt.Sprintf("%d/%d containers", readyCount, len(pod.Status.ContainerStatuses))
		return ready && pod.Status.Phase == corev1.PodRunning, detail, nil
	}

	return true, "exists", nil
}
