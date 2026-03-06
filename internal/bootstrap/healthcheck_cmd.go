package bootstrap

import (
	"net/http"
	"strings"
	"time"
)

func HealthcheckExitCode(addr string) int {
	url := healthcheckURL(addr)
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return 1
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		return 1
	}
	return 0
}

func healthcheckURL(addr string) string {
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/") + "/healthz"
	}
	if strings.HasPrefix(addr, ":") {
		return "http://127.0.0.1" + addr + "/healthz"
	}
	return "http://" + addr + "/healthz"
}
