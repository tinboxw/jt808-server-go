package server

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/fakeyanss/jt808-server-go/internal/protocol"
	"github.com/fakeyanss/jt808-server-go/internal/protocol/model"
	"github.com/fakeyanss/jt808-server-go/internal/storage"
	"github.com/fakeyanss/jt808-server-go/pkg/routines"
)

type TCPServer struct {
	listener net.Listener

	// sessions map[string]*model.Session
	mutex *sync.Mutex
}

func NewTCPServer() *TCPServer {
	return &TCPServer{
		mutex: &sync.Mutex{},
		// sessions: make(map[string]*model.Session),
	}
}

func (serv *TCPServer) Listen(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err == nil {
		serv.listener = l
		log.Debug().Msgf("Listening on %v", addr)
	}

	return err
}

func (serv *TCPServer) Start() {
	for {
		conn, err := serv.listener.Accept()
		if err != nil {
			log.Error().Err(err).Msg("Fail to do listener accept")
			switch {
			case errors.Is(err, io.EOF),
				errors.Is(err, io.ErrClosedPipe),
				errors.Is(err, net.ErrClosed):
				return // close connection when EOF or closed
			default:
				continue
			}
		} else {
			session := serv.accept(conn)
			routines.GoSafe(func() { serv.serve(session) })
		}
	}
}

func (serv *TCPServer) Stop() {
	serv.listener.Close()
}

// 将conn封装为逻辑session
func (serv *TCPServer) accept(conn net.Conn) *model.Session {
	remoteAddr := conn.RemoteAddr().String()
	serv.mutex.Lock()
	session := &model.Session{
		Conn: conn,
		ID:   remoteAddr, // using remote addr default
	}
	// serv.sessions[remoteAddr] = session
	storage.StoreSession(session)
	serv.mutex.Unlock()

	return session
}

func (serv *TCPServer) remove(session *model.Session) {
	serv.mutex.Lock()
	defer serv.mutex.Unlock()

	session.Conn.Close()
	storage.ClearSession(session.ID)
	// delete(serv.sessions, session.ID)

	log.Debug().Str("sessionId", session.ID).Msg("Closing connection from remote.")
}

// 处理每个session的消息
func (serv *TCPServer) serve(session *model.Session) {
	defer serv.remove(session)

	pg := protocol.NewPipeline(session.Conn)
	for {
		// 记录value ctx
		ctx := context.WithValue(context.Background(), model.SessionCtxKey{}, session)

		err := pg.ProcessConnRead(ctx)

		if err == nil {
			continue
		}

		log.Error().Err(err).Str("sessionId", session.ID).Msg("Failed to serve session")

		switch {
		case errors.Is(err, io.EOF),
			errors.Is(err, io.ErrClosedPipe),
			errors.Is(err, net.ErrClosed),
			errors.Is(err, storage.ErrDeviceNotFound):
			return // close connection when EOF or closed
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

// 发送消息到终端设备, 外部调用
func (serv *TCPServer) Send(id string, smsg any) {
	msg := smsg.(model.JT808Msg)
	// session := serv.sessions[id]
	session, err := storage.GetSession(id)
	if err != nil && errors.Is(err, storage.ErrSessionClosed) {
		log.Warn().Str("id", id).Msg("Fail to get session from cache, maybe conn was closed.")
		return
	}

	pg := protocol.NewPipeline(session.Conn)

	// 记录value ctx
	ctx := context.WithValue(context.Background(), model.ProcessDataCtxKey{}, &model.ProcessData{Outgoing: msg})

	err = pg.ProcessConnWrite(ctx)

	if err == nil {
		return
	}

	if errors.Is(err, io.ErrClosedPipe) || errors.Is(err, net.ErrClosed) {
		serv.remove(session)
	}

	log.Error().Err(err).Str("device", id).Msg("Failed to send jtmsg to device")
}
