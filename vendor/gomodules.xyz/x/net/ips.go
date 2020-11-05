package net

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strings"

	"gomodules.xyz/sets"
)

type IPRange struct {
	from net.IP
	to   net.IP
}

func NewIPRange(from string, to string) IPRange {
	return IPRange{net.ParseIP(from).To4(), net.ParseIP(to).To4()}
}

var privateIPRanges = []IPRange{
	NewIPRange("10.0.0.0", "10.255.255.255"),
	NewIPRange("172.16.0.0", "172.31.255.255"),
	NewIPRange("192.168.0.0", "192.168.255.255"),
}

func IsPrivateIP(ip net.IP) bool {
	for _, r := range privateIPRanges {
		if bytes.Compare(ip, r.from) >= 0 && bytes.Compare(ip, r.to) <= 0 {
			return true
		}
	}
	return false
}

var (
	knownLocalBridges = regexp.MustCompile(`^(docker|cbr|cni)[0-9]+$`)

	InterfaceDownErr       = errors.New("interface down")
	LoopbackInterfaceErr   = errors.New("loopback interface")
	KnownLocalInterfaceErr = errors.New("known local interface")
	NotFoundErr            = errors.New("no IPV4 address found")
)

/*
NodeIP returns a IPv4 address for a given set of interface names. It always prefers a private IP over a public IP.
If no interface name is given, all interfaces are checked.
*/
func NodeIP(interfaceName ...string) (string, net.IP, error) {
	var err error
	var ifaces []net.Interface

	if len(interfaceName) == 0 {
		ifaces, err = net.Interfaces()
		if err != nil {
			return "", nil, err
		}
	} else {
		ifaces = make([]net.Interface, len(interfaceName))
		for i, name := range interfaceName {
			d, err := net.InterfaceByName(name)
			if err != nil {
				return name, nil, err
			}
			ifaces[i] = *d
		}
	}

	type ipData struct {
		ip    net.IP
		iface string
	}
	internalIPs := make([]ipData, 0)
	externalIPs := make([]ipData, 0)
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			if len(ifaces) == 1 {
				return iface.Name, nil, InterfaceDownErr
			} else {
				continue
			}
		}
		if iface.Flags&net.FlagLoopback != 0 {
			if len(ifaces) == 1 {
				return iface.Name, nil, LoopbackInterfaceErr
			} else {
				continue
			}
		}
		if knownLocalBridges.MatchString(iface.Name) {
			if len(ifaces) == 1 {
				return iface.Name, nil, KnownLocalInterfaceErr
			} else {
				continue
			}
		}
		addrs, err := iface.Addrs()
		if err != nil {
			if len(ifaces) == 1 {
				return iface.Name, nil, err
			} else {
				continue
			}
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // Not an ipv4 address
			}
			if IsPrivateIP(ip) {
				internalIPs = append(internalIPs, ipData{ip: ip, iface: iface.Name})
			} else {
				externalIPs = append(externalIPs, ipData{ip: ip, iface: iface.Name})
			}
		}
	}
	if len(internalIPs) > 0 {
		return internalIPs[0].iface, internalIPs[0].ip, nil
	} else if len(externalIPs) > 0 {
		return externalIPs[0].iface, externalIPs[0].ip, nil
	} else {
		return "", nil, NotFoundErr
	}
}

func detectIPs(routable bool) ([]string, []string, error) {
	internalIPs := sets.NewString()
	externalIPs := sets.NewString()

	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		if knownLocalBridges.MatchString(iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			if IsPrivateIP(ip) {
				internalIPs.Insert(ip.String())
			} else {
				externalIPs.Insert(ip.String())
			}
		}
	}
	// If host is not assigned public IP directly, detect IP based on client ip
	if len(externalIPs) == 0 && routable {
		if resp, err := http.Get("https://ipinfo.io/ip"); err == nil {
			defer resp.Body.Close()
			if bytes, err := ioutil.ReadAll(resp.Body); err == nil {
				ip := net.ParseIP(strings.TrimSpace(string(bytes)))
				if ip != nil {
					ip = ip.To4()
				}
				if ip != nil {
					externalIPs.Insert(ip.String())
				}
			}
		}
	}
	if len(internalIPs)+len(externalIPs) == 0 {
		return nil, nil, NotFoundErr
	}
	return externalIPs.List(), internalIPs.List(), nil
}

// RoutableIPs returns routable public and private IPs associated with current host.
// It will also use https://ipinfo.io/ip to detect public IP, if no public IP is assigned to a host interface.
func RoutableIPs() ([]string, []string, error) {
	return detectIPs(true)
}

// HostIPs returns public and private IPs assigned to various interfaces on current host.
func HostIPs() ([]string, []string, error) {
	return detectIPs(false)
}
