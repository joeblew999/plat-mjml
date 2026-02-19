// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package template

import (
	"net/http"

	"github.com/joeblew999/plat-mjml/internal/logic/template"
	"github.com/joeblew999/plat-mjml/internal/svc"
	"github.com/joeblew999/plat-mjml/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetTemplateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetTemplateRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := template.NewGetTemplateLogic(r.Context(), svcCtx)
		resp, err := l.GetTemplate(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
