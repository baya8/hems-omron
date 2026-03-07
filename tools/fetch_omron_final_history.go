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
	// 2026年3月4日のデータを取得してみる（HARファイルの例に合わせて）
	date := "20260304"
	
	// HARファイルでグラフ表示時に使われていたクエリ
	// 0&0&... の部分は統計、IG0&ISI&IBI の部分が詳細データを指すはずです
	path := fmt.Sprintf("/getinfo.cgi?%s&0&%s&%s&IG0&ISI&IBI", deviceID, date, date)
	
	fmt.Printf("Accessing OMRON Data API...\nPath: %s\n", path)

	rawResponse := fetchRaw(path)
	
	// {} の中身を抽出
	start := strings.Index(rawResponse, "{")
	end := strings.LastIndex(rawResponse, "}")
	
	if start != -1 && end > start {
		data := rawResponse[start+1 : end]
		fmt.Println("\n--- Received Data (Raw) ---")
		fmt.Println(data)

		parts := strings.Split(data, "&")
		fmt.Printf("\n--- Parsed Analysis (%s) ---\n", date)
		
		for i, p := range parts {
			// カンマが含まれているパートを探す
			if strings.Contains(p, ",") {
				nums := strings.Split(p, ",")
				fmt.Printf("\n[パート %d] %d 個の連続した数値が見つかりました:\n", i, len(nums))
				
				// 48個あれば、それが30分ごとのデータ
				for j, n := range nums {
					if j >= 48 { break }
					hour := j / 2
					min := (j % 2) * 30
					fmt.Printf("  %02d:%02d -> %s\n", hour, min, n)
				}
			} else {
				fmt.Printf("パート %d: %s\n", i, p)
			}
		}
	} else {
		fmt.Println("No data found in braces {}.")
		fmt.Println(rawResponse)
	}
}

func fetchRaw(path string) string {
	conn, err := net.DialTimeout("tcp", targetIP+":80", 5*time.Second)
	if err != nil { return "Connection Error" }
	defer conn.Close()

	request := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", path, targetIP)
	conn.Write([]byte(request))

	var response strings.Builder
	io.Copy(&response, conn)
	return response.String()
}
