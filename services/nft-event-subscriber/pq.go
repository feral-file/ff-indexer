package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// Listener is a struct of listener service
type Listener struct {
	uri      string
	db       *sql.DB
	closeCh  chan struct{}
	Notify   chan *pq.Notification
	listener *pq.Listener
}

// Close will stop the listener
func (l *Listener) Close() {
	close(l.closeCh)
}

// report is a callback function of pq listener
func (l *Listener) report(et pq.ListenerEventType, err error) {
	switch et {
	case pq.ListenerEventConnected:
		logrus.Info("listener connection is established")
	default:
		logrus.Warnf("listener connection is disconnected. code: %d", et)
	}
	if err != nil {
		logrus.Error(err.Error())
	}
}

// Start will create a goroutine listening to all notifications from db
func (l *Listener) Start() {
	logrus.Info("connecting to database…")
	l.listener = pq.NewListener(l.uri, 2*time.Second, 16*time.Second, l.report)

	go func() {
		logrus.Info("start listening loop…")
		for {
			select {
			case n := <-l.listener.Notify:
				if n != nil {
					logrus.Debugf("receive a message from sql channel: %+v", n)
					l.Notify <- n
				}
			case <-l.closeCh:
				logrus.Info("receive stop signal, will close channels")
				if l.listener != nil {
					l.listener.Close()
				}
				close(l.Notify)
				return
			}
		}
	}()
}

// Watch will let listener listen to a specific channel so that
// the notifier can receive messages from it.
// Since the Listen function in psql will block until messages
// coming, we have to wrap it into a goroutine.
func (l *Listener) Watch(channel string) error {
	if l.listener != nil {
		g := new(errgroup.Group)
		g.Go(func() error {
			return l.listener.Listen(channel)
		})
		return nil
	}
	return fmt.Errorf("listener is not initialised")
}

// NewListener will return an instance of Listener
func NewListener(uri string) (*Listener, error) {
	closeCh := make(chan struct{})
	db, err := sql.Open("postgres", uri)
	if err != nil {
		return nil, err
	}

	return &Listener{
		db:      db,
		uri:     uri,
		closeCh: closeCh,
		Notify:  make(chan *pq.Notification, 1000),
	}, nil
}
