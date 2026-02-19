package svc

import (
	"fmt"
	"time"

	"github.com/joeblew999/plat-mjml/internal/config"
	"github.com/joeblew999/plat-mjml/internal/model"
	"github.com/joeblew999/plat-mjml/internal/db"
	"github.com/joeblew999/plat-mjml/internal/delivery"
	"github.com/joeblew999/plat-mjml/internal/events"
	"github.com/joeblew999/plat-mjml/pkg/mail"
	"github.com/joeblew999/plat-mjml/internal/mjml"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config             config.Config
	EmailsModel        model.EmailsModel
	TemplatesModel     model.TemplatesModel
	EmailEventsModel   model.EmailEventsModel
	SmtpProvidersModel model.SmtpProvidersModel
	Conn               sqlx.SqlConn
	Renderer           *mjml.Renderer
	DeliveryEngine     *delivery.Engine
	Events             *events.EventRecorder
	DB                 *db.DB
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

	// Create go-zero sqlx.SqlConn for circuit breaking + tracing
	conn := database.SqlConn()

	// Initialize go-zero models
	emailsModel := model.NewEmailsModel(conn)
	templatesModel := model.NewTemplatesModel(conn)
	emailEventsModel := model.NewEmailEventsModel(conn)
	smtpProvidersModel := model.NewSmtpProvidersModel(conn)

	// Event recorder (batched BulkInserter for email_events)
	eventRecorder, _ := events.NewEventRecorder(conn)

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

	deliveryEngine := delivery.NewEngine(emailsModel, eventRecorder, renderer, smtpConfig, delivery.Config{
		MaxRetries:   c.Delivery.MaxRetries,
		RetryBackoff: retryBackoff,
		MaxBackoff:   maxBackoff,
		RateLimit:    c.Delivery.RateLimit,
	})

	return &ServiceContext{
		Config:             c,
		EmailsModel:        emailsModel,
		TemplatesModel:     templatesModel,
		EmailEventsModel:   emailEventsModel,
		SmtpProvidersModel: smtpProvidersModel,
		Conn:               conn,
		Renderer:           renderer,
		DeliveryEngine:     deliveryEngine,
		Events:             eventRecorder,
		DB:                 database,
	}
}

// Close releases all resources held by the ServiceContext.
func (s *ServiceContext) Close() {
	if s.Events != nil {
		logx.Info("Flushing email events")
		s.Events.Flush()
	}
	logx.Info("Closing database")
	s.DB.Close()
}
