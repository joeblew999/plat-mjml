package ui

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

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
	_ = Dashboard().Render(w)
}

func (h *Handlers) handleTemplates(w http.ResponseWriter, r *http.Request) {
	templates := h.getTemplateInfos()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = TemplatesPage(templates).Render(w)
}

func (h *Handlers) handleQueue(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = QueuePage().Render(w)
}

func (h *Handlers) handleSendPage(w http.ResponseWriter, r *http.Request) {
	templates := h.getTemplateInfos()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = SendEmailPage(templates).Render(w)
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

	h.sendDatastarSignals(w, r, map[string]any{
		"jobs":    jobs,
		"loading": false,
	})
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
