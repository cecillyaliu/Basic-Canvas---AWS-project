package svc

import "net/http"

func Assignments(w http.ResponseWriter, r *http.Request) {
	(&Handler{}).Assignments(w, r)
}

func CheckHealth(w http.ResponseWriter, r *http.Request) {
	(&Handler{}).CheckHealth(w, r)
}

func Register(mux *http.ServeMux) {
	mux.Handle("/v2/assignments/", http.HandlerFunc(Assignments))
	mux.Handle("/v2/assignments", http.HandlerFunc(Assignments))
	mux.Handle("/healthz", http.HandlerFunc(CheckHealth))
}
