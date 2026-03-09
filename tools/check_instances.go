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
	fmt.Printf("Querying all instances on %s...\n", targetIP)

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 3610})
	if err != nil {
		fmt.Printf("Listen error: %v\n", err)
		return
	}
	defer conn.Close()

	// 0xD6 (自ノードインスタンスリストS) をノードプロファイルオブジェクト(0x0EF001)にリクエスト
	requestPacket := []byte{
		0x10, 0x81,         // EHD
		0x00, 0x0A,         // TID
		0x0E, 0xF0, 0x01,   // SEOJ (Node Profile)
		0x0E, 0xF0, 0x01,   // DEOJ (Node Profile)
		0x62,               // ESV (Get)
		0x01,               // OPC
		0xD6, 0x00,         // EPC: 0xD6 (Self-node instance list S)
	}

	remoteAddr, _ := net.ResolveUDPAddr("udp", targetIP)
	conn.WriteToUDP(requestPacket, remoteAddr)

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		fmt.Printf("Read error: %v\n", err)
		return
	}

	if n >= 14 && buffer[10] == 0x72 {
		pdc := int(buffer[13])
		data := buffer[14 : 14+pdc]
		
		fmt.Printf("\n--- Found Instances on this Device ---\n")
		numInstances := int(data[0])
		fmt.Printf("Number of instances: %d\n", numInstances)

		for i := 0; i < numInstances; i++ {
			offset := 1 + (i * 3)
			if offset+3 > len(data) { break }
			
			classGroup := data[offset]
			classCode := data[offset+1]
			instanceID := data[offset+2]
			
			className := getClassName(classGroup, classCode)
			fmt.Printf("  [%d] Class: 0x%02X%02X, Instance: 0x%02X (%s)\n", i+1, classGroup, classCode, instanceID, className)
		}
	} else {
		fmt.Printf("Failed to get instance list (ESV: 0x%02X)\n", buffer[10])
	}
}

func getClassName(cg, cc byte) string {
	switch {
	case cg == 0x02 && cc == 0x79: return "太陽光発電 (Photovoltaic Generation)"
	case cg == 0x02 && cc == 0x88: return "スマート電力量メータ (Smart Electric Energy Meter)"
	case cg == 0x02 && cc == 0x87: return "分電盤計測 (Distribution Panel Metering)"
	case cg == 0x02 && cc == 0x7D: return "蓄電池 (Storage Battery)"
	case cg == 0x0E && cc == 0xF0: return "ノードプロファイル (Node Profile)"
	default: return "その他のデバイス"
	}
}
