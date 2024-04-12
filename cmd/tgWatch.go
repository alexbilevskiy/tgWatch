package main

import (
	"github.com/alexbilevskiy/tgWatch/pkg/config"
	"github.com/alexbilevskiy/tgWatch/pkg/libs"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/mongo"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/tdlib"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/web"
	"log"
	"os"
)

func main() {
	config.InitConfiguration()

	tdlib.LoadOptionsList()
	mongo.InitGlobalMongo()

	args := os.Args
	var phone string

	if len(args) == 1 {
	} else if len(args) == 2 {
		log.Printf("Using single account %s", args[1])
		phone = args[1]
	} else {
		log.Fatalf("Invalid argument")
	}

	if phone == "" {
		libs.AS.Run()
	} else {
		libs.AS.RunOne(phone)
	}

	log.Printf("starting web server...")

	web.InitWeb()

	select {}
}
