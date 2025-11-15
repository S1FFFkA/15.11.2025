package http

import (
	"encoding/json"
	"time"
)

type AddLinksInputDTO struct {
	Links []string `json:"links"`
}

type link struct {
	URL    string `json:"url"`
	Status string `json:"status"`
}
type AddLinksOutDTO struct {
	ID       int    `json:"links_id"`
	LinkList []link `json:"linkList"`
}

type GenerateReportInputDTO struct {
	LinksNum []int `json:"links_id"`
}

type ErrorDTO struct {
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

func (e ErrorDTO) Error() string {
	b, err := json.MarshalIndent(e, "", "	")
	if err != nil {
		return err.Error()
	}
	return string(b)
}
