package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"strings"

	"github.com/joeblew999/plat-mjml/pkg/mjml"
	"github.com/joeblew999/plat-mjml/pkg/queue"
	"github.com/starfederation/datastar-go/datastar"
)

// Handlers provides HTTP handlers for the UI.
type Handlers struct {
	renderer *mjml.Renderer
	queue    *queue.Queue
}

// NewHandlers creates new UI handlers.
func NewHandlers(renderer *mjml.Renderer, q *queue.Queue) *Handlers {
	return &Handlers{
		renderer: renderer,
		queue:    q,
	}
}

// RegisterRoutes registers all UI routes.
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	// Pages
	mux.HandleFunc("GET /", h.handleDashboard)
	mux.HandleFunc("GET /templates", h.handleTemplates)
	mux.HandleFunc("GET /queue", h.handleQueue)
	mux.HandleFunc("GET /send", h.handleSendPage)

	// API endpoints for Datastar
	mux.HandleFunc("GET /api/stats", h.handleStats)
	mux.HandleFunc("GET /api/queue", h.handleQueueAPI)
	mux.HandleFunc("GET /api/preview/{slug}", h.handlePreview)
	mux.HandleFunc("POST /api/send", h.handleSend)
}

func (h *Handlers) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := Dashboard().Render(w); err != nil {
		log.Printf("render dashboard: %v", err)
	}
}

func (h *Handlers) handleTemplates(w http.ResponseWriter, r *http.Request) {
	templates := h.getTemplateInfos()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := TemplatesPage(templates).Render(w); err != nil {
		log.Printf("render templates page: %v", err)
	}
}

func (h *Handlers) handleQueue(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := QueuePage().Render(w); err != nil {
		log.Printf("render queue page: %v", err)
	}
}

func (h *Handlers) handleSendPage(w http.ResponseWriter, r *http.Request) {
	templates := h.getTemplateInfos()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := SendEmailPage(templates).Render(w); err != nil {
		log.Printf("render send page: %v", err)
	}
}

func (h *Handlers) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.queue.Stats(context.Background())
	if err != nil {
		h.sendDatastarError(w, r, err)
		return
	}

	h.sendDatastarSignals(w, r, map[string]any{
		"stats":   stats,
		"loading": false,
	})
}

func (h *Handlers) handleQueueAPI(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	jobs, err := h.queue.List(context.Background(), status, 50)
	if err != nil {
		h.sendDatastarError(w, r, err)
		return
	}

	// Render queue items as HTML fragment and patch into #queue-items
	sse := datastar.NewSSE(w, r)

	fragment := renderQueueItems(jobs)
	if err := sse.PatchElementf(`<div id="queue-items">%s</div>`, fragment); err != nil {
		log.Printf("datastar patch queue items: %v", err)
	}

	if err := sse.MarshalAndPatchSignals(map[string]any{"loading": false}); err != nil {
		log.Printf("datastar patch signals: %v", err)
	}
}

func (h *Handlers) handlePreview(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		h.sendDatastarError(w, r, nil)
		return
	}

	// Get test data for the template
	testData := mjml.TestData()
	data := testData[slug]
	if data == nil {
		data = testData["simple"]
	}

	html, err := h.renderer.RenderTemplate(slug, data)
	if err != nil {
		h.sendDatastarError(w, r, err)
		return
	}

	h.sendDatastarSignals(w, r, map[string]any{
		"previewHtml": html,
		"loading":     false,
	})
}

func (h *Handlers) handleSend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Template string         `json:"template"`
		To       []string       `json:"to"`
		Subject  string         `json:"subject"`
		Data     map[string]any `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendDatastarSignals(w, r, map[string]any{
			"sending": false,
			"result":  "Error: Invalid request",
		})
		return
	}

	job := queue.EmailJob{
		TemplateSlug: req.Template,
		Recipients:   req.To,
		Subject:      req.Subject,
		Data:         req.Data,
		Priority:     queue.PriorityNormal,
	}

	id, err := h.queue.Enqueue(context.Background(), job)
	if err != nil {
		h.sendDatastarSignals(w, r, map[string]any{
			"sending": false,
			"result":  "Error: " + err.Error(),
		})
		return
	}

	h.sendDatastarSignals(w, r, map[string]any{
		"sending": false,
		"result":  "Email queued with ID: " + id,
	})
}

func (h *Handlers) getTemplateInfos() []TemplateInfo {
	slugs := h.renderer.ListTemplates()
	infos := make([]TemplateInfo, 0, len(slugs))
	for _, slug := range slugs {
		infos = append(infos, TemplateInfo{
			Slug:        slug,
			Description: getTemplateDescription(slug),
		})
	}
	return infos
}

func (h *Handlers) sendDatastarSignals(w http.ResponseWriter, r *http.Request, signals map[string]any) {
	sse := datastar.NewSSE(w, r)
	if err := sse.MarshalAndPatchSignals(signals); err != nil {
		log.Printf("datastar patch signals: %v", err)
	}
}

func (h *Handlers) sendDatastarError(w http.ResponseWriter, r *http.Request, err error) {
	msg := "Unknown error"
	if err != nil {
		msg = err.Error()
	}
	h.sendDatastarSignals(w, r, map[string]any{
		"loading": false,
		"error":   msg,
	})
}

func renderQueueItems(jobs []*queue.EmailJob) string {
	if len(jobs) == 0 {
		return `<p class="hint" style="padding:2rem;text-align:center;">No emails in queue</p>`
	}

	var b strings.Builder
	b.WriteString(`<table style="width:100%;border-collapse:collapse;">`)
	b.WriteString(`<thead><tr>`)
	b.WriteString(`<th style="text-align:left;padding:0.75rem 1rem;border-bottom:2px solid var(--border);color:var(--text-muted);font-size:0.875rem;">Template</th>`)
	b.WriteString(`<th style="text-align:left;padding:0.75rem 1rem;border-bottom:2px solid var(--border);color:var(--text-muted);font-size:0.875rem;">Recipients</th>`)
	b.WriteString(`<th style="text-align:left;padding:0.75rem 1rem;border-bottom:2px solid var(--border);color:var(--text-muted);font-size:0.875rem;">Subject</th>`)
	b.WriteString(`<th style="text-align:left;padding:0.75rem 1rem;border-bottom:2px solid var(--border);color:var(--text-muted);font-size:0.875rem;">Status</th>`)
	b.WriteString(`<th style="text-align:left;padding:0.75rem 1rem;border-bottom:2px solid var(--border);color:var(--text-muted);font-size:0.875rem;">Created</th>`)
	b.WriteString(`</tr></thead><tbody>`)

	for _, job := range jobs {
		statusColor := "var(--text-muted)"
		switch job.Status {
		case "sent":
			statusColor = "var(--success)"
		case "failed":
			statusColor = "var(--danger)"
		case "pending", "scheduled":
			statusColor = "var(--warning)"
		case "retry", "processing":
			statusColor = "var(--primary)"
		}

		recipients := html.EscapeString(strings.Join(job.Recipients, ", "))
		created := job.CreatedAt.Format("Jan 2 15:04")

		b.WriteString(`<tr style="border-bottom:1px solid var(--border);">`)
		b.WriteString(fmt.Sprintf(`<td style="padding:0.75rem 1rem;font-weight:500;">%s</td>`, html.EscapeString(job.TemplateSlug)))
		b.WriteString(fmt.Sprintf(`<td style="padding:0.75rem 1rem;font-size:0.875rem;">%s</td>`, recipients))
		b.WriteString(fmt.Sprintf(`<td style="padding:0.75rem 1rem;font-size:0.875rem;">%s</td>`, html.EscapeString(job.Subject)))
		b.WriteString(fmt.Sprintf(`<td style="padding:0.75rem 1rem;"><span style="color:%s;font-weight:600;font-size:0.875rem;">%s</span></td>`, statusColor, html.EscapeString(job.Status)))
		b.WriteString(fmt.Sprintf(`<td style="padding:0.75rem 1rem;font-size:0.875rem;color:var(--text-muted);">%s</td>`, created))
		b.WriteString(`</tr>`)
	}

	b.WriteString(`</tbody></table>`)
	return b.String()
}

func getTemplateDescription(slug string) string {
	descriptions := map[string]string{
		"simple":                "Basic email template",
		"welcome":               "Welcome/activation email for new users",
		"reset_password":        "Password reset email with security info",
		"notification":          "System notification email",
		"premium_newsletter":    "Newsletter with premium fonts",
		"business_announcement": "Business announcement email",
	}
	if desc, ok := descriptions[slug]; ok {
		return desc
	}
	return "Email template"
}
