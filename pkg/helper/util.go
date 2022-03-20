package helper

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
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

func createOrOpen(path string) (*os.File, error) {
	_, err := os.Stat(path)
	switch {
	case err != nil && os.IsNotExist(err):
		dir := filepath.Dir(path)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, err
			}
		}
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		return f, nil
	case err != nil:
		return nil, err
	default:
		return os.Open(path)
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

func forceWriteFile(path string, content []byte) error {
	_, err := os.Stat(path)
	switch {
	case err != nil && os.IsNotExist(err):
		parent := filepath.Dir(path)
		if _, err := os.Stat(parent); os.IsNotExist(err) {
			if err := os.MkdirAll(parent, 0755); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(content)
		if err != nil {
			return err
		}
		return nil
	case err != nil:
		return err
	default:
		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(content)
		if err != nil {
			return err
		}
		return nil
	}
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
