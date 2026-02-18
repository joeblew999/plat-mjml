package mail

import (
	"strings"
)

// ValidateHTML performs basic HTML validation for email client compatibility.
func ValidateHTML(htmlContent string) []string {
	var issues []string

	if !strings.Contains(strings.ToLower(htmlContent), "doctype html") {
		issues = append(issues, "Missing DOCTYPE declaration")
	}

	if !strings.Contains(htmlContent, "xmlns:v=\"urn:schemas-microsoft-com:vml\"") {
		issues = append(issues, "Missing VML namespace for Outlook compatibility")
	}

	if !strings.Contains(htmlContent, "<!--[if mso") {
		issues = append(issues, "Missing Outlook conditional comments")
	}

	if !strings.Contains(htmlContent, "border-collapse:collapse") && !strings.Contains(htmlContent, "border-collapse: collapse") {
		issues = append(issues, "Missing border-collapse for table compatibility")
	}

	if strings.Contains(htmlContent, "display: flex") {
		issues = append(issues, "WARNING: CSS flexbox not supported in many email clients")
	}

	if strings.Contains(htmlContent, "background-image") && !strings.Contains(htmlContent, "mso-hide") {
		issues = append(issues, "WARNING: Background images not supported in Outlook")
	}

	return issues
}
