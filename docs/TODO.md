# todo

for github.com/preslavrachev/gomjml/mjml

## MJML Email Templates

Google Fonts integration needed for email templates. The current MJML templates use basic web fonts but should integrate with pkg/font for proper font management and self-hosted font delivery to ensure email client compatibility.

- [ ] Integrate MJML renderer with pkg/font package
- [ ] Add self-hosted Google Fonts for email templates  
- [ ] Update MJML templates to use pkg/font font loading
- [ ] Test font rendering across different email clients
- [ ] Add font fallbacks for email client compatibility

This will ensure emails render consistently across Gmail, Outlook, Apple Mail, etc. without relying on external CDN font loading which many email clients block.


## RPC and API Integration

This is a realyl good test of the https://github.com/zeromicro/go-zero system. We can add this to the example.  And then make it prodcue and RPC API and a MCP, and then add it to Claude mcp to test it.


It has a nice cli: https://go-zero.dev/en/docs/tasks/installation/goctl



```bash
go get github.com/zeromicro/go-zero@latest
```

Create `main.go`:

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/mcp"
)

func main() {
	var c mcp.McpConf
	conf.MustLoad("config.yaml", &c)
	logx.DisableStat()

	s := mcp.NewMcpServer(c)
	defer s.Stop()

	// 1) Register a TOOL  -----------------
	s.RegisterTool(mcp.Tool{
		Name:        "calculator",
		Description: "Basic arithmetic",
		InputSchema: mcp.InputSchema{
			Properties: map[string]any{
				"op": {"type": "string", "enum": []string{"add", "sub", "mul", "div"}},
				"a":  {"type": "number"},
				"b":  {"type": "number"},
			},
			Required: []string{"op", "a", "b"},
		},
		Handler: func(ctx context.Context, p map[string]any) (any, error) {
			type req struct {
				Op string  `json:"op"`
				A  float64 `json:"a"`
				B  float64 `json:"b"`
			}
			var r req
			_ = mcp.ParseArguments(p, &r)

			switch r.Op {
			case "add":
				return r.A + r.B, nil
			case "sub":
				return r.A - r.B, nil
			case "mul":
				return r.A * r.B, nil
			case "div":
				if r.B == 0 {
					return nil, fmt.Errorf("divide by zero")
				}
				return r.A / r.B, nil
			}
			return nil, fmt.Errorf("unknown op")
		},
	})

	// 2) Register a DYNAMIC PROMPT --------
	s.RegisterPrompt(mcp.Prompt{
		Name:        "time-greeting",
		Description: "Greets user with current time",
		Arguments: []mcp.PromptArgument{
			{Name: "name", Required: true},
		},
		Handler: func(ctx context.Context, args map[string]string) ([]mcp.PromptMessage, error) {
			name := args["name"]
			return []mcp.PromptMessage{
				{Role: mcp.RoleUser, Content: mcp.TextContent{Text: "Hi, my name is " + name}},
				{Role: mcp.RoleAssistant, Content: mcp.TextContent{
					Text: fmt.Sprintf("Hello %s! The time is %s", name, time.Now().Format(time.RFC1123))}},
			}, nil
		},
	})

	// 3) Register a RESOURCE -------------
	s.RegisterResource(mcp.Resource{
		Name:        "version",
		URI:         "file:///version.txt",
		Description: "Build version",
		MimeType:    "text/plain",
		Handler: func(ctx context.Context) (mcp.ResourceContent, error) {
			return mcp.ResourceContent{
				URI:      "file:///version.txt",
				MimeType: "text/plain",
				Text:     "v1.0.0+git.abc123",
			}, nil
		},
	})

	fmt.Printf("🚀 MCP server listening on %s:%d …\n", c.Host, c.Port)
	s.Start()
}
```

Create `config.yaml`:

```yaml
name: mcp-demo
host: 0.0.0.0
port: 8080
mcp:
  name: demo-server
  messageTimeout: 30s
  cors:
    - http://localhost:3000
```

---

### 3. Run it

```bash
go run .
# 🚀 MCP server listening on 0.0.0.0:8080 …
```

then add to claude to test the MCP:

claude mcp add go-zero-demo -- npx -y mcp-remote http://localhost:8080/sse


### 5. What you just got

| MCP primitive | Provided by go-zero | Typical usage |
|---------------|---------------------|---------------|
| **Tools**     | `RegisterTool()`    | Let the LLM call backend functions (SQL, REST, etc.) |
| **Prompts**   | `RegisterPrompt()`  | Re-usable, parameterised conversation starters |
| **Resources** | `RegisterResource()`| Expose files / binary blobs to the model |
| **Transport** | Built-in SSE        | Real-time, bi-directional, no extra side-cars |

All concurrency, JSON-RPC framing, CORS, and timeout handling are taken care of by the SDK .



---

datastar:

Its xml based and so we can built a Datastar based editor, such that the XML merges into the Web GUI.

We will explore compiling this to WASM much later, so that the Editor can run 100% in the browser, once we get DataSTar system working with WASM.

---

https://github.com/ViBiOh/mailer

