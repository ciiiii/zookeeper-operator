package zookeeper

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/go-zookeeper/zk"
	"k8s.io/klog"
)

type zkClient struct {
	*zk.Conn
}

func NewZKClient(host, port string) (*zkClient, error) {
	url := fmt.Sprintf("%s:%s", host, port)
	c, _, err := zk.Connect([]string{url}, time.Second)
	if err != nil {
		return nil, fmt.Errorf("client(%s), %v", url, err)
	}
	return &zkClient{Conn: c}, nil
}

func (z *zkClient) RegisterServer(id int, url string) error {
	_, err := z.IncrementalReconfig([]string{fmt.Sprintf("server.%d=%s", id, url)}, nil, -1)
	if err != nil {
		return err
	}
	klog.Infof("register server.%d=%s", id, url)
	return nil
}

func (z *zkClient) RemoveServer(id int) error {
	_, err := z.IncrementalReconfig(nil, []string{fmt.Sprint(id)}, -1)
	if err != nil {
		return err
	}
	klog.Infof("remove server.%d", id)
	return nil
}

func (z *zkClient) ListServer() ([]string, error) {
	serversStr, err := z.getConfig()
	if err != nil {
		return nil, err
	}
	return parseServerList(serversStr), nil
}

func (z *zkClient) ExistServer(id int) (bool, error) {
	serversStr, err := z.getConfig()
	if err != nil {
		return false, err
	}
	server := filterServer(serversStr, id)
	return server != "", nil
}

func (z *zkClient) getConfig() (string, error) {
	// ref: [API]#Get Configuration API
	b, _, err := z.Get("/zookeeper/config")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (z *zkClient) PrintDataTree(path string) error {
	l, _, err := z.Children(path)
	if err != nil {
		return err
	}
	if len(l) == 0 {
		v, _, err := z.Get(path)
		if err != nil {
			return err
		}
		fmt.Printf("%q: %d\n", path, len(v))
	} else {
		for _, p := range l {
			if err := z.PrintDataTree(filepath.Join(path, p)); err != nil {
				return err
			}
		}
	}
	return nil
}
