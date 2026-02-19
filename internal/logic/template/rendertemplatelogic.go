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

type RenderTemplateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRenderTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RenderTemplateLogic {
	return &RenderTemplateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RenderTemplateLogic) RenderTemplate(req *types.RenderTemplateRequest) (resp *types.RenderTemplateResponse, err error) {
	testData := mjml.TestData()
	data := testData[req.Slug]
	if data == nil {
		data = testData["simple"]
	}

	html, err := l.svcCtx.Renderer.RenderTemplate(req.Slug, data)
	if err != nil {
		return nil, errorx.ErrInternal("failed to render template: " + err.Error())
	}

	return &types.RenderTemplateResponse{
		Html:     html,
		Template: req.Slug,
		Size:     len(html),
	}, nil
}
