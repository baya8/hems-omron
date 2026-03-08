package main

import (
	"log"
	"time"
)

type BatchProcessor struct {
	DB     *DB
	Client *OmronClient
}

func NewBatchProcessor(db *DB, client *OmronClient) *BatchProcessor {
	return &BatchProcessor{
		DB:     db,
		Client: client,
	}
}

// RunBackfill は指定された範囲のデータを一括取得する（失敗しても fetch_status には記録しない）
func (p *BatchProcessor) RunBackfill(start, end string) {
	startTime, err := time.Parse("20060102", start)
	if err != nil {
		log.Fatalf("Invalid start date: %v", err)
	}
	endTime, err := time.Parse("20060102", end)
	if err != nil {
		log.Fatalf("Invalid end date: %v", err)
	}

	log.Printf("Running backfill from %s to %s...", start, end)

	for d := startTime; !d.After(endTime); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("20060102")

		// すでに取得済みならスキップ
		isFetched, err := p.DB.IsDateFetched(dateStr)
		if err == nil && isFetched {
			log.Printf("Date %s : skipped", dateStr)
			continue
		}

		// バックフィル時は失敗を記録しない (false)
		p.ProcessDate(dateStr, false)
		time.Sleep(100 * time.Millisecond)
	}
}

// ProcessDate は特定の日付のデータを取得してDBに保存する
func (p *BatchProcessor) ProcessDate(date string, recordFailure bool) {
	records, err := p.Client.FetchHourlyData(date)
	if err != nil {
		log.Printf("Error fetching data for %s: %v", date, err)
		if recordFailure {
			for h := 0; h < 24; h++ {
				p.DB.SaveFailedRecord(date, h, err.Error())
			}
		}
		return
	}

	for _, r := range records {
		if err := p.DB.SaveHourlyData(date, r, recordFailure); err != nil {
			log.Printf("Error saving data for %s %02d:00 : %v", date, r.Hour, err)
			if recordFailure {
				p.DB.SaveFailedRecord(date, r.Hour, err.Error())
			}
		}
	}
	log.Printf("Successfully processed data for %s", date)
}

// RetryFailedDates は過去に失敗したデータを再取得する
func (p *BatchProcessor) RetryFailedDates(excludeDate string) {
	failedDates, err := p.DB.GetFailedDates()
	if err != nil {
		log.Printf("Error getting failed dates: %v", err)
		return
	}

	if len(failedDates) > 0 {
		log.Printf("Retrying failed dates: %v", failedDates)
		for _, fd := range failedDates {
			dateClean := fd[0:4] + fd[5:7] + fd[8:10]
			if dateClean == excludeDate {
				continue
			}
			// リトライ時は失敗を記録（更新）する (true)
			p.ProcessDate(dateClean, true)
		}
	}
}
