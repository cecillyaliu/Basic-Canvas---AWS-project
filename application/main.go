package main

import (
	"demo/database"
	"demo/metrics"
	"demo/svc"
	"flag"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
)

var (
	dbEndpoint = flag.String("db_endpoint", "", "Endpoint of db instance")
	dbUsername = flag.String("db_username", "", "Username to access db")
	dbPassword = flag.String("db_password", "", "Password to access db")
)

func Init() {
	logFile, err := os.OpenFile("/home/admin/webservice/logs/csye6225.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		os.Exit(1)
	}
	log.Logger = zerolog.New(logFile)
	log.Print("Logger initialized")
}

func main() {
	defer metrics.Close()
	flag.Parse()
	Init()
	fmt.Println("Hello go!")

	mux := http.NewServeMux()

	database.New(*dbUsername, *dbPassword, *dbEndpoint)
	svc.Register(mux)

	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Println("Server error:", err)
	}
}
