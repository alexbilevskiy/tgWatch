package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/alexbilevskiy/tgWatch/internal/account"
	"github.com/alexbilevskiy/tgWatch/internal/config"
	"github.com/alexbilevskiy/tgWatch/internal/db"
	"github.com/alexbilevskiy/tgWatch/internal/tdlib"
	"github.com/alexbilevskiy/tgWatch/internal/web"
)

func main() {
	baseCtx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg, err := config.InitConfiguration()
	if err != nil {
		logger.Error("config read", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(baseCtx, os.Interrupt, os.Kill)
	defer stop()
	go func() {
		<-ctx.Done()
		<-time.After(30 * time.Second)
		logger.Error("service has not been stopped within the specified timeout; killed by force")
		os.Exit(1)
	}()

	err = tdlib.LoadOptionsList()
	if err != nil {
		logger.Error("tdlib load options", "error", err)
		os.Exit(1)
	}

	mongoClient := db.NewClient(ctx, cfg)

	args := os.Args
	var phone string

	if len(args) == 1 {
	} else if len(args) == 2 {
		logger.Info("Using single account", "phone", args[1])
		phone = args[1]
	} else {
		logger.Error("invalid arguments")
		os.Exit(1)
	}
	astorage := db.NewAccountsStorage(cfg, mongoClient)
	astore := account.NewAccountsStore(logger, cfg, mongoClient, astorage)
	creator := tdlib.NewAccountCreator(logger, cfg, astorage)

	err = astore.Run(ctx, phone)
	if err != nil {
		logger.Error("run tdlib", "error", err)
		os.Exit(1)
	}

	logger.Info("starting web server...")

	err = web.Run(logger, cfg, astore, creator)
	if err != nil {
		logger.Error("web server run", "error", err)
		os.Exit(1)
	}
	os.Exit(0)
}
