package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

type DB struct {
	conn *sql.DB
}

func ConnectDB(dsn string) (*DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	
	for i := 0; i < 5; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		log.Printf("DB connection failed, retrying... (%d/5)", i+1)
		time.Sleep(5 * time.Second)
	}
	
	if err != nil {
		return nil, fmt.Errorf("could not connect to DB: %v", err)
	}

	return &DB{conn: db}, nil
}

func (db *DB) Close() {
	db.conn.Close()
}

// SaveHourlyData は 1時間分のデータを保存または更新する
func (db *DB) SaveHourlyData(date string, record HourlyRecord, recordFailure bool) error {
	now := time.Now().In(jst)
	
	var exists bool
	err := db.conn.QueryRow("SELECT EXISTS(SELECT 1 FROM energy_data WHERE date = ? AND hour = ?)", date, record.Hour).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		query := `
			UPDATE energy_data SET
				gen_1 = ?, gen_2 = ?, gen_total = ?, consumption = ?, selling = ?, buying = ?,
				is_failed = FALSE, error_message = 'recovered', retried_at = ?
			WHERE date = ? AND hour = ?`
		_, err = db.conn.Exec(query, record.Gen1, record.Gen2, record.GenTotal, record.Consumption, record.Selling, record.Buying, now, date, record.Hour)
	} else {
		query := `
			INSERT INTO energy_data 
				(date, hour, gen_1, gen_2, gen_total, consumption, selling, buying, is_failed, error_message, fetched_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, FALSE, NULL, ?)`
		_, err = db.conn.Exec(query, date, record.Hour, record.Gen1, record.Gen2, record.GenTotal, record.Consumption, record.Selling, record.Buying, now)
	}
	return err
}

// SaveFailedRecord は取得に失敗した際に、値を 0 として保存する
func (db *DB) SaveFailedRecord(date string, hour int, errMsg string) error {
	now := time.Now().In(jst)
	
	// 失敗時は各値を 0 にして保存、すでにレコードがある場合は is_failed を TRUE にしてエラーを記録
	query := `
		INSERT INTO energy_data 
			(date, hour, gen_1, gen_2, gen_total, consumption, selling, buying, is_failed, error_message, fetched_at)
		VALUES (?, ?, 0, 0, 0, 0, 0, 0, TRUE, ?, ?)
		ON DUPLICATE KEY UPDATE is_failed = TRUE, error_message = VALUES(error_message), fetched_at = VALUES(fetched_at)`
	_, err := db.conn.Exec(query, date, hour, errMsg, now)
	return err
}

// GetFailedDates は取得に失敗している（is_failed=true）日付の一覧を取得する
func (db *DB) GetFailedDates() ([]string, error) {
	rows, err := db.conn.Query("SELECT DISTINCT date FROM energy_data WHERE is_failed = TRUE")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			return nil, err
		}
		dates = append(dates, date)
	}
	return dates, nil
}

// IsDateFetched は、その日の全データ（24時間分）が正常に取得できているか確認する
func (db *DB) IsDateFetched(date string) (bool, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM energy_data WHERE date = ? AND is_failed = FALSE", date).Scan(&count)
	if err != nil {
		return false, err
	}
	return count >= 24, nil
}
