package mjml

import (
	"testing"
	"time"
)

func TestServiceIntegration(t *testing.T) {
	service := NewService("./templates")
	if service == nil {
		t.Fatal("Service creation failed")
	}

	// Load a test template directly instead of from directory
	service.renderer.LoadTemplate("simple", `<mjml>
		<mj-head><mj-title>{{.Subject}}</mj-title></mj-head>
		<mj-body>
			<mj-section>
				<mj-column>
					<mj-text>Hello {{.Name}}</mj-text>
					<mj-text>{{.Message}}</mj-text>
				</mj-column>
			</mj-section>
		</mj-body>
	</mjml>`)

	// Test service with loaded template (skip Start which requires directory)
	t.Log("Service initialized successfully")

	// Test that service can render templates
	data := EmailData{
		Name:    "Service Test User",
		Subject: "Service Test",
		Title:   "Test",
		Message: "Service integration test",
		ButtonText: "Test Button",
		ButtonURL:  "https://example.com",
	}

	html, err := service.renderer.RenderTemplate("simple", data)
	if err != nil {
		t.Fatalf("Service rendering failed: %v", err)
	}

	if len(html) == 0 {
		t.Error("Service generated empty HTML")
	}

	// Test service shutdown
	err = service.Stop()
	if err != nil {
		t.Errorf("Service stop failed: %v", err)
	}
}

func TestServiceConcurrency(t *testing.T) {
	service := NewService("./templates")
	if service == nil {
		t.Fatal("Service creation failed")
	}

	// Load test template directly
	service.renderer.LoadTemplate("simple", `<mjml>
		<mj-head><mj-title>{{.Subject}}</mj-title></mj-head>
		<mj-body>
			<mj-section>
				<mj-column>
					<mj-text>Hello {{.Name}}</mj-text>
				</mj-column>
			</mj-section>
		</mj-body>
	</mjml>`)

	// Skip Start which requires directory - test concurrency directly
	defer service.Stop()

	// Test concurrent rendering
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			data := EmailData{
				Name:    "Concurrent Test User",
				Subject: "Concurrent Test",
				Title:   "Test",
				Message: "Concurrent rendering test",
				ButtonText: "Test Button",
				ButtonURL:  "https://example.com",
			}

			_, err := service.renderer.RenderTemplate("simple", data)
			if err != nil {
				t.Errorf("Concurrent rendering failed: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	timeout := time.After(5 * time.Second)
	completed := 0
	
	for completed < 10 {
		select {
		case <-done:
			completed++
		case <-timeout:
			t.Fatal("Concurrent test timed out")
		}
	}
}