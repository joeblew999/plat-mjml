// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package email

import (
	"context"

	"github.com/joeblew999/plat-mjml/internal/errorx"
	"github.com/joeblew999/plat-mjml/internal/svc"
	"github.com/joeblew999/plat-mjml/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetEmailStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetEmailStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetEmailStatusLogic {
	return &GetEmailStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetEmailStatusLogic) GetEmailStatus(req *types.GetEmailStatusRequest) (resp *types.GetEmailStatusResponse, err error) {
	job, err := l.svcCtx.Queue.GetStatus(l.ctx, req.Id)
	if err != nil {
		return nil, errorx.ErrInternal("failed to get email status: " + err.Error())
	}
	if job == nil {
		return nil, errorx.ErrNotFound("email not found: " + req.Id)
	}

	return &types.GetEmailStatusResponse{
		Id:         job.ID,
		Template:   job.TemplateSlug,
		Recipients: job.Recipients,
		Subject:    job.Subject,
		Status:     job.Status,
		Attempts:   job.Attempts,
		Error:      job.Error,
		CreatedAt:  job.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}
