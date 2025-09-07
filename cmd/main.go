package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	api "github.com/glkeru/EM_Subscriptions/internal/api"
	config "github.com/glkeru/EM_Subscriptions/internal/config"
	db "github.com/glkeru/EM_Subscriptions/internal/db"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

func main() {

	// config
	conf, err := config.ConfigLoad()
	if err != nil {
		log.Fatal("config fatal error", err)
	}
	// logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("logger fatal error", err)
	}
	defer logger.Sync()

	// database
	repo, err := db.NewRepository(conf)
	if err != nil {
		log.Fatal("database connection fatal error", err)
	}

	// server
	r, err := api.NewServer(repo, logger, conf)

	crs := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:8088", "http://127.0.0.1:8088"}})
	handler := crs.Handler(r)

	srv := &http.Server{
		Handler:      handler,
		Addr:         ":" + conf.Port,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}
	logger.Info("server starting")
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("start server error", err)
		}
	}()

	// shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	<-interrupt
	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = srv.Shutdown(timeout)
	if err != nil {
		logger.Error("shutdown error", zap.Error(err))
	} else {
		logger.Info("server stoped")
	}
}
