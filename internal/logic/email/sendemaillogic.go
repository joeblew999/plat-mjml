// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package email

import (
	"context"

	"github.com/joeblew999/plat-mjml/internal/errorx"
	"github.com/joeblew999/plat-mjml/internal/svc"
	"github.com/joeblew999/plat-mjml/internal/types"
	"github.com/joeblew999/plat-mjml/pkg/queue"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendEmailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendEmailLogic {
	return &SendEmailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendEmailLogic) SendEmail(req *types.SendEmailRequest) (resp *types.SendEmailResponse, err error) {
	job := queue.EmailJob{
		TemplateSlug: req.Template,
		Recipients:   req.To,
		Subject:      req.Subject,
		Priority:     queue.PriorityNormal,
	}

	id, err := l.svcCtx.Queue.Enqueue(l.ctx, job)
	if err != nil {
		return nil, errorx.ErrInternal("failed to enqueue email: " + err.Error())
	}

	return &types.SendEmailResponse{
		Id:         id,
		Status:     "queued",
		Recipients: len(req.To),
		Template:   req.Template,
	}, nil
}
