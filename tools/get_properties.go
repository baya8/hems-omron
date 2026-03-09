package main

import (
	"fmt"
	"net"
	"time"
)

const (
	targetIP = "0.0.0.0:3610"
)

func main() {
	fmt.Printf("Querying property map (0x0287) for %s...\n", targetIP)

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 3610})
	if err != nil {
		fmt.Printf("Listen error: %v\n", err)
		return
	}
	defer conn.Close()

	requestPacket := []byte{
		0x10, 0x81, 0x00, 0x0C, 0x0E, 0xF0, 0x01, 0x02, 0x87, 0x01, 0x62, 0x01, 0x9F, 0x00,
	}

	remoteAddr, _ := net.ResolveUDPAddr("udp", targetIP)
	conn.WriteToUDP(requestPacket, remoteAddr)

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		fmt.Printf("Read error: %v\n", err)
		return
	}

	if n >= 14 && buffer[10] == 0x72 {
		pdc := int(buffer[13])
		data := buffer[14 : 14+pdc]
		fmt.Println("\n--- Supported Get Properties (EPCs) for 0x0287 ---")
		
		if pdc < 16 {
			for i := 1; i < len(data); i++ {
				printEPCInfo(data[i])
			}
		} else {
			for i := 1; i < len(data); i++ {
				for bit := 0; bit < 8; bit++ {
					if (data[i] >> uint(bit) & 1) == 1 {
						epc := byte(0x80 + (i-1)*8 + bit)
						printEPCInfo(epc)
					}
				}
			}
		}
	} else {
		fmt.Printf("Failed to get property map (ESV: 0x%02X)\n", buffer[10])
	}
}

func printEPCInfo(epc byte) {
	name := "Unknown"
	switch epc {
	case 0x80: name = "動作状態 (Operation Status)"
	case 0x81: name = "設置場所 (Installation Location)"
	case 0x82: name = "規格Version情報 (Standard Version)"
	case 0x83: name = "識別番号 (Identification Number)"
	case 0x88: name = "異常発生状態 (Fault Status)"
	case 0x8A: name = "メーカーコード (Manufacturer Code)"
	case 0x9D: name = "状態変化通知プロパティマップ (Status Change Map)"
	case 0x9E: name = "Setプロパティマップ (Set Property Map)"
	case 0x9F: name = "Getプロパティマップ (Get Property Map)"
	case 0xB0: name = "瞬時電力計測値 (Instantaneous Power [Total])"
	case 0xB1: name = "積算電力量計測値 (Cumulative Power [Total])"
	case 0xB2: name = "積算電力量単位 (Unit for B1)"
	case 0xB3: name = "積算計測値履歴 (History for B1)"
	case 0xD3: name = "係数 (Coefficient)"
	case 0xE0: name = "瞬時電力計測値 (主幹) (Main Channel Power)"
	case 0xE2: name = "積算電力量単位 (主幹) (Unit for E1/E3)"
	}
	fmt.Printf("  0x%02X: %s\n", epc, name)
}
