package mywebsocket

type (
	Client interface {
		ID() string
		WriteMessage(data string) error
		WriteJson(data any) error
		ReadChan() <-chan any
		IsConnect() bool
		Connect() error
		Close()
	}
)
