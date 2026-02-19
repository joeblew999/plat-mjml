package template

var descriptions = map[string]string{
	"simple":                "Basic email template",
	"welcome":               "Welcome/activation email for new users",
	"reset_password":        "Password reset email with security info",
	"notification":          "System notification email",
	"premium_newsletter":    "Newsletter with premium fonts",
	"business_announcement": "Business announcement email",
}

func templateDescription(slug string) string {
	if desc, ok := descriptions[slug]; ok {
		return desc
	}
	return "Email template"
}
