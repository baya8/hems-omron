package main

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

const (
	targetIP = "192.168.50.25"
	deviceID = "00168978"
)

func main() {
	// 調査したい日付 (例: 2026年3月6日)
	day := "20260306"
	
	// 1時間ごとのデータを取得するための時間指定 (YYYYMMDD00 から YYYYMMDD23)
	start := day + "00"
	end   := day + "23"
	
	// 発電量(IG0)、売電量(ISI)、買電量(IBI) をリクエスト
	path := fmt.Sprintf("/getinfo.cgi?%s&0&%s&%s&IG0&ISI&IBI", deviceID, start, end)
	
	fmt.Printf("Fetching hourly data for %s via TCP...\n", day)
	fmt.Printf("URL: http://%s%s\n", targetIP, path)

	raw := fetchRawHTTP(path)
	
	startIdx := strings.Index(raw, "{")
	endIdx := strings.LastIndex(raw, "}")
	
	if startIdx != -1 && endIdx > startIdx {
		data := raw[startIdx+1 : endIdx]
		fmt.Println("\n--- Received Data ---")
		
		parts := strings.Split(data, "&")
		// オムロンの形式では、時間指定リクエストの場合、& で区切られたパートにカンマ区切りの24個（または48個）の数値が入るはずです
		for i, p := range parts {
			if strings.Contains(p, ",") {
				vals := strings.Split(p, ",")
				itemLabel := getItemLabel(i)
				fmt.Printf("\n[%s] %d 個のデータが見つかりました:\n", itemLabel, len(vals))
				
				for h, v := range vals {
					// 24個なら1時間ごと、48個なら30分ごと
					if len(vals) == 24 {
						fmt.Printf("  %02d:00 : %s Wh\n", h, v)
					} else {
						fmt.Printf("  %02d:%02d : %s Wh\n", h/2, (h%2)*30, v)
					}
				}
			} else {
				fmt.Printf("Part %d: %s\n", i, p)
			}
		}
	} else {
		fmt.Println("No valid data found.")
		fmt.Println(raw)
	}
}

func getItemLabel(index int) string {
	switch index {
	case 3: return "発電量 (IG0)"
	case 4: return "売電量 (ISI)"
	case 5: return "買電量 (IBI)"
	default: return fmt.Sprintf("項目 %d", index)
	}
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
