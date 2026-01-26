package rpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alexbilevskiy/tgwatch/internal/account"
	pbapi "github.com/alexbilevskiy/tgwatch/internal/generated/pb/api"
	"github.com/alexbilevskiy/tgwatch/internal/tdlib"
	"github.com/alexbilevskiy/tgwatch/internal/web/utils"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TgRpcApi struct {
	log *slog.Logger
	astore *account.AccountsStore
	pbapi.UnimplementedTgwatchServiceServer
}

func NewHandler(log *slog.Logger, astore *account.AccountsStore) *TgRpcApi {
	tgApi := &TgRpcApi{log:log, astore: astore}

	return tgApi
}

var accId int64 = 118137353

func (t *TgRpcApi) GetMe(ctx context.Context, req *pbapi.GetMeRequest) (*pbapi.GetMeResponse, error) {
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

func (t *TgRpcApi) SearchPublicPosts(ctx context.Context, req *pbapi.SearchPublicPostsRequest) (*pbapi.SearchPublicPostsResponse, error) {
	acc := t.astore.Get(accId)
	if acc == nil {
		return nil, errors.New("account not found")
	}

	foundPosts, err := acc.TdApi.SearchPublicPosts(ctx, req.Query, req.Offset, 100)
	if err != nil {
		return nil, fmt.Errorf("search public posts: %w", err)
	}
	res := &pbapi.SearchPublicPostsResponse{
		FoundMessages: make([]*pbapi.Message, 0, len(foundPosts.Messages)),
	}
	for _, p := range foundPosts.Messages {
		msg, errMsg := acc.TdApi.GetMessage(ctx, p.ChatId, p.Id)
		if errMsg != nil {
			t.log.Warn("unable to get message", "err", errMsg, "chat_id", p.ChatId, "id", p.Id)
		} else {
			p = msg
		}
		link := acc.TdApi.GetLink(ctx, p.ChatId, p.Id)
		formattedText := utils.GetContentWithText(p.Content, p.ChatId)
		//renderedText := utils.RenderText(formattedText.FormattedText)
		res.FoundMessages = append(res.FoundMessages, &pbapi.Message{
			Id:         p.Id,
			ChatId:     p.ChatId,
			Link:       link,
			SenderName: acc.TdApi.GetSenderName(ctx, p.SenderId),
			Date:       timestamppb.New(time.Unix(int64(p.Date), 0)),
			Text:       formattedText.Text,
		})
	}

	return res, nil
}
