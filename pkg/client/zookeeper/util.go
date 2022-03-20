package zookeeper

import (
	"fmt"
	"strings"
)

func parseServerList(s string) []string {
	l := []string{}
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "server.") {
			l = append(l, line)
		}
	}
	return l
}

func filterServer(s string, id int) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, fmt.Sprintf("server.%d", id)) {
			return line
		}
	}
	return ""
}
