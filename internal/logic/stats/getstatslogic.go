// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package stats

import (
	"context"

	"github.com/joeblew999/plat-mjml/internal/errorx"
	"github.com/joeblew999/plat-mjml/internal/svc"
	"github.com/joeblew999/plat-mjml/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetStatsLogic {
	return &GetStatsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetStatsLogic) GetStats() (resp *types.StatsResponse, err error) {
	stats, err := l.svcCtx.Queue.Stats(l.ctx)
	if err != nil {
		return nil, errorx.ErrInternal("failed to get stats: " + err.Error())
	}

	total := 0
	for _, count := range stats {
		total += count
	}

	return &types.StatsResponse{
		Stats: stats,
		Total: total,
	}, nil
}
