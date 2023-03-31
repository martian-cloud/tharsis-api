package controllers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
)

type health struct {
	Ok bool `json:"ok"`
} //@name Health

var healthy = health{Ok: true}

type healthController struct {
	respWriter response.Writer
}

// NewHealthController creates an instance of HealthController
func NewHealthController(respWriter response.Writer) Controller {
	return &healthController{respWriter}
}

// RegisterRoutes adds health routes to the router
func (c *healthController) RegisterRoutes(router chi.Router) {
	router.Get("/health", c.GetHealth)
}

// GetHealth godoc
// @Summary Get health of API
// @Description Get health of API
// @Tags health
// @Accept  json
// @Produce  json
// @Success 200 {object} Health
// @Router /health [get]
func (c *healthController) GetHealth(w http.ResponseWriter, _ *http.Request) {
	c.respWriter.RespondWithJSON(w, healthy, 200)
}
