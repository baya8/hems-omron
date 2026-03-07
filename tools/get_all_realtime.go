package main

import (
	"fmt"
	"net"
	"time"
)

const targetIP = "192.168.50.25:3610"

func main() {
	fmt.Printf("Fetching real-time energy data from %s...\n", targetIP)

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 3610})
	if err != nil {
		fmt.Printf("Listen Error: %v\n", err)
		return
	}
	defer conn.Close()

	// 1. 太陽光発電 (Instance 0x01) の発電量
	gen1 := getInstantaneousPower(conn, 0x02, 0x79, 0x01, 0x0011)
	// 2. 太陽光発電 (Instance 0x02) の発電量
	gen2 := getInstantaneousPower(conn, 0x02, 0x79, 0x02, 0x0012)
	// 3. 分電盤 (Instance 0x01) の主幹電力 (買電/売電)
	grid := getSignedInstantaneousPower(conn, 0x02, 0x87, 0x01, 0xE0, 0x0013)

	fmt.Println("\n--- Real-time Energy Status ---")
	
	solarGen := gen1
	if gen2 > gen1 { solarGen = gen2 }
	
	fmt.Printf("  太陽光発電 (Solar Generation): %d W\n", solarGen)

	if grid != 0 {
		if grid > 0 {
			fmt.Printf("  電力会社から購入 (Purchasing): %d W\n", grid)
		} else {
			fmt.Printf("  電力会社へ売電 (Selling)   : %d W\n", -grid)
		}
	} else {
		fmt.Printf("  電力会社との売買: 0 W (または取得失敗)\n")
	}

	// 消費電力の計算
	consumption := int32(solarGen) + grid
	fmt.Printf("  現在の消費電力 (Consumption) : %d W\n", consumption)
	
	fmt.Println("-------------------------------")
}

func getInstantaneousPower(conn *net.UDPConn, cg, cc byte, id byte, tid uint16) int {
	packet := []byte{
		0x10, 0x81, 
		byte(tid >> 8), byte(tid & 0xFF), 
		0x0E, 0xF0, 0x01, cg, cc, id, 0x62, 0x01, 0xE0, 0x00,
	}
	remoteAddr, _ := net.ResolveUDPAddr("udp", targetIP)
	conn.WriteToUDP(packet, remoteAddr)

	conn.SetReadDeadline(time.Now().Add(1500 * time.Millisecond))
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		fmt.Printf("[Debug] Class %02X%02X Instance %02X: %v\n", cg, cc, id, err)
		return 0
	}

	if n >= 14 && buffer[10] == 0x72 {
		return int(buffer[14])<<8 | int(buffer[15])
	}
	fmt.Printf("[Debug] Class %02X%02X Instance %02X: ESV 0x%02X\n", cg, cc, id, buffer[10])
	return 0
}

func getSignedInstantaneousPower(conn *net.UDPConn, cg, cc byte, id byte, epc byte, tid uint16) int32 {
	packet := []byte{
		0x10, 0x81, 
		byte(tid >> 8), byte(tid & 0xFF), 
		0x0E, 0xF0, 0x01, cg, cc, id, 0x62, 0x01, epc, 0x00,
	}
	remoteAddr, _ := net.ResolveUDPAddr("udp", targetIP)
	conn.WriteToUDP(packet, remoteAddr)

	conn.SetReadDeadline(time.Now().Add(1500 * time.Millisecond))
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		fmt.Printf("[Debug] Class %02X%02X Instance %02X: %v\n", cg, cc, id, err)
		return 0
	}

	if n >= 14 && buffer[10] == 0x72 {
		pdc := int(buffer[13])
		if pdc == 4 {
			return int32(uint32(buffer[14])<<24 | uint32(buffer[15])<<16 | uint32(buffer[16])<<8 | uint32(buffer[17]))
		} else if pdc == 2 {
			return int32(int16(uint16(buffer[14])<<8 | uint16(buffer[15])))
		}
	}
	fmt.Printf("[Debug] Class %02X%02X Instance %02X: ESV 0x%02X\n", cg, cc, id, buffer[10])
	return 0
}
