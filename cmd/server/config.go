package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

const defaultAddr = "127.0.0.1:19081"

func resolveAddr(flagAddr, portValue string) (string, error) {
	addr := strings.TrimSpace(flagAddr)
	if addr == "" {
		if strings.TrimSpace(portValue) == "" {
			addr = defaultAddr
		} else {
			port, err := parsePort(portValue)
			if err != nil {
				return "", fmt.Errorf("PORT无效: %w", err)
			}
			addr = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
		}
	}
	host, rawPort, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("监听地址无效: %w", err)
	}
	if host != "127.0.0.1" && host != "localhost" {
		return "", fmt.Errorf("监听地址必须绑定127.0.0.1或localhost")
	}
	if _, err := parsePort(rawPort); err != nil {
		return "", fmt.Errorf("监听端口无效: %w", err)
	}
	return addr, nil
}

func parsePort(value string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("端口必须是1到65535之间的整数")
	}
	return port, nil
}
