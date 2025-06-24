package lsp

import (
	"context"
	"github.com/wycleffsean/nostos/pkg/kube"
	"github.com/wycleffsean/nostos/pkg/types"
	"go.lsp.dev/protocol"
)

func StartRegistryWorker(ctx context.Context, state *ServerState) {
	go runRegistryWorker(ctx, state)
}

func runRegistryWorker(ctx context.Context, state *ServerState) {
	client := state.client

	log.Sugar().Infow("Starting FetchAndFillRegistry")

	registry, err := kube.FetchAndFillRegistry() // TODO: receives ctx
	if err != nil {
		log.Sugar().Warnw("FetchAndFillRegistry failed, using built-in registry", "error", err)
		registry = types.DefaultRegistry()
	}

	log.Sugar().Infow("Registry ready")
	state.mu.Lock()
	state.registry = registry
	state.mu.Unlock()
	state.registryReady <- registry
	close(state.registryReady) // ✅ signal completion to listeners

	if client != nil {
		_ = client.LogMessage(ctx, &protocol.LogMessageParams{
			Type:    protocol.MessageTypeInfo,
			Message: "nostos: Kubernetes type registry loaded.",
		})
	}
}
