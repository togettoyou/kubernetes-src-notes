package extender

import (
	"context"
	"encoding/json"
	"net/http"

	extenderapi "k8s.io/kube-scheduler/extender/v1"
)

// Handler 封装 Filter、Prioritize 、ProcessPreemption 和 Bind 阶段的入参和出参方法
type Handler interface {
	Filter(ctx context.Context, args extenderapi.ExtenderArgs) (*extenderapi.ExtenderFilterResult, error)
	Prioritize(ctx context.Context, args extenderapi.ExtenderArgs) (*extenderapi.HostPriorityList, error)
	ProcessPreemption(ctx context.Context, args extenderapi.ExtenderPreemptionArgs) (*extenderapi.ExtenderPreemptionResult, error)
	Bind(ctx context.Context, args extenderapi.ExtenderBindingArgs) (*extenderapi.ExtenderBindingResult, error)
}

type Server struct {
	handler Handler
}

func NewServer(handler Handler) *Server {
	return &Server{handler: handler}
}

func (s *Server) Filter() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var args extenderapi.ExtenderArgs
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := s.handler.Filter(r.Context(), args)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, res)
	}
}

func (s *Server) Prioritize() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var args extenderapi.ExtenderArgs
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := s.handler.Prioritize(r.Context(), args)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, res)
	}
}

func (s *Server) ProcessPreemption() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var args extenderapi.ExtenderPreemptionArgs
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := s.handler.ProcessPreemption(r.Context(), args)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, res)
	}
}

func (s *Server) Bind() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var args extenderapi.ExtenderBindingArgs
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		res, err := s.handler.Bind(r.Context(), args)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, res)
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
