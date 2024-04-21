package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/alexbilevskiy/tgWatch-proto/gen/go/tgrpc"
	"github.com/alexbilevskiy/tgWatch/pkg/libs"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/tdlib"
	"github.com/alexbilevskiy/tgWatch/pkg/libs/web"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/zelenin/go-tdlib/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
)

type TgRpcApi struct {
	*tgrpc.UnimplementedMessagesServer
}

func NewServer() *grpc.Server {
	tgApi := &TgRpcApi{}

	logger := log.Default()

	opts := []logging.Option{
		logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),
		// Add any other option (check functions starting with logging.With).
	}

	g := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logging.UnaryServerInterceptor(InterceptorLogger(logger), opts...),
		),
		grpc.ChainStreamInterceptor(
			logging.StreamServerInterceptor(InterceptorLogger(logger), opts...),
		),
	)
	tgrpc.RegisterMessagesServer(g, tgApi)
	reflection.Register(g)

	return g
}

func (t *TgRpcApi) GetScheduledMessages(ctx context.Context, req *tgrpc.GetScheduledMessagesRequest) (*tgrpc.GetScheduledMessagesResponse, error) {
	account := libs.AS.Get(req.Account)
	if account == nil {
		return nil, errors.New("invalid account")
	}
	if req.Peer == 0 {
		return nil, errors.New("invalid peer")
	}
	mess, err := libs.AS.Get(req.Account).TdApi.GetScheduledMessages(req.Peer)
	if err != nil {

		return nil, errors.New(fmt.Sprintf("failed to request messages: %s", err.Error()))
	}

	responseMessages := make([]*tgrpc.Message, 0)
	for _, m := range mess.Messages {
		var sendDate int32
		switch m.SchedulingState.MessageSchedulingStateType() {
		case client.TypeMessageSchedulingStateSendAtDate:
			sendDate = m.SchedulingState.(*client.MessageSchedulingStateSendAtDate).SendDate
		case client.TypeMessageSchedulingStateSendWhenOnline:
			sendDate = 0
		default:
			log.Printf("invalid SchedulingState type: %s", m.SchedulingState.MessageSchedulingStateType())
		}
		responseMessage := &tgrpc.Message{
			Id:          m.Id,
			ChatId:      m.ChatId,
			Date:        m.Date,
			TextPreview: web.RenderText(tdlib.GetContentWithText(m.Content, m.ChatId).FormattedText),
			SchedulingState: &tgrpc.SchedulingState{
				SchedulingStateType: m.SchedulingState.MessageSchedulingStateType(),
				SendDate:            sendDate,
			},
		}
		responseMessages = append(responseMessages, responseMessage)
	}

	return &tgrpc.GetScheduledMessagesResponse{Messages: responseMessages}, nil
}

func (t *TgRpcApi) ScheduleForwardedMessage(ctx context.Context, req *tgrpc.ScheduleForwardedMessageRequest) (*tgrpc.ScheduleForwardedMessageResponse, error) {
	account := libs.AS.Get(req.Account)
	if account == nil {
		return nil, errors.New("invalid account")
	}
	mess, err := libs.AS.Get(req.Account).TdApi.ScheduleForwardedMessage(req.TargetChatId, req.FromChatId, req.MessageIds, req.SendAtDate)
	if err != nil {

		return nil, errors.New(fmt.Sprintf("failed to schedule messages: %s", err.Error()))
	}

	responseMessages := make([]*tgrpc.Message, 0)
	for _, m := range mess.Messages {
		var sendDate int32
		if m.SchedulingState == nil {
			log.Printf("no scheduling state??? message already sent???")
			return nil, errors.New("message probably already sent")
		}
		switch m.SchedulingState.MessageSchedulingStateType() {
		case client.TypeMessageSchedulingStateSendAtDate:
			sendDate = m.SchedulingState.(*client.MessageSchedulingStateSendAtDate).SendDate
		case client.TypeMessageSchedulingStateSendWhenOnline:
			sendDate = 0
		default:
			log.Printf("invalid SchedulingState type: %s", m.SchedulingState.MessageSchedulingStateType())
		}
		responseMessage := &tgrpc.Message{
			Id:          m.Id,
			ChatId:      m.ChatId,
			Date:        m.Date,
			TextPreview: web.RenderText(tdlib.GetContentWithText(m.Content, m.ChatId).FormattedText),
			SchedulingState: &tgrpc.SchedulingState{
				SchedulingStateType: m.SchedulingState.MessageSchedulingStateType(),
				SendDate:            sendDate,
			},
		}
		responseMessages = append(responseMessages, responseMessage)
	}

	return &tgrpc.ScheduleForwardedMessageResponse{Messages: responseMessages}, nil
}

func InterceptorLogger(l *log.Logger) logging.Logger {
	return logging.LoggerFunc(func(_ context.Context, lvl logging.Level, msg string, fields ...any) {
		switch lvl {
		case logging.LevelDebug:
			msg = fmt.Sprintf("DEBUG :%v", msg)
		case logging.LevelInfo:
			msg = fmt.Sprintf("INFO :%v", msg)
		case logging.LevelWarn:
			msg = fmt.Sprintf("WARN :%v", msg)
		case logging.LevelError:
			msg = fmt.Sprintf("ERROR :%v", msg)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
		l.Println(append([]any{"RPC", msg}, fields...))
	})
}
