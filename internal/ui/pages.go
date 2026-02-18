// Package ui provides the Datastar-based web UI for plat-mjml.
package ui

import (
	"time"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	data "maragu.dev/gomponents-datastar"
)

// Layout wraps content in the base HTML layout.
func Layout(title string, content ...g.Node) g.Node {
	return h.HTML(
		h.Lang("en"),
		h.Head(
			h.Meta(h.Charset("utf-8")),
			h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1")),
			h.TitleEl(g.Text(title)),
			h.Script(h.Type("module"), h.Src("https://cdn.jsdelivr.net/gh/starfederation/datastar@1.0.0-RC.7/bundles/datastar.js")),
			h.StyleEl(h.Type("text/css"), g.Raw(styles)),
		),
		h.Body(
			h.Nav(h.Class("navbar"),
				h.Div(h.Class("nav-brand"), g.Text("plat-mjml")),
				h.Div(h.Class("nav-links"),
					h.A(h.Href("/"), g.Text("Dashboard")),
					h.A(h.Href("/templates"), g.Text("Templates")),
					h.A(h.Href("/queue"), g.Text("Queue")),
					h.A(h.Href("/send"), g.Text("Send")),
				),
			),
			h.Main(h.Class("container"), g.Group(content)),
			h.Footer(h.Class("footer"),
				g.Text("plat-mjml - MJML Email Platform"),
			),
		),
	)
}

// Dashboard renders the main dashboard page.
func Dashboard() g.Node {
	return Layout("Dashboard - plat-mjml",
		data.Signals(map[string]any{
			"stats":   map[string]int{},
			"loading": true,
		}),
		data.Init("@get('/api/stats')"),

		h.H1(g.Text("Email Dashboard")),

		// Stats cards
		h.Div(h.Class("stats-grid"),
			StatCard("pending", "Pending"),
			StatCard("processing", "Processing"),
			StatCard("sent", "Sent"),
			StatCard("retry", "Retry"),
			StatCard("failed", "Failed"),
			StatCard("scheduled", "Scheduled"),
		),

		// Quick actions
		h.Div(h.Class("section"),
			h.H2(g.Text("Quick Actions")),
			h.Div(h.Class("actions"),
				h.A(h.Href("/send"), h.Button(g.Text("Send Email"))),
				h.A(h.Href("/queue"), h.Button(g.Text("View Queue"))),
				h.A(h.Href("/templates"), h.Button(g.Text("Manage Templates"))),
			),
		),

		// Recent emails section with SSE updates
		h.Div(h.Class("section"),
			h.H2(g.Text("Recent Activity")),
			data.OnInterval("@get('/api/stats')", data.ModifierDuration, data.Duration(5*time.Second)),
			h.Div(h.ID("recent-list"),
				data.Show("!$loading"),
				h.P(g.Text("Stats loaded. Check queue for details.")),
			),
			h.Div(
				data.Show("$loading"),
				h.Span(h.Class("loading-spinner")),
				g.Text(" Loading..."),
			),
		),
	)
}

// StatCard renders a statistics card.
func StatCard(key, label string) g.Node {
	return h.Div(h.Class("stat-card"),
		h.Div(h.Class("stat-value"), data.Text("$stats."+key+" || 0")),
		h.Div(h.Class("stat-label"), g.Text(label)),
	)
}

// TemplatesPage renders the templates management page.
func TemplatesPage(templates []TemplateInfo) g.Node {
	var templateNodes []g.Node
	for _, t := range templates {
		slug := t.Slug
		templateNodes = append(templateNodes, h.Div(h.Class("template-item"),
			data.On("click", "$selected = '"+slug+"'; @get('/api/preview/"+slug+"')"),
			data.Class("active", "$selected === '"+slug+"'"),
			h.H3(g.Text(t.Slug)),
			h.P(g.Text(t.Description)),
		))
	}

	return Layout("Templates - plat-mjml",
		data.Signals(map[string]any{
			"selected":    "",
			"previewHtml": "",
			"loading":     false,
		}),

		h.H1(g.Text("Email Templates")),

		h.Div(h.Class("templates-grid"),
			// Template list
			h.Div(h.Class("template-list"),
				h.H2(g.Text("Available Templates")),
				g.Group(templateNodes),
			),

			// Preview panel
			h.Div(h.Class("preview-panel"),
				h.H2(g.Text("Preview")),
				h.Div(
					data.Show("$loading"),
					h.Span(h.Class("loading-spinner")),
					g.Text(" Loading preview..."),
				),
				h.Div(
					data.Show("!$loading && $previewHtml"),
					h.IFrame(
						h.ID("preview-frame"),
						data.Attr("srcdoc", "$previewHtml"),
						h.StyleAttr("width: 100%; height: 500px; border: 1px solid #ddd; border-radius: 8px;"),
					),
				),
				h.Div(
					data.Show("!$loading && !$previewHtml"),
					h.P(h.Class("hint"), g.Text("Select a template to preview")),
				),
			),
		),
	)
}

// QueuePage renders the queue monitoring page.
func QueuePage() g.Node {
	return Layout("Queue - plat-mjml",
		data.Signals(map[string]any{
			"jobs":    []any{},
			"filter":  "all",
			"loading": true,
		}),
		data.Init("@get('/api/queue')"),

		h.H1(g.Text("Email Queue")),

		// Filter buttons
		h.Div(h.Class("filter-bar"),
			h.Button(
				data.On("click", "$filter = 'all'; @get('/api/queue')"),
				data.Class("active", "$filter === 'all'"),
				g.Text("All"),
			),
			h.Button(
				data.On("click", "$filter = 'pending'; @get('/api/queue?status=pending')"),
				data.Class("active", "$filter === 'pending'"),
				g.Text("Pending"),
			),
			h.Button(
				data.On("click", "$filter = 'retry'; @get('/api/queue?status=retry')"),
				data.Class("active", "$filter === 'retry'"),
				g.Text("Retry"),
			),
			h.Button(
				data.On("click", "$filter = 'sent'; @get('/api/queue?status=sent')"),
				data.Class("active", "$filter === 'sent'"),
				g.Text("Sent"),
			),
			h.Button(
				data.On("click", "$filter = 'failed'; @get('/api/queue?status=failed')"),
				data.Class("active", "$filter === 'failed'"),
				g.Text("Failed"),
			),
		),

		// Auto-refresh toggle
		h.Div(h.Class("refresh-bar"),
			data.OnInterval("@get('/api/queue?status=' + ($filter === 'all' ? '' : $filter))", data.ModifierDuration, data.Duration(5*time.Second)),
			g.Text("Auto-refresh: 5s"),
		),

		// Queue list
		h.Div(h.Class("queue-list"),
			data.Show("$loading"),
			h.Div(h.Class("loading"),
				h.Span(h.Class("loading-spinner")),
				g.Text(" Loading queue..."),
			),
		),
		h.Div(h.ID("queue-items"),
			data.Show("!$loading"),
		),
	)
}

// SendEmailPage renders the send email form.
func SendEmailPage(templates []TemplateInfo) g.Node {
	var templateOptions []g.Node
	templateOptions = append(templateOptions, h.Option(h.Value(""), g.Text("Select template...")))
	for _, t := range templates {
		templateOptions = append(templateOptions, h.Option(h.Value(t.Slug), g.Text(t.Slug+" - "+t.Description)))
	}

	return Layout("Send Email - plat-mjml",
		data.Signals(map[string]any{
			"template": "",
			"to":       "",
			"subject":  "",
			"data":     "{}",
			"sending":  false,
			"result":   "",
		}),

		h.H1(g.Text("Send Email")),

		h.Form(h.Class("send-form"),
			data.On("submit", `
				event.preventDefault();
				$sending = true;
				@post('/api/send', {
					body: JSON.stringify({
						template: $template,
						to: $to.split(',').map(s => s.trim()),
						subject: $subject,
						data: JSON.parse($data || '{}')
					})
				})
			`),

			h.Div(h.Class("form-group"),
				h.Label(h.For("template"), g.Text("Template")),
				h.Select(h.ID("template"), data.Bind("template"),
					g.Group(templateOptions),
				),
			),

			h.Div(h.Class("form-group"),
				h.Label(h.For("to"), g.Text("Recipients (comma-separated)")),
				h.Input(h.ID("to"), h.Type("email"), data.Bind("to"),
					h.Placeholder("email@example.com, another@example.com"),
				),
			),

			h.Div(h.Class("form-group"),
				h.Label(h.For("subject"), g.Text("Subject")),
				h.Input(h.ID("subject"), h.Type("text"), data.Bind("subject"),
					h.Placeholder("Email subject"),
				),
			),

			h.Div(h.Class("form-group"),
				h.Label(h.For("data"), g.Text("Template Data (JSON)")),
				h.Textarea(h.ID("data"), data.Bind("data"),
					h.Placeholder(`{"name": "John", "email": "john@example.com"}`),
					h.Rows("5"),
				),
			),

			h.Button(h.Type("submit"),
				data.Attr("disabled", "$sending"),
				h.Span(data.Show("!$sending"), g.Text("Send Email")),
				h.Span(data.Show("$sending"),
					h.Span(h.Class("loading-spinner")),
					g.Text(" Sending..."),
				),
			),

			h.Div(h.Class("result"),
				data.Show("$result"),
				data.Text("$result"),
			),
		),
	)
}

// TemplateInfo holds template metadata for the UI.
type TemplateInfo struct {
	Slug        string
	Description string
}

const styles = `
:root {
	--primary: #6366f1;
	--primary-dark: #4f46e5;
	--success: #10b981;
	--warning: #f59e0b;
	--danger: #ef4444;
	--bg: #f8fafc;
	--card-bg: #ffffff;
	--text: #1e293b;
	--text-muted: #64748b;
	--border: #e2e8f0;
}

* {
	box-sizing: border-box;
	margin: 0;
	padding: 0;
}

body {
	font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
	background: var(--bg);
	color: var(--text);
	line-height: 1.6;
}

.navbar {
	background: var(--primary);
	color: white;
	padding: 1rem 2rem;
	display: flex;
	justify-content: space-between;
	align-items: center;
	box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.nav-brand {
	font-size: 1.5rem;
	font-weight: bold;
}

.nav-links a {
	color: white;
	text-decoration: none;
	margin-left: 2rem;
	opacity: 0.9;
	transition: opacity 0.2s;
}

.nav-links a:hover {
	opacity: 1;
}

.container {
	max-width: 1200px;
	margin: 0 auto;
	padding: 2rem;
}

.footer {
	text-align: center;
	padding: 2rem;
	color: var(--text-muted);
	border-top: 1px solid var(--border);
	margin-top: 2rem;
}

h1 {
	margin-bottom: 1.5rem;
	color: var(--text);
}

h2 {
	margin-bottom: 1rem;
	color: var(--text);
	font-size: 1.25rem;
}

.stats-grid {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
	gap: 1.5rem;
	margin-bottom: 2rem;
}

.stat-card {
	background: var(--card-bg);
	border-radius: 12px;
	padding: 1.5rem;
	text-align: center;
	box-shadow: 0 1px 3px rgba(0,0,0,0.1);
	border: 1px solid var(--border);
	transition: transform 0.2s, box-shadow 0.2s;
}

.stat-card:hover {
	transform: translateY(-2px);
	box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}

.stat-value {
	font-size: 2.5rem;
	font-weight: bold;
	color: var(--primary);
}

.stat-label {
	color: var(--text-muted);
	font-size: 0.875rem;
	text-transform: uppercase;
	letter-spacing: 0.05em;
}

.section {
	background: var(--card-bg);
	border-radius: 12px;
	padding: 1.5rem;
	margin-bottom: 1.5rem;
	border: 1px solid var(--border);
}

.actions {
	display: flex;
	gap: 1rem;
	flex-wrap: wrap;
}

button {
	background: var(--primary);
	color: white;
	border: none;
	padding: 0.75rem 1.5rem;
	border-radius: 8px;
	cursor: pointer;
	font-size: 1rem;
	font-weight: 500;
	transition: background 0.2s, transform 0.1s;
}

button:hover {
	background: var(--primary-dark);
}

button:active {
	transform: scale(0.98);
}

button:disabled {
	background: var(--text-muted);
	cursor: not-allowed;
}

button.active {
	background: var(--primary-dark);
	box-shadow: inset 0 2px 4px rgba(0,0,0,0.2);
}

.templates-grid {
	display: grid;
	grid-template-columns: 300px 1fr;
	gap: 1.5rem;
}

.template-list {
	background: var(--card-bg);
	border-radius: 12px;
	padding: 1rem;
	border: 1px solid var(--border);
}

.template-item {
	padding: 1rem;
	border-radius: 8px;
	cursor: pointer;
	transition: background 0.2s;
	border: 1px solid transparent;
}

.template-item:hover {
	background: var(--bg);
}

.template-item.active {
	background: var(--primary);
	color: white;
	border-color: var(--primary-dark);
}

.template-item.active p {
	color: rgba(255,255,255,0.8);
}

.template-item h3 {
	font-size: 1rem;
	margin-bottom: 0.25rem;
}

.template-item p {
	font-size: 0.875rem;
	color: var(--text-muted);
}

.preview-panel {
	background: var(--card-bg);
	border-radius: 12px;
	padding: 1.5rem;
	border: 1px solid var(--border);
}

.hint {
	color: var(--text-muted);
	font-style: italic;
}

.filter-bar {
	display: flex;
	gap: 0.5rem;
	margin-bottom: 1rem;
}

.refresh-bar {
	color: var(--text-muted);
	font-size: 0.875rem;
	margin-bottom: 1rem;
}

.queue-list {
	background: var(--card-bg);
	border-radius: 12px;
	border: 1px solid var(--border);
}

.loading {
	padding: 2rem;
	text-align: center;
	color: var(--text-muted);
}

.loading-spinner {
	display: inline-block;
	width: 16px;
	height: 16px;
	border: 2px solid var(--border);
	border-top-color: var(--primary);
	border-radius: 50%;
	animation: spin 1s linear infinite;
}

@keyframes spin {
	to { transform: rotate(360deg); }
}

.send-form {
	max-width: 600px;
	background: var(--card-bg);
	border-radius: 12px;
	padding: 2rem;
	border: 1px solid var(--border);
}

.form-group {
	margin-bottom: 1.5rem;
}

.form-group label {
	display: block;
	margin-bottom: 0.5rem;
	font-weight: 500;
}

.form-group input,
.form-group select,
.form-group textarea {
	width: 100%;
	padding: 0.75rem;
	border: 1px solid var(--border);
	border-radius: 8px;
	font-size: 1rem;
	transition: border-color 0.2s, box-shadow 0.2s;
}

.form-group input:focus,
.form-group select:focus,
.form-group textarea:focus {
	outline: none;
	border-color: var(--primary);
	box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
}

.result {
	margin-top: 1rem;
	padding: 1rem;
	border-radius: 8px;
	background: var(--bg);
}

@media (max-width: 768px) {
	.templates-grid {
		grid-template-columns: 1fr;
	}

	.nav-links a {
		margin-left: 1rem;
	}
}
`
