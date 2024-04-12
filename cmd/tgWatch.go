package main

import (
	"github.com/alexbilevskiy/tgWatch/pkg/config"
	"github.com/alexbilevskiy/tgWatch/pkg/consts"
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

	var accounts []*mongo.DbAccountData
	if len(args) == 1 {
		accounts = mongo.LoadAccounts("")
	} else if len(args) == 2 {
		log.Printf("Using single account %s", args[1])
		accounts = mongo.LoadAccounts(args[1])
	} else {
		log.Fatalf("Invalid argument")
	}

	for _, mongoAcc := range accounts {
		if mongoAcc.Status != consts.AccStatusActive {
			log.Printf("Wont use account %d, because its not active yet: `%s`", mongoAcc.Id, mongoAcc.Status)
			continue
		}
		log.Printf("Create account %d", mongoAcc.Id)
		libs.AS.Create(mongoAcc)
		libs.AS.Get(mongoAcc.Id).RunAccount()
	}
	log.Printf("starting web server...")

	web.InitWeb()

	select {}
}
