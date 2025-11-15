package logic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/jung-kurt/gofpdf"
)

var linksListMu sync.RWMutex

type LinksList struct {
	LinksList map[int][]Link `json:"links_list"`
}

func NewLinksList() *LinksList {
	return &LinksList{
		LinksList: make(map[int][]Link),
	}
}

const processingFile = "processing.json"
const storageFile = "storage.json"
const nextIDFile = "nextid.json"

func getNextIDUnsafe() (int, error) {
	var nextID int = 1

	file, err := os.ReadFile(nextIDFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return 0, errors.New("не удалось прочитать ")
	}

	if err == nil && strings.TrimSpace(string(file)) != "" {
		if err = json.Unmarshal(file, &nextID); err != nil {
			return 0, errors.New("не удалось разобрать содержимое ")
		}
	}

	currentID := nextID
	nextID++

	load, err := json.MarshalIndent(nextID, "", "  ")
	if err != nil {
		return 0, errors.New("не удалось подготовить данные для записи ")
	}

	if err := os.WriteFile(nextIDFile, load, 0644); err != nil {
		return 0, errors.New("не удалось записать данные ")
	}

	return currentID, nil
}

/*func GetNextID() (int, error) {
	linksListMu.Lock()
	defer linksListMu.Unlock()

	return getNextIDUnsafe()
}*/

func LoudToFile(file string, existing LinksList) error {
	load, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return errors.New("не удалось подготовить данные для записи")
	}

	if err := os.WriteFile(file, load, 0644); err != nil {
		return errors.New("не удалось записать данные в файл")
	}
	return nil
}

func LoadFromFile(fileLoud string) (LinksList, error) {
	var existing LinksList
	file, err := os.ReadFile(fileLoud)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return LinksList{}, errors.New("не удалось прочитать файл")
	}

	if err == nil && strings.TrimSpace(string(file)) != "" {
		if err = json.Unmarshal(file, &existing); err != nil {
			return LinksList{}, errors.New("не удалось разобрать содержимое файла")
		}
	}

	if existing.LinksList == nil {
		existing.LinksList = make(map[int][]Link)
	}
	return existing, nil
}
func DeleteFromFile(file string, nextID int) error {
	fileData, err := LoadFromFile(file)
	if err != nil {
		return err
	}

	delete(fileData.LinksList, nextID)

	err = LoudToFile(file, fileData)
	if err != nil {
		return err
	}

	return nil
}
func (l *LinksList) AddToProcessing(links []Link) (int, error) {
	linksListMu.Lock()
	defer linksListMu.Unlock()

	if len(links) == 0 {
		return 0, errors.New("получен пустой список ссылок")
	}

	existing, err := LoadFromFile(processingFile)
	if err != nil {
		return 0, err
	}

	nextID, err := getNextIDUnsafe()
	if err != nil {
		return 0, err
	}

	existing.LinksList[nextID] = links

	err = LoudToFile(processingFile, existing)
	if err != nil {
		return 0, err
	}

	return nextID, nil
}

// AddLinksWithStatus сохраняет ссылки со статусами напрямую в storage
func (l *LinksList) AddLinksWithStatus(links []Link) (int, error) {
	linksListMu.Lock()
	defer linksListMu.Unlock()

	if len(links) == 0 {
		return 0, errors.New("получен пустой список ссылок")
	}

	storage, err := LoadFromFile(storageFile)
	if err != nil {
		return 0, err
	}

	nextID, err := getNextIDUnsafe()
	if err != nil {
		return 0, err
	}

	storage.LinksList[nextID] = links

	err = LoudToFile(storageFile, storage)
	if err != nil {
		return 0, err
	}

	return nextID, nil
}

func (l *LinksList) UpdateStatusForLinksAndSave(nextID int) ([]Link, error) {
	linksListMu.Lock()
	existing, err := LoadFromFile(processingFile)
	if err != nil {
		linksListMu.Unlock()
		return nil, err
	}

	links := existing.LinksList[nextID]
	linksListMu.Unlock()

	var wg sync.WaitGroup
	for i := range links {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			links[idx].UpdateStatus()
		}(i)
	}
	wg.Wait()

	linksListMu.Lock()
	defer linksListMu.Unlock()

	storage, err := LoadFromFile(storageFile)
	if err != nil {
		return nil, err
	}

	storage.LinksList[nextID] = links

	err = LoudToFile(storageFile, storage)
	if err != nil {
		return nil, err
	}

	err = DeleteFromFile(processingFile, nextID)
	if err != nil {
		return nil, err
	}

	return links, nil
}

func (l *LinksList) GenerateReportPDF(ids []int) ([]byte, error) {
	linksListMu.RLock()
	defer linksListMu.RUnlock()

	if len(ids) == 0 {
		return nil, errors.New("получен пустой список ID")
	}

	storage, err := LoadFromFile(storageFile)
	if err != nil {
		return nil, errors.New("не удалось загрузить storage")
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "", 12)

	for _, id := range ids {
		if links, exists := storage.LinksList[id]; exists {
			for _, link := range links {

				status := "Not Available"
				if link.Status {
					status = "Available"
				}
				pdf.Cell(0, 10, fmt.Sprintf("%d: %s (%s)", id, link.URL, status))
				pdf.Ln(8)
			}
		}
	}

	var buf bytes.Buffer
	err = pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (l *LinksList) RecoverProcessingTasks() error {
	linksListMu.Lock()
	defer linksListMu.Unlock()

	processing, err := LoadFromFile(processingFile)
	if err != nil {
		return err
	}

	for nextID, links := range processing.LinksList {
		taskID := nextID
		taskLinks := make([]Link, len(links))
		copy(taskLinks, links)

		go func(id int, taskLinks []Link) {
			var wg sync.WaitGroup
			for i := range taskLinks {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					taskLinks[idx].UpdateStatus()
				}(i)
			}
			wg.Wait()

			linksListMu.Lock()
			storage, err := LoadFromFile(storageFile)
			if err == nil {
				storage.LinksList[id] = taskLinks
				err = LoudToFile(storageFile, storage)
				if err != nil {
					fmt.Println(err)
					return
				}
			}
			linksListMu.Unlock()

			linksListMu.Lock()
			err = DeleteFromFile(processingFile, id)
			if err != nil {
				fmt.Println(err)
				return
			}
			linksListMu.Unlock()
		}(taskID, taskLinks)
	}

	return nil
}
