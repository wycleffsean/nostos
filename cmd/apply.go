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
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	if isatty.IsTerminal(os.Stdout.Fd()) {
		return watchApply(dag)
	}
	return applyNonTTY(dag)
}

func applyNonTTY(dag *planner.DAG) error {
	color.NoColor = true

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
		fmt.Printf("%s applying %s\n", prefix, id)

		err := applyResource(context.TODO(), dyn, mapper, r, prefix)

		if err != nil {
			fmt.Printf("%s failed %s: %v\n", prefix, id, err)
			return err
		}
		fmt.Printf("%s applied %s\n", prefix, id)
	}
	return nil
}

type resourceStatus struct {
	ready  bool
	detail string
	err    error
}

func watchApply(dag *planner.DAG) error {
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
				st := statuses[n.ID]
				if st.detail == "applying" && err != nil {
					st.err = err
				} else {
					st.ready = ready
					st.detail = detail
					st.err = err
				}
				statuses[n.ID] = st
				mu.Unlock()
				if ready || err != nil {
					return
				}
				time.Sleep(2 * time.Second)
			}
		}(node)
	}

	applyErr := make(chan error, 1)
	go func() {
		for _, node := range dag.Order {
			mu.Lock()
			st := statuses[node.ID]
			st.detail = "applying"
			statuses[node.ID] = st
			mu.Unlock()
			err := applyResource(context.TODO(), dyn, mapper, node.Resource, "")
			mu.Lock()
			if err != nil {
				st := statuses[node.ID]
				st.err = err
				statuses[node.ID] = st
				mu.Unlock()
				applyErr <- err
				return
			}
			mu.Unlock()
		}
		applyErr <- nil
	}()

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

		select {
		case err := <-applyErr:
			if err != nil {
				return err
			}
			if allReady {
				return nil
			}
		default:
			if allReady {
				select {
				case err := <-applyErr:
					return err
				default:
				}
			}
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
		if errors.IsNotFound(err) {
			return false, "pending", nil
		}
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
