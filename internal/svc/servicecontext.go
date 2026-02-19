// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"github.com/joeblew999/plat-mjml/pkg/mjml"
	"github.com/joeblew999/plat-mjml/pkg/queue"
)

type ServiceContext struct {
	Renderer *mjml.Renderer
	Queue    *queue.Queue
}

func NewServiceContext(renderer *mjml.Renderer, q *queue.Queue) *ServiceContext {
	return &ServiceContext{
		Renderer: renderer,
		Queue:    q,
	}
}
