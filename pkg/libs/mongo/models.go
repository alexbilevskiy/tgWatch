package mongo

type ChatFilter struct {
	Id            int32
	Title         string
	IncludedChats []int64
}

type ChatPosition struct {
	ChatId   int64
	ListId   int32
	Order    int64
	IsPinned bool
}
