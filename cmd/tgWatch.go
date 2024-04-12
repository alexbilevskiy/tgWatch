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

	if len(args) == 1 {
		mongo.LoadAccounts("")
	} else if len(args) == 2 {
		log.Printf("Using single account %s", args[1])
		mongo.LoadAccounts(args[1])
	} else {
		log.Fatalf("Invalid argument")
	}

	libs.AS.Range(func(accId any, accInt any) bool {
		acc := accInt.(*libs.Account)
		if acc.Status != tdlib.AccStatusActive {
			log.Printf("Wont use account %d, because its not active yet: `%s`", acc.Id, acc.Status)
			return true
		}
		log.Printf("Init account %d", acc.Id)

		acc.RunAccount()
		return true
	})
	log.Printf("starting web server...")

	web.InitWeb()

	select {}
}
