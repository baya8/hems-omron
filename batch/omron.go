package main

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type OmronClient struct {
	IP       string
	DeviceID string
}

func (c *OmronClient) FetchHourlyData(date string) ([]HourlyRecord, error) {
	// IG1: PCS1, IG2: PCS2, IG0: Total, ISI: Sold, IBI: Bought
	start := date + "00"
	end := date + "23"
	path := fmt.Sprintf("/getinfo.cgi?%s&0&%s&%s&IG1&IG2&IG0&ISI&IBI", c.DeviceID, start, end)

	raw, err := c.fetchRawHTTP(path)
	if err != nil {
		return nil, err
	}

	startIdx := strings.Index(raw, "{")
	endIdx := strings.LastIndex(raw, "}")
	if startIdx == -1 || endIdx <= startIdx {
		return nil, fmt.Errorf("invalid response format")
	}

	data := raw[startIdx+1 : endIdx]
	parts := strings.Split(data, "&")
	if len(parts) < 8 {
		return nil, fmt.Errorf("insufficient data parts: %d", len(parts))
	}

	// 3:IG1, 4:IG2, 5:IG0, 6:ISI, 7:IBI
	gens1 := strings.Split(parts[3], ",")
	gens2 := strings.Split(parts[4], ",")
	gensT := strings.Split(parts[5], ",")
	sells := strings.Split(parts[6], ",")
	buys := strings.Split(parts[7], ",")

	var records []HourlyRecord
	for i := 0; i < 24; i++ {
		var g1, g2, gT, s, b int
		if i < len(gens1) { fmt.Sscanf(gens1[i], "%d", &g1) }
		if i < len(gens2) { fmt.Sscanf(gens2[i], "%d", &g2) }
		if i < len(gensT) { fmt.Sscanf(gensT[i], "%d", &gT) }
		if i < len(sells) { fmt.Sscanf(sells[i], "%d", &s) }
		if i < len(buys) { fmt.Sscanf(buys[i], "%d", &b) }

		records = append(records, HourlyRecord{
			Hour:        i,
			Gen1:        g1,
			Gen2:        g2,
			GenTotal:    gT,
			Selling:     s,
			Buying:      b,
			Consumption: gT + b - s,
		})
	}

	return records, nil
}

func (c *OmronClient) fetchRawHTTP(path string) (string, error) {
	conn, err := net.DialTimeout("tcp", c.IP+":80", 5*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", path, c.IP)
	
	var sb strings.Builder
	_, err = io.Copy(&sb, conn)
	if err != nil {
		return "", err
	}
	
	return sb.String(), nil
}
