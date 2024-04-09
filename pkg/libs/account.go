package libs

func RunAccount(accId int64) {
	initSharedSubVars(accId)
	initMongo(accId)
	initTdlib(accId)
	go listenUpdates(accId)
}
