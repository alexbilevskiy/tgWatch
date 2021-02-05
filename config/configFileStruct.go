package config

type ConfigFileStruct struct {
	ApiId   int32  `json:"ApiId"`
	ApiHash string `json:"ApiHash"`
	IgnoreChatIds map[string]string `json:"IgnoreChatIds"`
	IgnoreAuthorIds map[string]string `json:"IgnoreAuthorIds"`
	WebListen string `json:"WebListen"`
	MongoUri string `json:"MongoUri"`
}
