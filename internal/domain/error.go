package domain

type Error struct {
	Code    string
	Message string
	Fields  map[string]string
}

func (e Error) Error() string {
	return e.Message
}
