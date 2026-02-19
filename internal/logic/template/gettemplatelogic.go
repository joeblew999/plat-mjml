// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package template

import (
	"context"

	"github.com/joeblew999/plat-mjml/internal/errorx"
	"github.com/joeblew999/plat-mjml/internal/svc"
	"github.com/joeblew999/plat-mjml/internal/types"
	"github.com/joeblew999/plat-mjml/pkg/mjml"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTemplateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTemplateLogic {
	return &GetTemplateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTemplateLogic) GetTemplate(req *types.GetTemplateRequest) (resp *types.GetTemplateResponse, err error) {
	slugs := l.svcCtx.Renderer.ListTemplates()
	for _, slug := range slugs {
		if slug == req.Slug {
			return &types.GetTemplateResponse{
				Slug:        slug,
				Description: mjml.TemplateDescription(slug),
			}, nil
		}
	}

	return nil, errorx.ErrNotFound("template not found: " + req.Slug)
}
