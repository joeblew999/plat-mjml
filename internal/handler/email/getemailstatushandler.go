// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package email

import (
	"net/http"

	"github.com/joeblew999/plat-mjml/internal/logic/email"
	"github.com/joeblew999/plat-mjml/internal/svc"
	"github.com/joeblew999/plat-mjml/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetEmailStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetEmailStatusRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := email.NewGetEmailStatusLogic(r.Context(), svcCtx)
		resp, err := l.GetEmailStatus(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
