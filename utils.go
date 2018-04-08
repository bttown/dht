package dht

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

func IsOnline(ip string, port int) bool {
	addr := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		return false
	}

	defer conn.Close()
	return true
}

func GetMyIP() (string, error) {
	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	req, _ := http.NewRequest(http.MethodGet, "http://ifconfig.me", nil)
	req.Header.Set("User-Agent", "curl/7.29.0")

	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return "", err
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.Trim(string(b), "\n"), err
}
