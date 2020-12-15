package main

import (
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func main() {
	// Init logger
	InitBasicLogger()

	// address to listen to
	addr := "127.0.0.1:4444"

	// Prepare API
	// Add CORS
	allowedHeaders := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type"})
	allowedOrigins := handlers.AllowedOrigins([]string{"*"})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "HEAD", "POST"})

	router := mux.NewRouter()

	faucetEndpoint := FaucetEndpoint{}
	faucetEndpoint.StartHandler(router)

	// Start Server
	srv := &http.Server{
		Handler: handlers.CORS(allowedHeaders, allowedOrigins, allowedMethods)(router),
		Addr:    addr,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.WithFields(log.Fields{
		"addr": addr,
	}).Info("Starting Faucet")
	if err := srv.ListenAndServe(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("ListenAndServer() error")
	}
}

// InitBasicLogger initialize a basic logger that will not log to file only stdout
func InitBasicLogger() {
	log.SetLevel(log.DebugLevel)
	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)
	log.SetLevel(log.DebugLevel)
}
