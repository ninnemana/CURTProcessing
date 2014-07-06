package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Part struct {
	Images []PartImage `json:"images"`
	ID     int         `json:"partID"`
}

type PartImage struct {
	ID     int    `json:"imageID"`
	Size   string `json:"size"`
	Path   string `json:"path"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
	PartID int    `json:"partID"`
	Sort   string `json:"sort"`
}

func main() {
	parts, err := getPartIDs()
	if err != nil {
		panic(err)
	}

	ch := make(chan error)
	for _, part := range parts {
		go func(id int, c chan error) {
			p := Part{ID: id}
			err = p.get()
			if err != nil {
				c <- err
			}
			fmt.Printf("Part #%d has (%d) images\n", p.ID, len(p.Images))
			c <- nil
		}(part, ch)
	}

	for _, _ = range parts {
		err := <-ch
		if err != nil {
			fmt.Println(err)
		}
	}
}

func getPartIDs() ([]int, error) {
	parts := make([]int, 0)
	resp, err := http.Get("http://api.curtmfg.com/v2/GetAllPartID?dataType=json")
	if err != nil {
		return parts, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return parts, err
	}

	if err = json.Unmarshal(data, &parts); err != nil {
		return parts, err
	}

	return parts, nil
}

func (p *Part) get() error {

	resp, err := http.Get(fmt.Sprintf("http://api.curtmfg.com/v2/GetPart?partID=%d&dataType=json", p.ID))
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(data, &p); err != nil {
		return err
	}

	return nil
}
