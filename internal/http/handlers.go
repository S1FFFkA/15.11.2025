package http

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"time"

	"github.com/S1FFFkA/15.11.2025/logic"
)

type Handler struct {
	linksList *logic.LinksList
	server    *Server
}

func NewHandler(linksList *logic.LinksList) *Handler {
	return &Handler{
		linksList: linksList,
	}
}

func (h *Handler) SetServer(server *Server) {
	h.server = server
}

func (h *Handler) AddLinksAndCheckStatus(w http.ResponseWriter, r *http.Request) {
	if h.server != nil {
		select {
		case <-h.server.Context().Done():
			http.Error(w, "Сервер останавливается", http.StatusServiceUnavailable)
			return
		default:
		}
	}

	activeRequests.Add(1)
	defer activeRequests.Done()

	var input AddLinksInputDTO
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errdto := ErrorDTO{
			Message: err.Error(),
			Time:    time.Now(),
		}
		http.Error(w, errdto.Error(), http.StatusBadRequest)
		return
	}

	if len(input.Links) == 0 {
		http.Error(w, "получен пустой список", http.StatusBadRequest)
		return
	}
	var links []logic.Link
	for i := 0; i < len(input.Links); i++ {
		l := logic.NewLink(input.Links[i])
		links = append(links, *l)
	}

	NextID, err := h.linksList.AddToProcessing(links)
	if err != nil {
		errdto := ErrorDTO{
			Message: err.Error(),
			Time:    time.Now(),
		}
		http.Error(w, errdto.Error(), http.StatusBadRequest)
		return
	}
	links, err = h.linksList.UpdateStatusForLinksAndSave(NextID)
	if err != nil {
		errdto := ErrorDTO{
			Message: err.Error(),
			Time:    time.Now(),
		}
		http.Error(w, errdto.Error(), http.StatusBadRequest)
		return
	}
	var linksDTO []link
	for i := 0; i < len(links); i++ {
		var status string
		if links[i].Status == false {
			status = "Not Available"
		}
		if links[i].Status == true {
			status = "Available"
		}
		l := link{
			URL:    links[i].URL,
			Status: status,
		}
		linksDTO = append(linksDTO, l)

	}

	output := AddLinksOutDTO{
		ID:       NextID,
		LinkList: linksDTO,
	}

	b, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		errdto := ErrorDTO{
			Message: err.Error(),
			Time:    time.Now(),
		}
		http.Error(w, errdto.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		errdto := ErrorDTO{
			Message: err.Error(),
			Time:    time.Now(),
		}
		http.Error(w, errdto.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) GenerateReport(w http.ResponseWriter, r *http.Request) {
	if h.server != nil {
		select {
		case <-h.server.Context().Done():
			http.Error(w, "Сервер останавливается", http.StatusServiceUnavailable)
			return
		default:
		}
	}

	activeRequests.Add(1)
	defer activeRequests.Done()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input GenerateReportInputDTO
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errdto := ErrorDTO{
			Message: err.Error(),
			Time:    time.Now(),
		}
		http.Error(w, errdto.Error(), http.StatusBadRequest)
		return
	}

	if len(input.LinksNum) == 0 {
		http.Error(w, "отправлен пустой запрос", http.StatusBadRequest)
		return
	}

	pdfData, err := h.linksList.GenerateReportPDF(input.LinksNum)
	if err != nil {
		errdto := ErrorDTO{
			Message: err.Error(),
			Time:    time.Now(),
		}
		http.Error(w, errdto.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=report.pdf")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfData)))
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(pdfData)
	if err != nil {
		errdto := ErrorDTO{
			Message: err.Error(),
			Time:    time.Now(),
		}
		http.Error(w, errdto.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) Shutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	activeRequests.Add(1)
	defer activeRequests.Done()

	if h.server == nil {
		http.Error(w, "Сервер не инициализирован", http.StatusInternalServerError)
		return
	}

	go func() {
		timeout := 30 * time.Second
		if err := h.server.Shutdown(timeout); err != nil {
			log.Printf("Ошибка при остановке сервера: %v", err)
		}
	}()

	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("Graceful shutdown инициирован"))
	if err != nil {
		errdto := ErrorDTO{
			Message: err.Error(),
			Time:    time.Now(),
		}
		http.Error(w, errdto.Error(), http.StatusInternalServerError)
	}
}
