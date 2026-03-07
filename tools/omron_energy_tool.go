package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

const (
	targetIP = "192.168.50.25"
	deviceID = "00168978"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run omron_energy_tool.go monitor             - リアルタイム監視")
		fmt.Println("  go run omron_energy_tool.go export <YYYYMM>       - 指定した月のデータを日別CSV保存")
		fmt.Println("  go run omron_energy_tool.go export-hourly <YYYYMMDD> - 指定した日のデータを1時間別CSV保存")
		return
	}

	mode := os.Args[1]
	switch mode {
	case "monitor":
		runMonitor()
	case "export":
		if len(os.Args) < 3 {
			fmt.Println("Error: 年月(YYYYMM)を指定してください。例: 202603")
			return
		}
		runExport(os.Args[2], false)
	case "export-hourly":
		if len(os.Args) < 3 {
			fmt.Println("Error: 日付(YYYYMMDD)を指定してください。例: 20260306")
			return
		}
		runExport(os.Args[2], true)
	default:
		fmt.Println("Unknown mode:", mode)
	}
}

// リアルタイム監視（ECHONET Lite 経由）
func runMonitor() {
	fmt.Printf("Starting Real-time Monitor (Target: %s)\n", targetIP)
	conn, _ := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 3610})
	defer conn.Close()

	for {
		solar1 := getSolarPower(conn, 0x01)
		solar2 := getSolarPower(conn, 0x02)
		totalSolar := solar1 + solar2

		gridRaw := getRawEL(conn, 0x02, 0x87, 0x01, 0xC6)
		grid := int32(0)
		if len(gridRaw) >= 4 {
			grid = int32(int16(uint16(gridRaw[2])<<8 | uint16(gridRaw[3])))
		}

		consumption := int32(totalSolar) + grid
		fmt.Printf("[%s] 発電: %4dW (①%4d+②%4d) | 買/売: %4dW | 消費: %4dW\n", 
			time.Now().Format("15:04:05"), totalSolar, solar1, solar2, grid, consumption)
		
		time.Sleep(5 * time.Second)
	}
}

// データのCSVエクスポート（HTTP CGI 経由）
func runExport(date string, hourly bool) {
	var start, end, filename string
	if hourly {
		start = date + "00"
		end   = date + "23"
		filename = "energy_hourly_" + date + ".csv"
	} else {
		start = date + "01"
		end   = date + "31"
		filename = "energy_daily_" + date + ".csv"
	}

	// IG1: パワコン1, IG2: パワコン2, IG0: 合計発電, ISI: 売電, IBI: 買電
	path := fmt.Sprintf("/getinfo.cgi?%s&0&%s&%s&IG1&IG2&IG0&ISI&IBI", deviceID, start, end)
	
	fmt.Printf("Exporting detailed data for %s...\n", date)
	raw := fetchRawHTTP(path)
	
	startIdx := strings.Index(raw, "{")
	endIdx := strings.LastIndex(raw, "}")
	if startIdx == -1 {
		fmt.Println("Error: データの取得に失敗しました。")
		return
	}

	data := raw[startIdx+1 : endIdx]
	parts := strings.Split(data, "&")
	// インデックス: 3:IG1, 4:IG2, 5:IG0, 6:ISI, 7:IBI (CGIの仕様による)
	if len(parts) < 8 {
		fmt.Println("Error: データ形式が不正です。")
		return
	}

	gens1 := strings.Split(parts[3], ",")
	gens2 := strings.Split(parts[4], ",")
	gensT := strings.Split(parts[5], ",")
	sells := strings.Split(parts[6], ",")
	buys := strings.Split(parts[7], ",")

	file, _ := os.Create(filename)
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"Time", "PCS1(Wh)", "PCS2(Wh)", "TotalGen(Wh)", "Sold(Wh)", "Bought(Wh)", "Consumption(Wh)"}
	if !hourly { header[0] = "Date" }
	writer.Write(header)

	for i := 0; i < len(gensT); i++ {
		var timeLabel string
		if hourly {
			timeLabel = fmt.Sprintf("%02d:00", i)
		} else {
			timeLabel = fmt.Sprintf("%s%02d", date, i+1)
		}

		g1, g2, gT, s, b := 0, 0, 0, 0, 0
		if i < len(gens1) { fmt.Sscanf(gens1[i], "%d", &g1) }
		if i < len(gens2) { fmt.Sscanf(gens2[i], "%d", &g2) }
		if i < len(gensT) { fmt.Sscanf(gensT[i], "%d", &gT) }
		if i < len(sells) { fmt.Sscanf(sells[i], "%d", &s) }
		if i < len(buys) { fmt.Sscanf(buys[i], "%d", &b) }
		
		c := gT + b - s
		
		row := []string{
			timeLabel, 
			fmt.Sprintf("%d", g1), 
			fmt.Sprintf("%d", g2), 
			fmt.Sprintf("%d", gT), 
			fmt.Sprintf("%d", s), 
			fmt.Sprintf("%d", b), 
			fmt.Sprintf("%d", c),
		}
		writer.Write(row)
	}

	fmt.Printf("SUCCESS: %s に保存しました。\n", filename)
}

// 共通関数
func getSolarPower(conn *net.UDPConn, instanceID byte) int {
	packet := []byte{0x10, 0x81, 0x00, 0x01, 0x0E, 0xF0, 0x01, 0x02, 0x79, instanceID, 0x62, 0x01, 0xE0, 0x00}
	remoteAddr, _ := net.ResolveUDPAddr("udp", targetIP+":3610")
	conn.WriteToUDP(packet, remoteAddr)
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err == nil && n >= 14 && buffer[10] == 0x72 {
		return int(buffer[14])<<8 | int(buffer[15])
	}
	return 0
}

func getRawEL(conn *net.UDPConn, cg, cc, id, epc byte) []byte {
	packet := []byte{0x10, 0x81, 0x00, 0x02, 0x0E, 0xF0, 0x01, cg, cc, id, 0x62, 0x01, epc, 0x00}
	remoteAddr, _ := net.ResolveUDPAddr("udp", targetIP+":3610")
	conn.WriteToUDP(packet, remoteAddr)
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err == nil && n >= 14 && buffer[10] == 0x72 {
		return buffer[14 : 14+int(buffer[13])]
	}
	return nil
}

func fetchRawHTTP(path string) string {
	conn, err := net.DialTimeout("tcp", targetIP+":80", 3*time.Second)
	if err != nil { return "" }
	defer conn.Close()
	fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", path, targetIP)
	var sb strings.Builder
	io.Copy(&sb, conn)
	return sb.String()
}
