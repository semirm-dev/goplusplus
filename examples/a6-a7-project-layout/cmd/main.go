// 🌩 Go++ 🌩 #6 · Coding guidelines: Project layout
// 🌩 Go++ 🌩 #7 · Coding guidelines: Inside a feature/slice
//
// By feature, not by layer (controllers/, services/, repositories/).
// Each feature starts with one file and grows when it needs to.
package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

	discount "github.com/semirm-dev/goplusplus/examples/a6-a7-project-layout/discount/transport"
	"github.com/semirm-dev/goplusplus/examples/a6-a7-project-layout/orders"
	"github.com/semirm-dev/goplusplus/examples/a6-a7-project-layout/orders/repository"
	"github.com/semirm-dev/goplusplus/examples/a6-a7-project-layout/orders/transport"
)

// The only place the features meet.
func main() {
	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	svc := orders.NewService(repository.NewPostgres(db), discount.Client{})

	mux := http.NewServeMux()
	transport.Routes(mux, svc)
	log.Fatal(http.ListenAndServe(":8080", mux))
}
