package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	apiKey := os.Getenv("WEATHER_API_KEY")
	if apiKey == "" {
		apiKey = "3526bcb2539d4e95b08181934260907"
	}

	app := NewApp(
		NewZipcodeClient(http.DefaultClient, "https://viacep.com.br"),
		NewWeatherClient(http.DefaultClient, "https://api.weatherapi.com/v1", apiKey),
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           app,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
