package types

import (
	"encoding/json"
	"net/http"
)

type MesCode int

const (
	ErrCode MesCode = -1
	SucCode MesCode = 1
)

type Mes string

const (
	ErrMes Mes = "Operate Failure"
	SucMes Mes = "Operate Successfully"
)

func NewResponseObj[V any](mesCode MesCode, mes Mes, content V) ResponseObj[V] {
	return ResponseObj[V]{
		Code:    mesCode,
		Mes:     mes,
		Content: content,
	}
}

type ResponseObj[V any] struct {
	Code    MesCode `json:"code"`
	Mes     Mes     `json:"mes"`
	Content V       `json: "content"`
}

type EmptyOjb struct{}

func WriteJsonResponse(w http.ResponseWriter, err error, code MesCode, mes Mes, obj any) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		ro := NewResponseObj(ErrCode, ErrMes, EmptyOjb{})
		json.NewEncoder(w).Encode(ro)
	} else {
		w.WriteHeader(http.StatusOK)
		ro := NewResponseObj(SucCode, SucMes, obj)
		json.NewEncoder(w).Encode(ro)
	}
}
