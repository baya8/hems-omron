package main

import (
	"fmt"
	"net"
	"time"
)

const (
	multicastAddr = "224.0.23.0:3610"
	port          = ":3610"
)

// ECHONET Lite Discovery Packet (Get Self-node instance list S)
var discoveryPacket = []byte{
	0x10, 0x81, // EHD1, EHD2 (ECHONET Lite)
	0x00, 0x01, // TID (Transaction ID)
	0x0E, 0xF0, 0x01, // SEOJ (Source Object: Node Profile)
	0x0E, 0xF0, 0x01, // DEOJ (Destination Object: Node Profile)
	0x62,       // ESV (Service Code: Get)
	0x01,       // OPC (Number of properties)
	0xD6,       // EPC (Property Code: Self-node instance list S)
	0x00,       // PDC (Property Data Counter)
}

func main() {
	fmt.Println("ECHONET Lite Device Discovery starting...")

	// 1. Setup UDP connection for receiving responses
	addr, err := net.ResolveUDPAddr("udp", port)
	if err != nil {
		panic(err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// 2. Send Multicast Discovery Packet
	mAddr, err := net.ResolveUDPAddr("udp", multicastAddr)
	if err != nil {
		panic(err)
	}

	_, err = conn.WriteToUDP(discoveryPacket, mAddr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Sent discovery packet to %s\n", multicastAddr)

	// 3. Listen for responses
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buffer := make([]byte, 1024)

	fmt.Println("Waiting for responses (5 seconds)...")
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			fmt.Printf("Error reading: %v\n", err)
			continue
		}

		if n < 12 {
			continue // Too short for ECHONET Lite
		}

		// Basic validation: Check EHD (0x10 0x81)
		if buffer[0] == 0x10 && buffer[1] == 0x81 {
			esv := buffer[10]
			fmt.Printf("\n[Device Found]\n")
			fmt.Printf("IP Address: %s\n", remoteAddr.IP.String())
			fmt.Printf("ESV: 0x%02X\n", esv)
			
			// Parse Self-node instance list if EPC is 0xD6
			if n > 12 && buffer[11] == 0xD6 {
				pdc := int(buffer[12])
				if pdc > 0 && n >= 13+pdc {
					instances := buffer[13 : 13+pdc]
					fmt.Printf("Instances Data (Hex): %X\n", instances)
					// The first byte of instances is the number of instances
					// Each instance is 3 bytes (ClassGroup, ClassCode, InstanceID)
				}
			}
		}
	}

	fmt.Println("\nDiscovery finished.")
}
