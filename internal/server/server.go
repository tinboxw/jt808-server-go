package server

type Server interface {
	Listen(addr string) error
	Start()
	Stop()
	Send(sessionId string, smsg any)
	SendV2(phone string, smsg any, rspFn func(any) error) error
}
