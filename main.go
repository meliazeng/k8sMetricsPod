package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {

	// set up Metrics server
	if err := prometheus.Register(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "network_latency_to_downstream_srvice",
			Help: "The network transmission delay to downstream service",
		},
		func() float64 {

			curl := exec.Command("curl", "https://www.google.ca", "-s", "-o", "/dev/null", "-w", "%{time_starttransfer}")
			out, err := curl.Output()

			if err != nil {
				log.Println(err)
				return 0
			}
			value, err := strconv.ParseFloat(string(out), 64)
			if err != nil {
				log.Println(err)
				return 1
			}
			return value
		},
	)); err == nil {
		fmt.Println("GaugeFunc 'network_latency_to_downstream_srvice' registered.")
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// setup query server
	http.HandleFunc("/", handler)
	http.ListenAndServe(":443", nil)

	go func() {
		log.Println("Starting metrics server...")
		log.Fatal(srv.ListenAndServe())
	}()

	// Configure Logging
	LOG_FILE_LOCATION := os.Getenv("LOG_FILE_LOCATION")
	if LOG_FILE_LOCATION != "" {
		log.SetOutput(&lumberjack.Logger{
			Filename:   LOG_FILE_LOCATION,
			MaxSize:    500, // megabytes
			MaxBackups: 3,
			MaxAge:     28,   //days
			Compress:   true, // disabled by default
		})
	}
	// Graceful Shutdown
	waitForShutdown(srv)
}

func handler(w http.ResponseWriter, r *http.Request) {
	keys, ok := r.URL.Query()["key"]
	// w.Header().Set("Content-Type", "application/json")

	if !ok || len(keys[0]) < 1 {
		log.Println("Parameter key is missing")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Parameter key is missing")
		return
	}

	key := keys[0]
	curl := exec.Command("curl", key, "-s", "-o", "/dev/null", "-w", "%{time_starttransfer}")
	out, err := curl.Output()

	if err != nil {
		log.Println(err)
		fmt.Fprintf(w, "0")
		return
	}
	/*	value, err := strconv.ParseFloat(string(out), 64)
		if err != nil {
			log.Println(err)
			fmt.Fprintf(w, "0")
			return
		} */
	fmt.Fprintf(w, string(out))
}

func waitForShutdown(srv *http.Server) {
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	<-interruptChan

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	srv.Shutdown(ctx)

	log.Println("Shutting down")
	os.Exit(0)
}
