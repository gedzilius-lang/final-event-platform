// Package httputil provides standardised JSON response helpers for NiteOS services.
// All services use these helpers for consistent error and success response formats.
package httputil

import (
	"encoding/json"
	"net/http"
)

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error writes a standardised error response.
// Format: {"error": "<code>", "message": "<human-readable>"}
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}

// OK writes a 200 JSON response.
func OK(w http.ResponseWriter, v any) {
	JSON(w, http.StatusOK, v)
}

// Created writes a 201 JSON response.
func Created(w http.ResponseWriter, v any) {
	JSON(w, http.StatusCreated, v)
}

// NoContent writes a 204 response with no body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Health is the standard health check response written by every service's GET /healthz.
type Health struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}

// Healthz writes a standard 200 health response.
func Healthz(w http.ResponseWriter, service, version string) {
	OK(w, Health{Status: "ok", Service: service, Version: version})
}

// Respond writes a JSON response — alias used by service handlers.
func Respond(w http.ResponseWriter, status int, v any) {
	JSON(w, status, v)
}

// RespondError writes a JSON error response — alias used by service handlers.
func RespondError(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}
