package server

type Server interface {
	Listen(addr string) error
	Start()
	Stop()
	Send(string, any)
}
