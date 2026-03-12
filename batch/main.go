package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	// フラグ設定
	targetDate := flag.String("date", "", "Target date (YYYYMMDD). Default is yesterday.")
	startDate := flag.String("start", "", "Start date for bulk import (YYYYMMDD).")
	endDate := flag.String("end", "", "End date for bulk import (YYYYMMDD). Default is yesterday.")
	flag.Parse()

	// 必須の環境変数を取得
	dsn, err := requireEnv("DB_DSN")
	if err != nil {
		log.Fatal(err)
	}
	omronIP, err := requireEnv("OMRON_IP")
	if err != nil {
		log.Fatal(err)
	}
	deviceID, err := requireEnv("OMRON_DEVICE_ID")
	if err != nil {
		log.Fatal(err)
	}

	// DB接続とクライアント初期化
	db, err := ConnectDB(dsn)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	client := &OmronClient{
		IP:       omronIP,
		DeviceID: deviceID,
	}

	// プロセッサの生成
	processor := NewBatchProcessor(db, client)

	// 1. バックフィルモード（範囲指定）
	if *startDate != "" {
		if *endDate == "" {
			*endDate = time.Now().AddDate(0, 0, -1).Format("20060102")
		}
		processor.RunBackfill(*startDate, *endDate)
		log.Println("Bulk import finished.")
		return
	}

	// 2. 単発取得モード
	if *targetDate == "" {
		// デフォルトは「今日」を取得対象とする（日中の進捗確認のため）
		*targetDate = time.Now().In(jst).Format("20060102")
	}

	log.Printf("Starting batch for date: %s", *targetDate)
	processor.ProcessDate(*targetDate, true)

	// 当日分だけでなく、前日分も確実に更新する（前日の最終データ確定のため）
	yesterday := time.Now().In(jst).AddDate(0, 0, -1).Format("20060102")
	if *targetDate != yesterday {
		log.Printf("Also updating yesterday: %s", yesterday)
		processor.ProcessDate(yesterday, true)
	}

	// 3. 過去の失敗分のリトライ
	processor.RetryFailedDates(*targetDate)

	log.Println("Batch finished.")
}

// requireEnv は必須の環境変数を取得し、存在しない場合はエラーを返す
func requireEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return value, nil
}
