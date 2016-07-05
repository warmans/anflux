package server

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func NewCloseChecker(c *websocket.Conn) *CloseChecker {
	checker := &CloseChecker{C: make(chan bool)}
	go checker.StartChecking(c)
	return checker
}

type CloseChecker struct {
	C chan bool
}

func (cc *CloseChecker) StartChecking(c *websocket.Conn) {
	for {
		if _, _, err := c.ReadMessage(); err != nil {
			cc.C <- true
			return
		}
	}
}

func Fail(rw http.ResponseWriter, msg string, status int) {
	if status >= http.StatusInternalServerError {
		log.Printf("[%d] %s", status, msg)
	}
	http.Error(rw, msg, status)
}
