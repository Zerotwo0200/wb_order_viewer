package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/example/wb-order-service/internal/usecase"
	"github.com/gorilla/mux"
)

type Server struct {
	Router *mux.Router
	UCGet  usecase.GetOrderByID
}

func NewServer(uc usecase.GetOrderByID) *Server {
	s := &Server{Router: mux.NewRouter(), UCGet: uc}
	s.Router.HandleFunc("/api/order/{id}", s.handleGet).Methods(http.MethodGet)
	s.Router.PathPrefix("/").Handler(http.FileServer(http.Dir("web")))
	return s
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	o, ok := s.UCGet.Execute(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(o)
}
