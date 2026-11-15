package cross_mac

import (
	"errors"
	"net"
	"syscall"
	"time"
)

// some address on the lan that isn't us
func localNetTarget() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	myIP := conn.LocalAddr().(*net.UDPAddr).IP.To4()
	conn.Close()
	if myIP == nil {
		return nil, errors.New("no ipv4 address")
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok || !ipnet.IP.Equal(myIP) {
			continue
		}
		mask := ipnet.Mask
		if len(mask) == net.IPv6len {
			mask = mask[12:]
		}
		target := myIP.Mask(mask)
		target[3]++
		if target.Equal(myIP) {
			target[3]++
		}
		return target, nil
	}
	return nil, errors.New("couldn't find interface for " + myIP.String())
}

func sendLocalNetPacket(target net.IP) bool {
	conn, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: target, Port: 9})
	if err != nil {
		return !errors.Is(err, syscall.EHOSTUNREACH)
	}
	defer conn.Close()
	_, err = conn.Write([]byte("hi :3"))
	return !errors.Is(err, syscall.EHOSTUNREACH)
}

func requestLocalNetwork(wait time.Duration) {
	target, err := localNetTarget()
	if err != nil {
		return
	}
	deadline := time.Now().Add(wait)
	for {
		if sendLocalNetPacket(target) {
			return
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}
