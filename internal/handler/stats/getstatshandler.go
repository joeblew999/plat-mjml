// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package stats

import (
	"net/http"

	"github.com/joeblew999/plat-mjml/internal/logic/stats"
	"github.com/joeblew999/plat-mjml/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetStatsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := stats.NewGetStatsLogic(r.Context(), svcCtx)
		resp, err := l.GetStats()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
