package rpc

import (
	"context"
	"errors"

	"github.com/alexbilevskiy/tgwatch/internal/account"
	pbapi "github.com/alexbilevskiy/tgwatch/internal/generated/pb/api"
	"github.com/alexbilevskiy/tgwatch/internal/tdlib"
)

type TgRpcApi struct {
	astore *account.AccountsStore
	pbapi.UnimplementedTgwatchServiceServer
}

func NewHandler(astore *account.AccountsStore) *TgRpcApi {
	tgApi := &TgRpcApi{astore: astore}

	return tgApi
}

func (t *TgRpcApi) GetMe(context.Context, *pbapi.GetMeRequest) (*pbapi.GetMeResponse, error) {
	var accId int64 = 118137353
	acc := t.astore.Get(accId)
	if acc == nil {
		return nil, errors.New("account not found")
	}

	return &pbapi.GetMeResponse{
		Id:       acc.Me.Id,
		Username: tdlib.GetUsername(acc.Me.Usernames),
		Name:     tdlib.GetUserFullname(acc.Me),
	}, nil
}
