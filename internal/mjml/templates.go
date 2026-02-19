package mjml

import (
	"time"
)

// EmailData represents common email template data
type EmailData struct {
	// Recipient information
	Name  string `json:"name"`
	Email string `json:"email"`

	// Email metadata
	Subject   string    `json:"subject"`
	Timestamp time.Time `json:"timestamp"`

	// Brand/company information
	CompanyName string `json:"company_name"`
	CompanyLogo string `json:"company_logo"`
	CompanyURL  string `json:"company_url"`

	// Content
	Title   string `json:"title"`
	Message string `json:"message"`

	// Action items
	ButtonText string `json:"button_text"`
	ButtonURL  string `json:"button_url"`

	// Font support (optional — used by all templates)
	FontCSS   string `json:"font_css,omitempty"`
	FontStack string `json:"font_stack,omitempty"`
}

// WelcomeEmailData extends EmailData for welcome emails
type WelcomeEmailData struct {
	EmailData
	ActivationURL string `json:"activation_url"`
	LoginURL      string `json:"login_url"`
}

// ResetPasswordData extends EmailData for password reset emails
type ResetPasswordData struct {
	EmailData
	ResetURL    string        `json:"reset_url"`
	ExpiresIn   time.Duration `json:"expires_in"`
	RequestIP   string        `json:"request_ip"`
	RequestTime time.Time     `json:"request_time"`
}

// NotificationData extends EmailData for notification emails
type NotificationData struct {
	EmailData
	NotificationType string                 `json:"notification_type"`
	Priority         string                 `json:"priority"`
	Details          map[string]any `json:"details"`
	ActionRequired   bool                   `json:"action_required"`
}

// NewsletterData extends EmailData for newsletter emails with optional font support
type NewsletterData struct {
	EmailData
	PreviewText        string              `json:"preview_text"`
	Subtitle           string              `json:"subtitle"`
	Greeting           string              `json:"greeting"`
	ContentBlocks      []string            `json:"content_blocks"`
	CallToActionURL    string              `json:"call_to_action_url"`
	CallToActionText   string              `json:"call_to_action_text"`
	HeroImage          string              `json:"hero_image"`
	HeroImageAlt       string              `json:"hero_image_alt"`
	FeaturedTitle      string              `json:"featured_title"`
	FeaturedContent    []FeaturedItem      `json:"featured_content"`
	SocialLinks        []SocialLink        `json:"social_links"`
	CompanyAddress     string              `json:"company_address"`
	UnsubscribeURL     string              `json:"unsubscribe_url"`
}

// FeaturedItem represents an item in featured content
type FeaturedItem struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// SocialLink represents a social media link
type SocialLink struct {
	Platform string `json:"platform"` // twitter, facebook, instagram, linkedin, etc.
	URL      string `json:"url"`
}

// TemplateDescription returns a human-readable description for a template slug.
func TemplateDescription(slug string) string {
	desc, ok := templateDescriptions[slug]
	if !ok {
		return "Email template"
	}
	return desc
}

var templateDescriptions = map[string]string{
	"simple":                "Basic email template",
	"welcome":               "Welcome/activation email for new users",
	"reset_password":        "Password reset email with security info",
	"notification":          "System notification email",
	"premium_newsletter":    "Newsletter with premium fonts",
	"business_announcement": "Business announcement email",
}

// TestData provides canonical test data for all template types
func TestData() map[string]any {
	baseData := EmailData{
		Name:        "Test User",
		Email:       "test@example.com",
		Subject:     "Test Email",
		Title:       "Test Title",
		Message:     "This is a test message",
		ButtonText:  "Click Here",
		ButtonURL:   "https://example.com",
		Timestamp:   time.Now(),
		CompanyName: "Test Company",
		CompanyLogo: "https://via.placeholder.com/200x80/3498db/ffffff?text=LOGO",
		CompanyURL:  "https://testcompany.com",
		FontStack:   "'Inter', Arial, Helvetica, sans-serif",
	}

	return map[string]any{
		"simple": baseData,
		
		"welcome": WelcomeEmailData{
			EmailData:     baseData,
			ActivationURL: "https://testcompany.com/activate?token=test123",
			LoginURL:      "https://testcompany.com/login",
		},
		
		"reset_password": ResetPasswordData{
			EmailData:   baseData,
			ResetURL:    "https://testcompany.com/reset?token=test456",
			ExpiresIn:   24 * time.Hour,
			RequestIP:   "192.168.1.1",
			RequestTime: time.Now(),
		},
		
		"notification": NotificationData{
			EmailData:        baseData,
			NotificationType: "system",
			Priority:         "high",
			ActionRequired:   true,
			Details: map[string]any{
				"server": "test-server",
				"metric": "CPU usage",
			},
		},
		
		"premium_newsletter": NewsletterData{
			EmailData: EmailData{
				Name:        "Premium Subscriber",
				Email:       "subscriber@example.com",
				Subject:     "Premium Newsletter - January 2024",
				Title:       "Premium Newsletter",
				CompanyName: "Premium Content Co.",
				CompanyLogo: "https://via.placeholder.com/180x70/4299e1/ffffff?text=PREMIUM",
				Timestamp:   time.Now(),
				FontStack:   "'Inter', Arial, Helvetica, sans-serif",
			},
			PreviewText:      "Your monthly dose of premium content",
			Subtitle:         "January 2024 Edition",
			Greeting:         "Hello Premium Subscriber,",
			ContentBlocks: []string{
				"Welcome to our January edition! We're excited to share the latest insights.",
				"This month, we're focusing on emerging trends in technology.",
			},
			CallToActionURL:  "https://premium.example.com/january-2024",
			CallToActionText: "Read Full Edition",
			FeaturedTitle:    "This Month's Highlights",
			FeaturedContent: []FeaturedItem{
				{
					Title:       "Market Analysis Report",
					Description: "Deep dive into Q4 market trends",
					URL:         "https://premium.example.com/market-analysis",
				},
			},
			SocialLinks: []SocialLink{
				{Platform: "twitter", URL: "https://twitter.com/premium"},
				{Platform: "linkedin", URL: "https://linkedin.com/company/premium"},
			},
			CompanyAddress: "123 Premium St, NY 10001",
			UnsubscribeURL: "https://premium.example.com/unsubscribe?token=abc123",
		},

		"business_announcement": map[string]any{
			"subject":              "Grand Opening Announcement",
			"preview":              "You're invited to our grand opening event",
			"company_name":         "Test Company",
			"company_logo":         "https://via.placeholder.com/150x60/040B4F/ffffff?text=COMPANY",
			"name":                 "Test User",
			"font_stack":           "'Inter', 'Helvetica Neue', Helvetica, Arial, sans-serif",
			"location":             "San Francisco, CA",
			"venue":                "Innovation Center",
			"address":              "123 Main Street, Suite 100",
			"title":                "Grand Opening Event",
			"message":              "We're thrilled to invite you to our grand opening celebration.",
			"description":          "Join us for an evening of networking, demos, and refreshments.",
			"primary_button_text":  "RSVP Now",
			"primary_button_url":   "https://testcompany.com/rsvp",
			"call_to_action_text":  "We hope to see you there!",
			"closing_message":      "Best regards, The Team",
			"visit_title":          "Visit Us",
			"visit_message":        "We look forward to welcoming you to our new location.",
			"hours": []map[string]any{
				{"day": "Monday - Friday", "time": "9:00 AM - 6:00 PM"},
				{"day": "Saturday", "time": "10:00 AM - 4:00 PM"},
			},
			"disclaimer":      "You are receiving this email because you signed up for updates.",
			"privacy_url":     "https://testcompany.com/privacy",
			"unsubscribe_url": "https://testcompany.com/unsubscribe",
		},
	}
}