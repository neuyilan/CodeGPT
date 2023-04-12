package util

import (
	"errors"
	"net"
	"os/exec"
)

// IsCommandAvailable check command exits.
func IsCommandAvailable(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func GetClientIp() (string, error) {
	addresses, err := net.InterfaceAddrs()

	if err != nil {
		return "", err
	}

	for _, address := range addresses {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}

		}
	}

	return "", errors.New("can not find the client ip addresses")
}
