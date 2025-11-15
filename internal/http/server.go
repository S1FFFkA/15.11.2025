package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/S1FFFkA/15.11.2025/logic"
)

var activeRequests sync.WaitGroup

var ErrServerClosed = http.ErrServerClosed

type Server struct {
	handler *Handler
	port    string
	ctx     context.Context
	cancel  context.CancelFunc
	srv     *http.Server
}

func NewServer(port string, linksList *logic.LinksList) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		handler: NewHandler(linksList),
		port:    port,
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (s *Server) Start() error {
	s.srv = &http.Server{
		Addr:    fmt.Sprintf(":%s", s.port),
		Handler: nil,
	}

	s.handler.SetServer(s)

	http.HandleFunc("/add", s.handler.AddLinksAndCheckStatus)
	http.HandleFunc("/report", s.handler.GenerateReport)
	http.HandleFunc("/shutdown", s.handler.Shutdown)

	log.Printf("Сервер запущен на порту %s", s.port)
	log.Println("Эндпоинты:")
	log.Println("/add - добавить ссылки и получить статусы")
	log.Println("/report - сгенерировать PDF отчет")
	log.Println("/shutdown - graceful shutdown сервера")

	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(timeout time.Duration) error {
	s.cancel()

	done := make(chan struct{})
	go func() {
		activeRequests.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Все активные запросы завершены")
	case <-time.After(timeout):
		log.Printf("Таймаут ожидания завершения запросов (%v)", timeout)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return s.srv.Shutdown(ctx)
}

func (s *Server) Context() context.Context {
	return s.ctx
}
