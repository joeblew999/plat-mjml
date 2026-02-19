// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package email

import (
	"context"
	"errors"

	"github.com/joeblew999/plat-mjml/internal/errorx"
	"github.com/joeblew999/plat-mjml/internal/model"
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
	email, err := l.svcCtx.EmailsModel.FindOne(l.ctx, req.Id)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, errorx.ErrNotFound("email not found: " + req.Id)
		}
		return nil, errorx.ErrInternal("failed to get email status: " + err.Error())
	}

	return &types.GetEmailStatusResponse{
		Id:         email.Id,
		Template:   email.TemplateSlug,
		Recipients: model.ParseRecipients(email.Recipients),
		Subject:    email.Subject,
		Status:     email.Status,
		Attempts:   int(email.Attempts),
		Error:      model.NullStringValue(email.Error),
		CreatedAt:  email.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}
