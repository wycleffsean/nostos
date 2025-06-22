package lsp

import (
	"context"
	"go.lsp.dev/protocol"
	"github.com/wycleffsean/nostos/pkg/kube"
)

func StartRegistryWorker(ctx context.Context, state *ServerState) {
	go runRegistryWorker(ctx, state)
}

func runRegistryWorker(ctx context.Context, state *ServerState) {
	client := state.client

	log.Sugar().Infow("Starting FetchAndFillRegistry")

	registry, err := kube.FetchAndFillRegistry() // TODO: receives ctx
	if err != nil {
		log.Sugar().Errorw("FetchAndFillRegistry failed", "error", err)
		if client != nil {
			_ = client.ShowMessage(ctx, &protocol.ShowMessageParams{
				Type:    protocol.MessageTypeError,
				Message: "Failed to fetch Kubernetes resources: " + err.Error(),
			})
		}
		close(state.registryReady) // still close to unblock waiters
		return
	}

	log.Sugar().Infow("Registry ready")
	state.registryReady <- registry
	close(state.registryReady) // ✅ signal completion to listeners

	if client != nil {
		_ = client.LogMessage(ctx, &protocol.LogMessageParams{
			Type:    protocol.MessageTypeInfo,
			Message: "nostos: Kubernetes type registry loaded.",
		})
	}
}
