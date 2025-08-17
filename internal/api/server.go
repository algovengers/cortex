package api

import (
	"cortex/internal"
	"cortex/internal/db"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Server struct {
	router *mux.Router
	queue  *internal.JobQueue
	db     *db.Db
	logger *logrus.Logger
}

func NewServer(db *db.Db, qu *internal.JobQueue) *Server {
	s := &Server{
		router: mux.NewRouter(),
		queue:  qu,
		logger: logrus.New(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.HandleFunc("/schedule", s.scheduleJob).Methods("POST")
	s.router.HandleFunc("/status/{id}", s.getJobStatus).Methods("GET")
}

func (s *Server) Start(port string) error {
	srv := &http.Server{
		Handler:      s.router,
		Addr:         ":" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	s.logger.Infof("Server starting on port %s", port)
	return srv.ListenAndServe()
}
