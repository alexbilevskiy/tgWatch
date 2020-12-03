package config

type ConfigFileStruct struct {
	ApiId   int32  `json:"ApiId"`
	ApiHash string `json:"ApiHash"`
	IgnoreChatIds map[string]bool `json:"IgnoreChatIds"`
	IgnoreAuthorIds map[string]bool `json:"IgnoreAuthorIds"`
}
