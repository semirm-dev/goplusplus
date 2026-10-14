package transport

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/semirm-dev/goplusplus/examples/a6-a7-project-layout/orders"
)

func Routes(mux *http.ServeMux, svc *orders.Service) {
	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var o orders.Order
		if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := svc.Create(r.Context(), o); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		o, err := svc.Get(r.Context(), r.PathValue("id"))
		if errors.Is(err, orders.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(o)
	})
}
