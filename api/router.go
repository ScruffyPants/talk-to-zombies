package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/olahol/melody"
	"github.com/sirupsen/logrus"

	"github.com/ScruffyPants/talk-to-zombies/communication"
)

type Router struct {
	httpServer       *http.Server
	websocketHandler *melody.Melody

	communicationService communication.Service
}

type RouterSettings struct {
	Address string

	PingInterval time.Duration
	PongWait     time.Duration
	WriteWait    time.Duration
}

func NewRouter(settings RouterSettings, communicationService communication.Service) *Router {
	mux := &http.ServeMux{}

	r := &Router{
		communicationService: communicationService,
		httpServer: &http.Server{
			Addr:    settings.Address,
			Handler: mux,
		},
	}

	r.websocketHandler = melody.New()
	r.websocketHandler.Config.PingPeriod = settings.PingInterval
	r.websocketHandler.Config.PongWait = settings.PongWait
	r.websocketHandler.Config.WriteWait = settings.WriteWait

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if err := r.websocketHandler.HandleRequest(w, req); err != nil {
			logrus.Errorf("error handling request: %s", err.Error())
		}
	})

	r.websocketHandler.HandleConnect(r.OnConnect)
	r.websocketHandler.HandleMessage(r.HandleMessage)
	r.websocketHandler.HandleDisconnect(r.HandleDisconnect)

	return r
}

func (r *Router) Start() {
	go func() {
		if err := r.httpServer.ListenAndServe(); err != nil {
			panic(err)
		}
	}()
}

func (r *Router) OnConnect(session *melody.Session) {
	connectionIP := getUserIP(session.Request)

	connectionID, err := r.communicationService.NewConnection(&melodySessionConnection{session})
	if err != nil {
		logrus.Infof("error creating new connection: %s", err.Error())

		if err = session.CloseWithMsg([]byte(fmt.Sprintf("error creating new connection: %s", err.Error()))); err != nil {
			logrus.Infof("error closing with message: %s", err.Error())
		}
	}

	session.Set("connection_id", connectionID)
	session.Set("connection_ip", connectionIP)
	session.Set("api", "websocket")
	session.Set("connection_time", time.Now())
}

func (r *Router) HandleMessage(session *melody.Session, bytes []byte) {
	ctx := context.Background()
	for k, v := range session.Keys {
		ctx = context.WithValue(ctx, k, v)
	}

	messageString := string(bytes)
	messageSplit := strings.Split(messageString, " ")

	if len(messageSplit) == 0 {
		if err := session.Write([]byte(fmt.Sprintf("message cannot be empty"))); err != nil {
			logrus.WithContext(ctx).Errorf("error writting message to ws connection: %s", err.Error())
		}
		return
	}

	message := communication.Message{
		Type: messageSplit[0],
	}

	if len(messageSplit) > 1 {
		message.Arguments = messageSplit[1:]
	}

	connectionID, ok := session.Get("connection_id")
	if !ok {
		if err := session.CloseWithMsg([]byte("connection ID not found, disconnecting")); err != nil {
			logrus.WithContext(ctx).Errorf("error closing with message: %s", err.Error())
		}
		return
	}

	if err := r.communicationService.HandleMessage(ctx, connectionID.(string), message); err != nil {
		logrus.WithContext(ctx).Errorf("error handling user message: %s", err.Error())

		if err = session.Write([]byte(err.Error())); err != nil {
			logrus.WithContext(ctx).Errorf("error writting message: %s", err.Error())
		}
	}
}

func (r *Router) HandleDisconnect(session *melody.Session) {
	connectionID, ok := session.Get("connection_id")
	if !ok {
		logrus.Errorf("error connection_id not found in melody session")

		if err := session.CloseWithMsg([]byte("connection ID not found, disconnecting")); err != nil {
			logrus.Errorf("error closing with message: %s", err.Error())
		}

		return
	}

	r.communicationService.HandleDisconnect(nil, connectionID.(string))
}

type melodySessionConnection struct {
	*melody.Session
}

func (c *melodySessionConnection) SendMessage(message []byte) error {
	return c.Write(message)
}

func getUserIP(r *http.Request) string {
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		return strings.Split(forwardedFor, ",")[0]
	}

	// fallback to remote Address
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}

	return ip
}
