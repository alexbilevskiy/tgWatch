package main

import (
	"context"
	"log"
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
	cfg, err := config.InitConfiguration()
	if err != nil {
		log.Printf("config read error: %v", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()
	go func() {
		<-ctx.Done()
		<-time.After(30 * time.Second)
		log.Printf("service has not been stopped within the specified timeout; killed by force")
		os.Exit(1)
	}()

	err = tdlib.LoadOptionsList()
	if err != nil {
		log.Printf("tdlib load options error: %v", err)
		os.Exit(1)
	}

	mongoClient := db.NewClient(ctx, cfg)

	args := os.Args
	var phone string

	if len(args) == 1 {
	} else if len(args) == 2 {
		log.Printf("Using single account %s", args[1])
		phone = args[1]
	} else {
		log.Printf("invalid arguments")
		os.Exit(1)
	}
	astorage := db.NewAccountsStorage(cfg, mongoClient)
	astore := account.NewAccountsStore(cfg, mongoClient, astorage)
	creator := tdlib.NewAccountCreator(cfg, astorage)

	if phone == "" {
		err = astore.Run(ctx)
	} else {
		err = astore.RunOne(ctx, phone)
	}
	if err != nil {
		log.Printf("run tdlib error: %v", err)
		os.Exit(1)
	}

	log.Printf("starting web server...")

	err = web.Run(cfg, astore, creator)
	if err != nil {
		log.Printf("web server run error: %v", err)
		os.Exit(1)
	}
	os.Exit(0)
}
