// MJML CLI - Email template rendering tool
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/joeblew999/plat-mjml/pkg/mail"
	"github.com/joeblew999/plat-mjml/pkg/mjml"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "render":
		renderCmd(os.Args[2:])
	case "validate":
		validateCmd(os.Args[2:])
	case "send":
		sendCmd(os.Args[2:])
	case "list":
		listCmd(os.Args[2:])
	case "version":
		fmt.Println("mjml v0.1.0")
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`MJML - Email Template CLI

Usage:
  mjml <command> [options]

Commands:
  render     Render MJML templates to HTML
  validate   Validate HTML for email client compatibility
  send       Send a test email via SMTP
  list       List available templates
  version    Show version
  help       Show this help

Examples:
  mjml render -template=welcome -out=email.html
  mjml validate -file=email.html
  mjml send -to=test@example.com -file=email.html
  mjml list -dir=./templates

Environment Variables:
  DATA_PATH           Base data directory (default: ./.data)
  MJML_TEMPLATE_PATH  Template directory (default: $DATA_PATH/templates)
  GMAIL_USERNAME      Gmail username for sending
  GMAIL_APP_PASSWORD  Gmail app password for sending`)
}

func renderCmd(args []string) {
	fs := flag.NewFlagSet("render", flag.ExitOnError)
	templateName := fs.String("template", "", "Template name to render")
	templateDir := fs.String("dir", "./templates", "Template directory")
	outFile := fs.String("out", "", "Output file (default: stdout)")
	dataFile := fs.String("data", "", "JSON data file for template")
	fs.Parse(args)

	if *templateName == "" {
		fmt.Println("Error: -template is required")
		os.Exit(1)
	}

	renderer := mjml.NewRenderer(
		mjml.WithTemplateDir(*templateDir),
		mjml.WithCache(false),
	)

	if err := renderer.LoadTemplatesFromDir(*templateDir); err != nil {
		fmt.Printf("Error loading templates: %v\n", err)
		os.Exit(1)
	}

	// Use test data if no data file provided
	var data any
	if *dataFile != "" {
		content, err := os.ReadFile(*dataFile)
		if err != nil {
			fmt.Printf("Error reading data file: %v\n", err)
			os.Exit(1)
		}
		data = string(content)
	} else {
		testData := mjml.TestData()
		if d, ok := testData[*templateName]; ok {
			data = d
		} else {
			data = testData["simple"]
		}
	}

	html, err := renderer.RenderTemplate(*templateName, data)
	if err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
		os.Exit(1)
	}

	if *outFile != "" {
		if err := os.WriteFile(*outFile, []byte(html), 0644); err != nil {
			fmt.Printf("Error writing output: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Rendered to %s (%d bytes)\n", *outFile, len(html))
	} else {
		fmt.Println(html)
	}
}

func validateCmd(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	file := fs.String("file", "", "HTML file to validate")
	fs.Parse(args)

	if *file == "" {
		fmt.Println("Error: -file is required")
		os.Exit(1)
	}

	content, err := os.ReadFile(*file)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	issues := mail.ValidateHTML(string(content))
	if len(issues) == 0 {
		fmt.Printf("✓ %s - No compatibility issues found\n", *file)
	} else {
		fmt.Printf("⚠ %s - Found %d issue(s):\n", *file, len(issues))
		for _, issue := range issues {
			fmt.Printf("  • %s\n", issue)
		}
		os.Exit(1)
	}
}

func sendCmd(args []string) {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	to := fs.String("to", "", "Recipient email address")
	file := fs.String("file", "", "HTML file to send")
	subject := fs.String("subject", "MJML Test Email", "Email subject")
	fs.Parse(args)

	if *to == "" || *file == "" {
		fmt.Println("Error: -to and -file are required")
		os.Exit(1)
	}

	smtpCfg := mail.Config{
		SMTPHost:  "smtp.gmail.com",
		SMTPPort:  "587",
		Username:  os.Getenv("GMAIL_USERNAME"),
		Password:  os.Getenv("GMAIL_APP_PASSWORD"),
		FromEmail: os.Getenv("GMAIL_USERNAME"),
		FromName:  "MJML Email",
	}
	if smtpCfg.Username == "" || smtpCfg.Password == "" {
		fmt.Println("Error: GMAIL_USERNAME and GMAIL_APP_PASSWORD environment variables required")
		os.Exit(1)
	}

	content, err := os.ReadFile(*file)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	if err := mail.Send(smtpCfg, *to, *subject, string(content)); err != nil {
		fmt.Printf("Error sending email: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Email sent to %s\n", *to)
}

func listCmd(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	dir := fs.String("dir", "./templates", "Template directory")
	fs.Parse(args)

	entries, err := os.ReadDir(*dir)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Templates in %s:\n", *dir)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".mjml") {
			name := strings.TrimSuffix(entry.Name(), ".mjml")
			info, _ := entry.Info()
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			fmt.Printf("  • %s (%s)\n", name, formatBytes(size))
		}
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

