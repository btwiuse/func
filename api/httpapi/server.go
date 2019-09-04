package httpapi

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/func/func/api"
	"go.uber.org/zap"
)

// An Error is a json encoded error message from the api.
type Error struct {
	Msg string `json:"message"`
}

// Server is func http api server.
type Server struct {
	API    api.API
	Logger *zap.Logger

	once   sync.Once
	router *http.ServeMux
}

func (s *Server) setupRoutes() {
	s.router = http.NewServeMux()
	s.router.HandleFunc("/apply", s.handleApply())
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.once.Do(s.setupRoutes)
	s.router.ServeHTTP(w, r)
}

func (s *Server) respond(w http.ResponseWriter, data interface{}, status int) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.Logger.Error("Could not encode json response", zap.Error(err))
		http.Error(w, "Could not encode response", http.StatusInternalServerError)
	}
}

func (s *Server) decode(w http.ResponseWriter, r *http.Request, v interface{}) error { // nolint: unparam
	return json.NewDecoder(r.Body).Decode(v)
}

func (s *Server) handleApply() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			s.respond(w, Error{Msg: "Method not allowed"}, http.StatusMethodNotAllowed)
			return
		}

		if r.Body == nil {
			s.Logger.Debug("Body not set")
			s.respond(w, Error{Msg: "No body"}, http.StatusBadRequest)
			return
		}

		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			s.respond(w, Error{Msg: "Invalid content type"}, http.StatusUnsupportedMediaType)
			return
		}

		var body applyRequest
		if err := s.decode(w, r, &body); err != nil {
			s.Logger.Debug("Could not decode body", zap.Error(err))
			s.respond(w, Error{Msg: "Could not decode body"}, http.StatusBadRequest)
			return
		}
		_ = r.Body.Close()

		apireq := &api.ApplyRequest{
			Project: body.Project,
			Config:  body.Config,
		}

		apiresp, err := s.API.Apply(r.Context(), apireq)
		if err != nil {
			s.Logger.Debug("Apply error", zap.Error(err))
			aerr, ok := err.(*api.Error)
			if ok {
				if aerr.Diagnostics.HasErrors() {
					response := &applyResponse{
						Diagnostics: diagsFromHCL(aerr.Diagnostics),
					}
					s.respond(w, response, http.StatusBadRequest)
					return
				}
				var status int
				switch aerr.Code {
				case api.ValidationError:
					status = http.StatusBadRequest
				case api.Unavailable:
					status = http.StatusServiceUnavailable
				default:
					// Unknown error
					status = http.StatusInternalServerError
				}
				s.respond(w, Error{Msg: aerr.Message}, status)
				return
			}
			// Unknown error
			s.respond(w, Error{Msg: "Could not apply changes"}, http.StatusInternalServerError)
			return
		}

		src := make([]*sourceRequest, len(apiresp.SourcesRequired))
		for i, s := range apiresp.SourcesRequired {
			src[i] = &sourceRequest{
				Key:     s.Key,
				URL:     s.URL,
				Headers: s.Headers,
			}
		}
		response := applyResponse{
			SourcesRequired: src,
		}

		s.respond(w, response, http.StatusOK)
	}
}
