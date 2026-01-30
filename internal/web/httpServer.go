package web

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/AlekseyZapadovnikov/L0_DemoService/internal/entity"
	"github.com/AlekseyZapadovnikov/L0_DemoService/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type OrderGiver interface {
	GiveOrderByUID(UID string) (entity.Order, error)
}

type Server struct {
	router  *chi.Mux
	server  *http.Server
	service OrderGiver
}

//go:embed static/*
var staticFS embed.FS

func New(addr string, service OrderGiver) *Server {
	router := chi.NewRouter()

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	server := &Server{
		router:  router,
		server:  srv,
		service: service,
	}

	return server
}

func (s *Server) Start() error {
	s.routes()
	return s.server.ListenAndServe()
}

func (s *Server) routes() {
	s.router.Use(middleware.RequestID)
	s.router.Use(HTTPLogger)

	s.router.Get("/order/{UID}", s.handleOrderByUID)

	content, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(content))

	s.router.Handle("/*", fileServer)
}

func (s *Server) handleOrderByUID(rw http.ResponseWriter, r *http.Request) {
	uid := chi.URLParam(r, "UID")

	if _, err := uuid.Parse(uid); err != nil {
		http.Error(rw, "UID is invalid or missing", http.StatusBadRequest)
		return
	}

	order, err := s.service.GiveOrderByUID(uid)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			http.Error(rw, "order not found", http.StatusNotFound)
			return
		}
		http.Error(rw, "internal server error", http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(&order)
	if err != nil {
		slog.Warn("couldn`t marshal json", slog.String("error", err.Error()), slog.String("UID", uid))
		http.Error(rw, "internal server error", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	if _, err := rw.Write(data); err != nil {
		slog.Warn("couldn`t write response", slog.String("error", err.Error()), slog.String("UID", uid))
		http.Error(rw, "internal server error", http.StatusInternalServerError)
	}
}
