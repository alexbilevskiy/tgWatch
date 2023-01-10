module github.com/alexbilevskiy/tgWatch

go 1.16

require (
	github.com/zelenin/go-tdlib v0.0.0-00010101000000-000000000000
	go.mongodb.org/mongo-driver v1.9.1
)

replace github.com/zelenin/go-tdlib => github.com/alexbilevskiy/go-tdlib v0.4.2-0.20230110083222-aad851f24a21
