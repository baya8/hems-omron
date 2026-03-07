package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type DB struct {
	conn *sql.DB
}

func ConnectDB(dsn string) (*DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	
	// 接続確認（DBが立ち上がるまで少し待機するリトライを入れるとより堅牢）
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
func (db *DB) SaveHourlyData(date string, record HourlyRecord) error {
	now := time.Now()
	
	// すでにデータがあるか確認（リトライかどうかを判定）
	var exists bool
	err := db.conn.QueryRow("SELECT EXISTS(SELECT 1 FROM energy_data WHERE date = ? AND hour = ?)", date, record.Hour).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		// リトライ成功として更新
		query := `
			UPDATE energy_data SET
				gen_1 = ?, gen_2 = ?, gen_total = ?, consumption = ?, selling = ?, buying = ?,
				retried_at = ?
			WHERE date = ? AND hour = ?`
		_, err = db.conn.Exec(query, record.Gen1, record.Gen2, record.GenTotal, record.Consumption, record.Selling, record.Buying, now, date, record.Hour)
	} else {
		// 新規保存
		query := `
			INSERT INTO energy_data 
				(date, hour, gen_1, gen_2, gen_total, consumption, selling, buying, fetched_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
		_, err = db.conn.Exec(query, date, record.Hour, record.Gen1, record.Gen2, record.GenTotal, record.Consumption, record.Selling, record.Buying, now)
	}

	if err != nil {
		return err
	}

	// 取得ステータスを更新
	return db.UpdateFetchStatus(date, record.Hour, false, "")
}

// UpdateFetchStatus は取得ステータスを更新する
func (db *DB) UpdateFetchStatus(date string, hour int, isFailed bool, errMsg string) error {
	query := `
		INSERT INTO fetch_status (date, hour, is_failed, error_message, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE is_failed = VALUES(is_failed), error_message = VALUES(error_message), updated_at = VALUES(updated_at)`
	_, err := db.conn.Exec(query, date, hour, isFailed, errMsg, time.Now())
	return err
}

// GetFailedDates は失敗している日付の一覧を取得する（リトライ用）
func (db *DB) GetFailedDates() ([]string, error) {
	rows, err := db.conn.Query("SELECT DISTINCT date FROM fetch_status WHERE is_failed = TRUE")
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
