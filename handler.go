package command

type Handler interface {
	Handle(cmd Command)
}
