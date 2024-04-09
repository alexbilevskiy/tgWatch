package main

import (
	"github.com/alexbilevskiy/tgWatch/pkg/config"
	"github.com/alexbilevskiy/tgWatch/pkg/libs"
	"log"
	"os"
)

func main() {
	config.InitConfiguration()

	libs.InitSharedVars()
	libs.InitGlobalMongo()

	args := os.Args
	if len(args) == 1 {
		libs.LoadAccounts("")
	} else if len(args) == 2 {
		log.Printf("Using single account %s", args[1])
		libs.LoadAccounts(args[1])
	} else {
		log.Fatalf("Invalid argument")
	}

	//go libs.InitVoskModel()

	//@TODO: check if goroutine with specific account is alive?
	for accId, acc := range libs.Accounts {
		if acc.Status != libs.AccStatusActive {
			log.Printf("Wont use account %d, because its not active yet: `%s`", acc.Id, acc.Status)
			continue
		}
		log.Printf("Init account %d", acc.Id)

		libs.RunAccount(accId)
	}

	libs.InitWeb()

	select {}
}
