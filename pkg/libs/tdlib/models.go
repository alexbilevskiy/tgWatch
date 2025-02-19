package tdlib

type TdlibOption struct {
	Name        string `json:"Name"`
	Type        string `json:"Type"`
	Writable    bool   `json:"Writable"`
	Description string `json:"Description"`
	Value       interface{}
}
