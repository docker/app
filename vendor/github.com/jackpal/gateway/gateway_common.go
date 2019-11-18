package gateway

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"strconv"
	"strings"
)

var errNoGateway = errors.New("no gateway found")

func parseWindowsRoutePrint(output []byte) (net.IP, error) {
	// Windows route output format is always like this:
	// ===========================================================================
	// Interface List
	// 8 ...00 12 3f a7 17 ba ...... Intel(R) PRO/100 VE Network Connection
	// 1 ........................... Software Loopback Interface 1
	// ===========================================================================
	// IPv4 Route Table
	// ===========================================================================
	// Active Routes:
	// Network Destination        Netmask          Gateway       Interface  Metric
	//           0.0.0.0          0.0.0.0      192.168.1.1    192.168.1.100     20
	// ===========================================================================
	//
	// Windows commands are localized, so we can't just look for "Active Routes:" string
	// I'm trying to pick the active route,
	// then jump 2 lines and pick the third IP
	// Not using regex because output is quite standard from Windows XP to 8 (NEEDS TESTING)
	lines := strings.Split(string(output), "\n")
	sep := 0
	for idx, line := range lines {
		if sep == 3 {
			// We just entered the 2nd section containing "Active Routes:"
			if len(lines) <= idx+2 {
				return nil, errNoGateway
			}

			fields := strings.Fields(lines[idx+2])
			if len(fields) < 3 {
				return nil, errNoGateway
			}

			ip := net.ParseIP(fields[2])
			if ip != nil {
				return ip, nil
			}
		}
		if strings.HasPrefix(line, "=======") {
			sep++
			continue
		}
	}
	return nil, errNoGateway
}


func parseLinuxProcNetRoute(f []byte) (net.IP, error) {
	/* /proc/net/route file:
	   Iface   Destination Gateway     Flags   RefCnt  Use Metric  Mask
	   eno1    00000000    C900A8C0    0003    0   0   100 00000000    0   00
	   eno1    0000A8C0    00000000    0001    0   0   100 00FFFFFF    0   00
	*/
	const (
		sep   = "\t" // field separator
		field = 2    // field containing hex gateway address
	)
	scanner := bufio.NewScanner(bytes.NewReader(f))
	for scanner.Scan() {
		// Skip header line
		if !scanner.Scan() {
			return nil, errors.New("Invalid linux route file")
		}

		// get field containing gateway address
		tokens := strings.Split(scanner.Text(), sep)
		if len(tokens) <= field {
			return nil, errors.New("Invalid linux route file")
		}
		gatewayHex := "0x" + tokens[field]

		// cast hex address to uint32
		d, _ := strconv.ParseInt(gatewayHex, 0, 64)
		d32 := uint32(d)

		// make net.IP address from uint32
		ipd32 := make(net.IP, 4)
		binary.LittleEndian.PutUint32(ipd32, d32)

		// format net.IP to dotted ipV4 string
		return net.IP(ipd32), nil
	}
	return nil, errors.New("Failed to parse linux route file")
}

func parseDarwinRouteGet(output []byte) (net.IP, error) {
	// Darwin route out format is always like this:
	//    route to: default
	// destination: default
	//        mask: default
	//     gateway: 192.168.1.1
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "gateway:" {
			ip := net.ParseIP(fields[1])
			if ip != nil {
				return ip, nil
			}
		}
	}

	return nil, errNoGateway
}

func parseBSDSolarisNetstat(output []byte) (net.IP, error) {
	// netstat -rn produces the following on FreeBSD:
	// Routing tables
	//
	// Internet:
	// Destination        Gateway            Flags      Netif Expire
	// default            10.88.88.2         UGS         em0
	// 10.88.88.0/24      link#1             U           em0
	// 10.88.88.148       link#1             UHS         lo0
	// 127.0.0.1          link#2             UH          lo0
	//
	// Internet6:
	// Destination                       Gateway                       Flags      Netif Expire
	// ::/96                             ::1                           UGRS        lo0
	// ::1                               link#2                        UH          lo0
	// ::ffff:0.0.0.0/96                 ::1                           UGRS        lo0
	// fe80::/10                         ::1                           UGRS        lo0
	// ...
	outputLines := strings.Split(string(output), "\n")
	for _, line := range outputLines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "default" {
			ip := net.ParseIP(fields[1])
			if ip != nil {
				return ip, nil
			}
		}
	}

	return nil, errNoGateway
}
