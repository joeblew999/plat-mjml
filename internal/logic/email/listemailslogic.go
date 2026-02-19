// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package email

import (
	"context"

	"github.com/joeblew999/plat-mjml/internal/errorx"
	"github.com/joeblew999/plat-mjml/internal/model"
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
	emails, err := l.svcCtx.EmailsModel.ListByStatus(l.ctx, req.Status, req.Limit)
	if err != nil {
		return nil, errorx.ErrInternal("failed to list emails: " + err.Error())
	}

	items := make([]types.GetEmailStatusResponse, 0, len(emails))
	for _, e := range emails {
		items = append(items, types.GetEmailStatusResponse{
			Id:         e.Id,
			Template:   e.TemplateSlug,
			Recipients: model.ParseRecipients(e.Recipients),
			Subject:    e.Subject,
			Status:     e.Status,
			Attempts:   int(e.Attempts),
			Error:      model.NullStringValue(e.Error),
			CreatedAt:  e.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return &types.ListEmailsResponse{
		Emails: items,
		Count:  len(items),
	}, nil
}
