# Odyssey Files

An `odyssey.no` file is the **entrypoint** for a Nostos project. The top‑level keys
map to **kubectl contexts/clusters**. Each cluster key contains namespaces; each
namespace lists the **resources** to materialize.

```no
let
  svc: import(./svc.no)
  deploy: import(./deploy.no)
in
my-cluster:
  default:
  - svc
  - deploy
```

- **`let … in`** binds local names to imported modules/resources (Nix‑style).
- **Cluster key** (`my-cluster`) is inferred as the kubectl context/cluster.
- **Namespace** (`default`) groups resources for that namespace.
- **Resource list** are expressions that evaluate to Kubernetes specs.

Nostos evaluates the file into a **typed graph** using built‑in Kubernetes type
information plus CRDs discovered from the cluster. The resulting graph is diffed
against live state to produce a safe plan.
