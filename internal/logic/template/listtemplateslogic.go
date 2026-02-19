// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package template

import (
	"context"

	"github.com/joeblew999/plat-mjml/internal/svc"
	"github.com/joeblew999/plat-mjml/internal/types"
	"github.com/joeblew999/plat-mjml/pkg/mjml"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListTemplatesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListTemplatesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTemplatesLogic {
	return &ListTemplatesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListTemplatesLogic) ListTemplates() (resp *types.ListTemplatesResponse, err error) {
	slugs := l.svcCtx.Renderer.ListTemplates()

	items := make([]types.TemplateItem, 0, len(slugs))
	for _, slug := range slugs {
		items = append(items, types.TemplateItem{
			Slug:        slug,
			Description: mjml.TemplateDescription(slug),
		})
	}

	return &types.ListTemplatesResponse{
		Templates: items,
		Count:     len(items),
	}, nil
}
