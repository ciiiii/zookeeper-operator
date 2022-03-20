package helper

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"k8s.io/klog/v2"
)

func parseMyId(host string) int {
	ordinal, err := parseStsPodOrdinal(host)
	if err != nil {
		klog.Fatal(err)
	}
	return ordinal + 1
}

func parseStsPodOrdinal(host string) (int, error) {
	separatorIndex := strings.LastIndex(host, "-")
	if separatorIndex == -1 {
		return -1, fmt.Errorf("failed to parse ordinal from hostname: %s", host)
	}
	ordinal, err := strconv.Atoi(host[separatorIndex+1:])
	if err != nil {
		return -1, fmt.Errorf("failed to parse ordinal from hostname: %s", host)
	}
	return ordinal, nil
}

func envWithDefault(key string, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func parseCurrentServer(id int, l []string) (string, error) {
	for _, line := range l {
		if strings.HasPrefix(line, fmt.Sprintf("server.%d", id)) {
			return line, nil
		}
	}
	return "", fmt.Errorf("server.%d not found", id)
}

func parseServerRole(s string) string {
	switch {
	case strings.Contains(s, string(ROLE_OBSERVER)):
		return ROLE_OBSERVER
	case strings.Contains(s, string(ROLE_PARTICIPANT)):
		return ROLE_PARTICIPANT
	default:
		return ROLE_UNKNOWN
	}
}

func generateServerStr(host, role string) string {
	return fmt.Sprintf("%s.%s:%s:%s:%s;%s", host, domain, followerPort, leaderElectionPort, role, clientPort)
}

func checkEnsembleAvailable(domain string) (bool, error) {
	if domain == "" {
		return false, fmt.Errorf("domain is not set")
	}
	_, err := net.LookupIP(domain)
	if err != nil {
		if strings.Contains(err.Error(), "no such host") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func ContainsString(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, str string) (result []string) {
	for _, item := range slice {
		if item == str {
			continue
		}
		result = append(result, item)
	}
	return result
}
