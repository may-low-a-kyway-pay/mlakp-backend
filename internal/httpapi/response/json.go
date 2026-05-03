package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type ErrorBody struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type SuccessBody struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
}

type PaginatedBody struct {
	Success    bool       `json:"success"`
	Data       any        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Default().Error("failed to write json response", "error", err)
	}
}

func Success(w http.ResponseWriter, status int, data any) {
	JSON(w, status, SuccessBody{
		Success: true,
		Data:    data,
	})
}

func Paginated(w http.ResponseWriter, status int, data any, pagination Pagination) {
	JSON(w, status, PaginatedBody{
		Success:    true,
		Data:       data,
		Pagination: pagination,
	})
}

func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, ErrorBody{
		Success: false,
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
