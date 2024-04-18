package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/alexbilevskiy/tgWatch-proto/gen/go/tgrpc"
	"github.com/alexbilevskiy/tgWatch/pkg/libs"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
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

	mess, err := libs.AS.Get(req.Account).TdApi.GetScheduledMessages(req.Peer)
	if err != nil {

		return nil, errors.New(fmt.Sprintf("failed to request messages: %s", err.Error()))
	}

	responseMessages := make([]*tgrpc.Message, 0)
	for _, m := range mess.Messages {
		responseMessage := &tgrpc.Message{Id: m.Id}
		responseMessages = append(responseMessages, responseMessage)
	}

	return &tgrpc.GetScheduledMessagesResponse{Messages: responseMessages}, nil
}
func (t *TgRpcApi) GetScheduledHistory(ctx context.Context, req *tgrpc.GetScheduledHistoryRequest) (*tgrpc.GetScheduledHistoryResponse, error) {

	return &tgrpc.GetScheduledHistoryResponse{Messages: make([]*tgrpc.Message, 0)}, nil
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
