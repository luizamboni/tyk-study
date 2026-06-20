package response

import (
	"encoding/json"
	"net/http"
)

func Error(writer http.ResponseWriter, status int, code, detail string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]string{
		"error":  code,
		"detail": detail,
	})
}
