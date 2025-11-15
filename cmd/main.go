package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/S1FFFkA/15.11.2025/internal/http"
	"github.com/S1FFFkA/15.11.2025/logic"
)

func main() {
	linksList := logic.NewLinksList()
	if err := linksList.RecoverProcessingTasks(); err != nil {
		log.Printf("Предупреждение: не удалось восстановить задачи: %v", err)
	} else {
		log.Println("Восстановление задач при старте завершено")
	}

	server := http.NewServer("8080", linksList)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sig := <-sigChan
	log.Printf("Получен сигнал: %v. Начинаем graceful shutdown...", sig)

	if err := server.Shutdown(30 * time.Second); err != nil {
		log.Printf("Ошибка при остановке сервера: %v", err)
	} else {
		log.Println("Сервер успешно остановлен")
	}
}
