package svc

import (
	"fmt"
	"time"

	"github.com/joeblew999/plat-mjml/internal/config"
	"github.com/joeblew999/plat-mjml/pkg/db"
	"github.com/joeblew999/plat-mjml/pkg/delivery"
	"github.com/joeblew999/plat-mjml/pkg/mail"
	"github.com/joeblew999/plat-mjml/pkg/mjml"
	"github.com/joeblew999/plat-mjml/pkg/queue"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
)

type ServiceContext struct {
	Config         config.Config
	Renderer       *mjml.Renderer
	Queue          *queue.Queue
	DB             *db.DB
	DeliveryEngine *delivery.Engine
}

func NewServiceContext(c config.Config) *ServiceContext {
	// Parallel initialization: template loading and database opening are independent
	var renderer *mjml.Renderer
	var database *db.DB

	err := mr.Finish(
		func() error {
			renderer = mjml.NewRenderer(
				mjml.WithTemplateDir(c.Templates.Dir),
				mjml.WithFontDir(c.Fonts.Dir),
				mjml.WithCache(true),
			)
			return renderer.LoadTemplatesFromDir(c.Templates.Dir)
		},
		func() error {
			var e error
			database, e = db.Open(c.Database.Path)
			return e
		},
	)
	if err != nil {
		logx.Must(fmt.Errorf("failed to initialize: %w", err))
	}

	// Create queue using go-zero sqlx.SqlConn for circuit breaking + tracing
	conn := database.SqlConn()
	emailQueue := queue.NewQueue(conn)

	// Parse delivery config
	retryBackoff, _ := time.ParseDuration(c.Delivery.RetryBackoff)
	if retryBackoff == 0 {
		retryBackoff = 5 * time.Minute
	}
	maxBackoff, _ := time.ParseDuration(c.Delivery.MaxBackoff)
	if maxBackoff == 0 {
		maxBackoff = 4 * time.Hour
	}

	smtpConfig := mail.Config{
		SMTPHost:  c.SMTP.Host,
		SMTPPort:  c.SMTP.Port,
		Username:  c.SMTP.Username,
		Password:  c.SMTP.Password,
		FromEmail: c.SMTP.FromEmail,
		FromName:  c.SMTP.FromName,
	}

	deliveryEngine := delivery.NewEngine(emailQueue, renderer, smtpConfig, delivery.Config{
		MaxRetries:   c.Delivery.MaxRetries,
		RetryBackoff: retryBackoff,
		MaxBackoff:   maxBackoff,
		RateLimit:    c.Delivery.RateLimit,
	})

	return &ServiceContext{
		Config:         c,
		Renderer:       renderer,
		Queue:          emailQueue,
		DB:             database,
		DeliveryEngine: deliveryEngine,
	}
}

// Close releases all resources held by the ServiceContext.
func (s *ServiceContext) Close() {
	if s.Queue.Events != nil {
		logx.Info("Flushing email events")
		s.Queue.Events.Flush()
	}
	logx.Info("Closing database")
	s.DB.Close()
}
