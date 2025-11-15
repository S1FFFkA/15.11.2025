package logic

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Link struct {
	URL    string `json:"url"`
	Status bool   `json:"status"` // TRUE=available || FALSE=NOTavailable
}

func NewLink(URL string) *Link {
	if URL == "" {
		fmt.Println("ведена пустая ссылка")
		return nil
	}
	return &Link{
		URL:    URL,
		Status: false,
	}
}

func (l *Link) UpdateStatus() {
	l.Status = LinkStatus(l.URL)
}

func LinkStatus(URL string) bool {

	url := strings.TrimSpace(URL)
	if url == "" {
		return false
	}

	urlsToTry := []string{}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		urlsToTry = append(urlsToTry, url)
	} else {
		urlsToTry = append(urlsToTry, "https://"+url, "http://"+url)
	}

	for _, testURL := range urlsToTry {
		if checkURLStatus(testURL) {
			return true
		}
	}

	return false
}

func checkURLStatus(url string) bool {

	client := &http.Client{
		Timeout: 3 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("слишком много редиректов")
			}
			return nil
		},
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}

	if resp == nil {
		return false
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	if statusCode >= http.StatusOK && statusCode < http.StatusBadRequest {
		return true
	}

	if statusCode >= http.StatusMultipleChoices && statusCode < http.StatusBadRequest {
		return true
	}

	return false
}
