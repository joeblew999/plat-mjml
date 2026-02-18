package server

import "github.com/joeblew999/plat-mjml/pkg/delivery"

// deliveryService adapts delivery.Engine to the service.Service interface.
type deliveryService struct {
	engine  *delivery.Engine
	workers int
}

func newDeliveryService(engine *delivery.Engine, workers int) *deliveryService {
	return &deliveryService{engine: engine, workers: workers}
}

func (s *deliveryService) Start() {
	s.engine.Start(s.workers)
}

func (s *deliveryService) Stop() {
	s.engine.Stop()
}
