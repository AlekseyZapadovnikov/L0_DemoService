package server

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Asus/L0_DemoServise/internal/entity"
)

type OrderGiver interface {
	GiveOrderByUID(UID string) (entity.Order, error)
}

type Server struct {
	router  *http.ServeMux
	server  *http.Server
	service OrderGiver
}

func New(addr string, OrdService OrderGiver) *Server {
	srv := &Server{
		router:  http.NewServeMux(),
		service: OrdService,
	}
	srv.server = &http.Server{
		Addr:    addr,
		Handler: srv,
	}
	srv.routes()
	return srv
}

// Start запускает сервер.
func (s *Server) Start() error {
	slog.Info("server starting", "address", s.server.Addr)
	return s.server.ListenAndServe()
}

// Для того чтобы не писать логирование в каждом HandleFunc логируем все тут
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slog.Info("request received", "method", r.Method, "path", r.URL.Path)
	s.router.ServeHTTP(w, r) // находим нужный хэндлер и вызываем
}

// эта функция заполняет наш маршрутизатор нужными хендлерами
func (s *Server) routes() {
    s.router.HandleFunc("GET /", s.handleHomePage())      
    s.router.HandleFunc("GET /order/{UID}", s.handleOrderByUID()) 
}


// handleHomePage() просто загружает домашнюю страницу html
var tmpl = template.Must(template.ParseGlob("internal/server/templates/*.html")) // загрузили все html

func (s *Server) handleHomePage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "homePage.html", nil)
	}
}

// выводит данные о заказе в браузер в формате json
func (s *Server) handleOrderByUID() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ожидаем URL вида: /order/<uid>
        uid := strings.TrimPrefix(r.URL.Path, "/order/")
        if uid == "" || strings.Contains(uid, "/") {
            http.NotFound(w, r)
            return
        }

        ord, err := s.service.GiveOrderByUID(uid)
        if err != nil {
            http.Error(w, "order not found", http.StatusNotFound)
            return
        }

        // Возвращаем JSON вместо рендеринга шаблона
        w.Header().Set("Content-Type", "application/json")
        if err := json.NewEncoder(w).Encode(ord); err != nil {
            slog.Error("failed to encode order to JSON", "error", err)
            http.Error(w, "internal server error", http.StatusInternalServerError)
        }
    }
}



