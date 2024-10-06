# Goals

Create a "programming language" for kubernetes.  It should be familiar for anybody thats's worked with
the yaml manifests, but provide the nice semantics of a programming language.  We should be able to DRY
up bits, assert referential integrity before application, and do better than lame string templating
(Helm).

It's a typed/declarative system which should pull types from the cluster itself.  This means it should
know about API versions and CRDs. We should strive for determinism, so we provide a programming
environment but no IO or randomness.

The grammar should be very similar to yaml, kind of like how JSX blends markup and logic.

Helm does pre and post hooks.  It stores the release states in the cluster

# Ref

- https://yaml.org/spec/1.2.2
- https://pkg.go.dev/k8s.io/client-go/openapi3
- https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/
- https://tamerlan.dev/how-to-build-a-language-server-with-go/
- https://github.com/tliron/glsp
