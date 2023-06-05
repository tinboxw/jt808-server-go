package wrapper

import "github.com/fakeyanss/jt808-server-go/internal/server"

type Jt808Server struct {
	server.Server
}

func New() *Jt808Server {
	srv := &Jt808Server{Server: server.NewTCPServer()}
	return srv
}
