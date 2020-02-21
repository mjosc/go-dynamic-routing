package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func NewMockService() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data []byte
		var err error

		path := r.URL.Path

		if r.Header.Get("Content-Type") == "application/json" {
			w.Header().Set("Content-Type", "application/json")

			data, err = json.Marshal(&MockServiceAPIResponse{
				Status: "success",
				Path:   path,
			})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("{\"status\": \"internal error\""))
				return
			}
		} else {
			data = []byte(fmt.Sprintf("<p>success @ %v </p>", path))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(data))
	})
}

type MockServiceAPIResponse struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}
