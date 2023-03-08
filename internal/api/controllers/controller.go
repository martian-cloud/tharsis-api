package controllers

import (
	"github.com/go-chi/chi/v5"
)

// Controller encapsulates the logic for registering handler functions
type Controller interface {
	// RegisterRoutes adds controller handlers to the router
	RegisterRoutes(router chi.Router)
}
