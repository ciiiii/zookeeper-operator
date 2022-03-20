package zookeeper

import (
	"fmt"
	"time"

	"github.com/go-zookeeper/zk"
)

// 3 FourLetterWord(ruok, srvr, cons)
func Ruok(host, port string) bool {
	url := fmt.Sprintf("%s:%s", host, port)
	return zk.FLWRuok([]string{url}, time.Second)[0]
}

func Srvr(host, port string) (*zk.ServerStats, error) {
	url := fmt.Sprintf("%s:%s", host, port)
	serverStats, ok := zk.FLWSrvr([]string{url}, time.Second)
	if !ok {
		return nil, fmt.Errorf("send srvr request failed")
	}
	return serverStats[0], nil
}

func Cons(host, port string) ([]*zk.ServerClient, error) {
	url := fmt.Sprintf("%s:%s", host, port)
	cons, ok := zk.FLWCons([]string{url}, time.Second)
	if !ok {
		return nil, fmt.Errorf("send cons request failed")
	}
	if cons[0].Error != nil {
		return nil, cons[0].Error
	}
	return cons[0].Clients, nil
}
