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

type ListEmailsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListEmailsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListEmailsLogic {
	return &ListEmailsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListEmailsLogic) ListEmails(req *types.ListEmailsRequest) (resp *types.ListEmailsResponse, err error) {
	jobs, err := l.svcCtx.Queue.List(l.ctx, req.Status, req.Limit)
	if err != nil {
		return nil, errorx.ErrInternal("failed to list emails: " + err.Error())
	}

	emails := make([]types.GetEmailStatusResponse, 0, len(jobs))
	for _, job := range jobs {
		emails = append(emails, types.GetEmailStatusResponse{
			Id:         job.ID,
			Template:   job.TemplateSlug,
			Recipients: job.Recipients,
			Subject:    job.Subject,
			Status:     job.Status,
			Attempts:   job.Attempts,
			Error:      job.Error,
			CreatedAt:  job.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return &types.ListEmailsResponse{
		Emails: emails,
		Count:  len(emails),
	}, nil
}
