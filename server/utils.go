package server

import (
	"context"
	"log/slog"

	"github.com/netapp/ontap-mcp/ontap"
)

func (a *App) clusterModelOrDefault(ctx context.Context, cluster string) ontap.Remote {
	remote := ontap.Remote{Model: ontap.CDOT}
	if info, err := a.getClusterRemote(ctx, cluster); err == nil {
		remote = info
		if remote.Model == "" {
			remote.Model = ontap.CDOT
		}
	} else {
		a.logger.Warn("failed to determine cluster model, choosing default model as CDOT", slog.String("cluster", cluster), slog.String("error", err.Error()))
	}
	return remote
}
