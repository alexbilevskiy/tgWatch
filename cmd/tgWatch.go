package main

import (
	"log"
	"os"
	"tgWatch/config"
	"tgWatch/libs"
)

func main() {
	config.InitConfiguration()

	libs.InitSharedVars()
	libs.InitGlobalMongo()

	args := os.Args
	if len(args) == 1 {
		libs.LoadAccounts(libs.GetAccountsFilter(nil))
	} else if len(args) == 2 {
		log.Printf("Using single account %s", args[1])
		libs.LoadAccounts(libs.GetAccountsFilter(&args[1]))
	} else {
		log.Fatalf("Invalid argument")
	}

	//go libs.InitVoskModel()

	//@TODO: check if goroutine with specific account is alive
	//@TODO: reload list when new account added
	for accId, acc := range libs.Accounts {
		if acc.Status != libs.AccStatusActive {
			log.Printf("Wont use account %d, because its not active yet: `%s`", acc.Id, acc.Status)
			continue
		}
		log.Printf("Init account %d", acc.Id)

		libs.InitSharedSubVars(accId)
		libs.InitMongo(accId)
		libs.InitTdlib(accId)
		go libs.ListenUpdates(accId)
	}

	libs.InitWeb()

	select {}
}
