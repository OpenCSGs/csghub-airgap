package utils

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func GetDefaultInterface() (string, error) {
	iFaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range iFaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		if isVirtualInterface(iface.Name) {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}

		if !isPhysicalInterface(iface.Name) {
			continue
		}

		return iface.Name, nil
	}

	return "", fmt.Errorf("no available network interface found")
}

func GetInterfaceIP(ifaceName string) (string, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return "", err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		if ip != nil && ip.To4() != nil {
			return ip.String(), nil
		}
	}

	return "", fmt.Errorf("no IPv4 found for interface %s", ifaceName)
}

func isPhysicalInterface(name string) bool {
	path := filepath.Join("/sys/class/net", name, "device")
	_, err := os.Stat(path)
	return err == nil
}

func isVirtualInterface(name string) bool {
	prefixes := []string{
		"lo", "docker", "cni", "flannel", "kube", "veth",
		"br-", "virbr", "vmnet", "tun", "tap", "dummy",
	}

	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}
