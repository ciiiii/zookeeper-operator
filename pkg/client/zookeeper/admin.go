package zookeeper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type leaderResult struct {
	LeaderId int `json:"leader_id"`
}

func GetLeader(host, port string) (int, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:%s/commands/leader", host, port))
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}
	var result leaderResult
	if err := json.Unmarshal(body, &result); err != nil {
		return -1, err
	}
	return result.LeaderId, nil
}
