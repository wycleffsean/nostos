package planner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"

	"github.com/wycleffsean/nostos/pkg/workspace"
)

func TestFilterSystemNamespace(t *testing.T) {
	resources := []ResourceType{
		{APIVersion: "v1", Kind: "A", Metadata: map[string]interface{}{"name": "a", "namespace": "default"}},
		{APIVersion: "v1", Kind: "B", Metadata: map[string]interface{}{"name": "b", "namespace": "kube-system"}},
	}
	filtered := FilterSystemNamespace(resources)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 resource got %d", len(filtered))
	}
	if ResourceID(filtered[0]) != "v1:A:default:a" {
		t.Fatalf("unexpected resource: %+v", filtered[0])
	}
}

func TestFilterClusterScoped(t *testing.T) {
	resources := []ResourceType{
		{APIVersion: "v1", Kind: "Namespace", Metadata: map[string]interface{}{"name": "default"}},
		{APIVersion: "v1", Kind: "ConfigMap", Metadata: map[string]interface{}{"name": "cfg", "namespace": "default"}},
	}
	filtered := FilterClusterScoped(resources)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 resource got %d", len(filtered))
	}
	if ResourceID(filtered[0]) != "v1:ConfigMap:default:cfg" {
		t.Fatalf("unexpected resource: %+v", filtered[0])
	}
}

func TestBuildPlanFromOdyssey(t *testing.T) {
	tmp := t.TempDir()

	// workspace and odyssey file
	workspace.Set(tmp)

	odyssey := `
test:
  foo:
    - svc.yaml
`
	if err := os.WriteFile(filepath.Join(tmp, "odyssey.no"), []byte(odyssey), 0644); err != nil {
		t.Fatalf("write odyssey: %v", err)
	}

	svc := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: cfg
`
	if err := os.WriteFile(filepath.Join(tmp, "svc.yaml"), []byte(svc), 0644); err != nil {
		t.Fatalf("write svc: %v", err)
	}

	kubeconfig := `
apiVersion: v1
kind: Config
current-context: test
contexts:
- name: test
  context:
    cluster: test
    user: test
clusters:
- name: test
  cluster:
    server: https://example.com
users:
- name: test
  user: {}
`
	kc := filepath.Join(tmp, "kubeconfig")
	if err := os.WriteFile(kc, []byte(kubeconfig), 0644); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}

	viper.Set("kubeconfig", kc)
	viper.Set("context", "")

	plan, err := BuildPlanFromOdyssey(false, false)
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}

	if len(plan.Resources) != 2 {
		t.Fatalf("expected 2 resources got %d", len(plan.Resources))
	}
	if id := ResourceID(plan.Resources[0]); id != "v1:Namespace::foo" {
		t.Fatalf("unexpected namespace resource %s", id)
	}
	if id := ResourceID(plan.Resources[1]); id != "v1:ConfigMap:foo:cfg" {
		t.Fatalf("unexpected resource %s", id)
	}
}

func TestBuildPlanFromOdysseyImports(t *testing.T) {
	tmp := t.TempDir()
	workspace.Set(tmp)

	odyssey := `
test:
  foo:
    - import(./svc.no)
`
	if err := os.WriteFile(filepath.Join(tmp, "odyssey.no"), []byte(odyssey), 0644); err != nil {
		t.Fatalf("write odyssey: %v", err)
	}

	svc := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: cfg
`
	if err := os.WriteFile(filepath.Join(tmp, "svc.no"), []byte(svc), 0644); err != nil {
		t.Fatalf("write svc: %v", err)
	}

	kubeconfig := `
apiVersion: v1
kind: Config
current-context: test
contexts:
- name: test
  context:
    cluster: test
    user: test
clusters:
- name: test
  cluster:
    server: https://example.com
users:
- name: test
  user: {}
`
	kc := filepath.Join(tmp, "kubeconfig")
	if err := os.WriteFile(kc, []byte(kubeconfig), 0644); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}

	viper.Set("kubeconfig", kc)
	viper.Set("context", "")

	plan, err := BuildPlanFromOdyssey(false, false)
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}

	if len(plan.Resources) != 2 {
		t.Fatalf("expected 2 resources got %d", len(plan.Resources))
	}
	if id := ResourceID(plan.Resources[0]); id != "v1:Namespace::foo" {
		t.Fatalf("unexpected namespace resource %s", id)
	}
	if id := ResourceID(plan.Resources[1]); id != "v1:ConfigMap:foo:cfg" {
		t.Fatalf("unexpected resource %s", id)
	}
}
