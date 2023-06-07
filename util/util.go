package util

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"strings"
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

func CountWords(s string) int {
	return len(strings.Fields(s))
}

// 判断所给路径文件/文件夹是否存在
func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
