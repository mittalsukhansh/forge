package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strconv"

	"github.com/mittalsukhansh/forge/internal/pipeline"
	"github.com/mittalsukhansh/forge/internal/queue"
	"github.com/mittalsukhansh/forge/internal/store"
)

type API struct {
	store *store.Store
	queue *queue.Queue
}

func New(s *store.Store, q *queue.Queue) http.Handler {
	a := &API{store: s, queue: q}

	mux := http.NewServeMux()
	registerDashboard(mux)

	mux.HandleFunc("GET /health", a.health)
	mux.HandleFunc("GET /api/jobs", a.list)
	mux.HandleFunc("POST /api/jobs", a.submit)
	mux.HandleFunc("GET /api/jobs/{id}", a.get)

	return mux
}

func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func fail(w http.ResponseWriter, status int, message string) {
	respond(w, status, map[string]string{
		"error": message,
	})
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	respond(w, 200, map[string]string{
		"application": "Forge",
		"version":     "0.2.0",
		"status":      "healthy",
	})
}

func (a *API) submit(w http.ResponseWriter, r *http.Request) {
	contentType, _, err := mime.ParseMediaType(
		r.Header.Get("Content-Type"),
	)

	if err != nil || contentType != "application/json" {
		fail(w, 415, "Content-Type must be application/json")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 65536)

	var body struct {
		Pipeline string `json:"pipeline"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fail(w, 400, "Invalid request body")
		return
	}

	p, err := pipeline.Parse([]byte(body.Pipeline))

	if err != nil {
		fail(w, 400, err.Error())
		return
	}

	id, err := a.queue.Submit(p)

	if errors.Is(err, queue.ErrQueueFull) {
		fail(w, 503, err.Error())
		return
	}

	if err != nil {
		fail(w, 500, "Unable to queue job")
		return
	}

	respond(w, 202, map[string]any{
		"id":     id,
		"status": "queued",
	})
}

func (a *API) list(w http.ResponseWriter, r *http.Request) {
	jobs, err := a.store.List()

	if err != nil {
		fail(w, 500, "Unable to retrieve jobs")
		return
	}

	respond(w, 200, jobs)
}

func (a *API) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)

	if err != nil || id < 1 {
		fail(w, 400, "Invalid job ID")
		return
	}

	job, err := a.store.Get(id)

	if errors.Is(err, sql.ErrNoRows) {
		fail(w, 404, "Job not found")
		return
	}

	if err != nil {
		fail(w, 500, "Unable to retrieve job")
		return
	}

	respond(w, 200, job)
}
