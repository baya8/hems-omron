package main

import (
	"flag"
	"log"
	"os"
	"time"
)

func main() {
	// フラグ設定（手動で日付指定したい場合用）
	targetDate := flag.String("date", "", "Target date (YYYYMMDD). Default is yesterday.")
	flag.Parse()

	// 環境変数から設定を取得
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "omron_user:omron_password@tcp(localhost:3306)/omron_energy?parseTime=true"
	}
	omronIP := os.Getenv("OMRON_IP")
	if omronIP == "" {
		omronIP = "192.168.50.25"
	}
	deviceID := os.Getenv("OMRON_DEVICE_ID")
	if deviceID == "" {
		deviceID = "00168978"
	}

	// クライアント初期化
	db, err := ConnectDB(dsn)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	client := &OmronClient{
		IP:       omronIP,
		DeviceID: deviceID,
	}

	// 1. 対象日の決定
	if *targetDate == "" {
		// 1:00 AM 起動を想定し、前日の日付を取得
		*targetDate = time.Now().AddDate(0, 0, -1).Format("20060102")
	}

	log.Printf("Starting batch for date: %s", *targetDate)
	
	// 2. データの取得と保存
	processDate(db, client, *targetDate)

	// 3. 過去の失敗分のリトライ
	failedDates, err := db.GetFailedDates()
	if err == nil && len(failedDates) > 0 {
		log.Printf("Retrying failed dates: %v", failedDates)
		for _, fd := range failedDates {
			// fd は YYYY-MM-DD 形式なので YYYYMMDD に変換
			dateClean := fd[0:4] + fd[5:7] + fd[8:10]
			if dateClean == *targetDate {
				continue // すでに処理済み
			}
			processDate(db, client, dateClean)
		}
	}

	log.Println("Batch finished.")
}

func processDate(db *DB, client *OmronClient, date string) {
	records, err := client.FetchHourlyData(date)
	if err != nil {
		log.Printf("Error fetching data for %s: %v", date, err)
		// 0時から23時まで一括で失敗としてマーク
		for h := 0; h < 24; h++ {
			db.UpdateFetchStatus(date, h, true, err.Error())
		}
		return
	}

	for _, r := range records {
		if err := db.SaveHourlyData(date, r); err != nil {
			log.Printf("Error saving data for %s %02d:00 : %v", date, r.Hour, err)
			db.UpdateFetchStatus(date, r.Hour, true, err.Error())
		}
	}
	log.Printf("Successfully processed data for %s", date)
}
