package main

import (
	"log"
	"os"

	"github.com/alexbilevskiy/tgWatch/internal/account"
	"github.com/alexbilevskiy/tgWatch/internal/config"
	"github.com/alexbilevskiy/tgWatch/internal/db"
	"github.com/alexbilevskiy/tgWatch/internal/tdlib"
	"github.com/alexbilevskiy/tgWatch/internal/web"
)

func main() {
	cfg, err := config.InitConfiguration()
	if err != nil {
		log.Fatal(err)
	}
	err = tdlib.LoadOptionsList()
	if err != nil {
		log.Fatal(err)
	}

	mongoClient := db.NewClient(cfg)

	args := os.Args
	var phone string

	if len(args) == 1 {
	} else if len(args) == 2 {
		log.Printf("Using single account %s", args[1])
		phone = args[1]
	} else {
		log.Fatalf("Invalid argument")
	}
	astorage := db.NewAccountsStorage(cfg, mongoClient)
	astore := account.NewAccountsStore(mongoClient, astorage)

	if phone == "" {
		go astore.Run()
	} else {
		astore.RunOne(phone)
	}

	log.Printf("starting web server...")

	web.InitWeb(cfg)

	select {}
}
