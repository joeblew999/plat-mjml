// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"github.com/joeblew999/plat-mjml/internal/config"
	"github.com/joeblew999/plat-mjml/pkg/mjml"
	"github.com/joeblew999/plat-mjml/pkg/queue"
)

type ServiceContext struct {
	Config   config.Config
	Renderer *mjml.Renderer
	Queue    *queue.Queue
}

func NewServiceContext(c config.Config, renderer *mjml.Renderer, q *queue.Queue) *ServiceContext {
	return &ServiceContext{
		Config:   c,
		Renderer: renderer,
		Queue:    q,
	}
}
