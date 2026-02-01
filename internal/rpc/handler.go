package rpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"unicode/utf16"

	"github.com/alexbilevskiy/tgwatch/internal/account"
	pbapi "github.com/alexbilevskiy/tgwatch/internal/generated/pb/api"
	"github.com/alexbilevskiy/tgwatch/internal/tdlib"
	"github.com/alexbilevskiy/tgwatch/internal/web/utils"
	"github.com/zelenin/go-tdlib/client"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TgRpcApi struct {
	log    *slog.Logger
	astore *account.AccountsStore
	pbapi.UnimplementedTgwatchServiceServer
}

func NewHandler(log *slog.Logger, astore *account.AccountsStore) *TgRpcApi {
	tgApi := &TgRpcApi{log: log, astore: astore}

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

func (t *TgRpcApi) SearchPublicPostsFiltered(ctx context.Context, req *pbapi.SearchPublicPostsFilteredRequest) (*pbapi.SearchPublicPostsFilteredResponse, error) {
	acc := t.astore.Get(accId)
	if acc == nil {
		return nil, errors.New("account not found")
	}

	res := &pbapi.SearchPublicPostsFilteredResponse{
		FoundMessages: make([]*pbapi.FilteredMessage, 0, req.Limit),
	}

	var foundCnt int32
	offset := ""
	for foundCnt < req.Limit {
		foundPosts, err := acc.TdApi.SearchPublicPosts(ctx, req.Query, offset, 100)
		if err != nil {
			return nil, fmt.Errorf("search public posts filtered: %w", err)
		}
		offset = foundPosts.NextOffset
		parsed := t.ParseMessages(ctx, acc, foundPosts.Messages)
		res.FoundMessages = append(res.FoundMessages, parsed...)
		foundCnt += int32(len(parsed))
		if foundCnt >= req.Limit {
			break
		}
	}
	extraLinks := t.SetVerdictsByLinks(ctx, acc, res.FoundMessages)
	extraMessages := t.GetExtraMessages(ctx, acc, extraLinks)
	parsedExtra := t.ParseMessages(ctx, acc, extraMessages)
	_ = t.SetVerdictsByLinks(ctx, acc, parsedExtra)
	for _, m := range parsedExtra {
		found := false
		for _, existing := range res.FoundMessages {
			if existing.Id == m.Id && existing.ChatId == m.ChatId {
				found = true
			}
		}
		if !found {
			res.FoundMessages = append(res.FoundMessages, m)
		}
	}

	return res, nil
}

func (t *TgRpcApi) ParseMessages(ctx context.Context, acc *account.Account, messages []*client.Message) []*pbapi.FilteredMessage {
	var res []*pbapi.FilteredMessage
	for _, p := range messages {
		msg, errMsg := acc.TdApi.GetMessage(ctx, p.ChatId, p.Id)
		if errMsg != nil {
			t.log.Warn("unable to get message", "err", errMsg, "chat_id", p.ChatId, "id", p.Id)
		} else {
			p = msg
		}
		if p.ForwardInfo != nil {
			continue
		}
		link := acc.TdApi.GetLink(ctx, p.ChatId, p.Id)
		contentLinks := t.getLinks(p.Content)

		formattedText := utils.GetContentWithText(p.Content, p.ChatId)
		//renderedText := utils.RenderText(formattedText.FormattedText)

		res = append(res, &pbapi.FilteredMessage{
			Id:         p.Id,
			ChatId:     p.ChatId,
			Link:       link,
			Source:     pbapi.FilteredMessage_FROM_SEARCH,
			SenderName: acc.TdApi.GetSenderName(ctx, p.SenderId),
			Date:       timestamppb.New(time.Unix(int64(p.Date), 0)),
			Text:       formattedText.Text,
			Links:      contentLinks,
		})
	}

	return res
}

func (t *TgRpcApi) SetVerdictsByLinks(ctx context.Context, acc *account.Account, messages []*pbapi.FilteredMessage) map[string]bool {
	extraLinks := make(map[string]bool)
	for _, m := range messages {
		cntLinkToMessage := 0
		cntLinkToChat := 0
		for _, l := range m.Links {
			linkType, err := acc.TdApi.GetLinkType(ctx, l)
			if err != nil {
				t.log.Warn("unable to get link info", "link", l, "source", m.Link, "err", err)
				continue
			}
			typ := linkType.InternalLinkTypeConstructor()
			switch typ {
			case client.ConstructorInternalLinkTypeMessage:
				extraLinks[l] = true
				cntLinkToMessage++
			case client.ConstructorInternalLinkTypePublicChat:
				cntLinkToChat++
			case client.ConstructorInternalLinkTypeChatInvite:
				cntLinkToChat++
			default:
				t.log.Info("skipping link", "type", typ, "link", l, "source", m.Link, "err", err)
			}
		}
		if cntLinkToMessage == 0 {
			m.Verdict = pbapi.FilteredMessage_OK_ONLY_CHANNEL_LINKS
		} else {
			m.Verdict = pbapi.FilteredMessage_FAIL_HAS_MESSAGE_LINKS
		}
	}

	return extraLinks
}

func (t *TgRpcApi) GetExtraMessages(ctx context.Context, acc *account.Account, extraLinks map[string]bool) []*client.Message {
	extraMessages := make([]*client.Message, 0, len(extraLinks))
	for l, _ := range extraLinks {
		_, m, err := acc.TdApi.GetLinkInfoResolved(ctx, l)
		if err != nil {
			t.log.Warn("unable to get link info", "link", l, "err", err)
			continue
		}
		if me, ok := m.(error); ok {
			t.log.Warn("unable to resolve link", "link", l, "err", me)
			continue
		}
		mm, ok := m.(*client.Message)
		if !ok {
			t.log.Warn("unable to cast resolved message", "link", l, "m, m")
			continue
		}

		extraMessages = append(extraMessages, mm)
	}

	return extraMessages
}

func (t *TgRpcApi) getLinks(content client.MessageContent) []string {
	var caption *client.FormattedText
	cType := content.MessageContentConstructor()
	switch cType {
	case client.ConstructorMessageText:
		msg := content.(*client.MessageText)
		caption = msg.Text
	case client.ConstructorMessagePhoto:
		msg := content.(*client.MessagePhoto)
		caption = msg.Caption
	case client.ConstructorMessageVideo:
		msg := content.(*client.MessageVideo)
		caption = msg.Caption
	case client.ConstructorMessageAnimation:
		msg := content.(*client.MessageAnimation)
		caption = msg.Caption
	default:
		t.log.Warn("unknown content type", "type", cType)
		return nil
	}
	utfText := utf16.Encode([]rune(caption.Text))

	uniq := make(map[string]struct{})
	for _, e := range caption.Entities {
		eType := e.Type.TextEntityTypeConstructor()
		switch eType {
		case client.ConstructorTextEntityTypeMention:
			link := fmt.Sprintf("https://t.me/%s", string(utf16.Decode(utfText[e.Offset:e.Offset+e.Length])))
			uniq[link] = struct{}{}
		case client.ConstructorTextEntityTypeUrl:
			link := string(utf16.Decode(utfText[e.Offset : e.Offset+e.Length]))
			uniq[link] = struct{}{}
		case client.ConstructorTextEntityTypeTextUrl:
			m := e.Type.(*client.TextEntityTypeTextUrl)
			uniq[m.Url] = struct{}{}
		default:
			continue
		}
	}
	res := make([]string, 0, len(uniq))
	for k := range uniq {
		res = append(res, k)
	}

	return res
}
