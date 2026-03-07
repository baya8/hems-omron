package main

import (
	"database/sql"
	"time"
)

// EnergyData は energy_data テーブルのレコード
type EnergyData struct {
	Date         string         `db:"date"`
	Hour         int            `db:"hour"`
	Gen1         sql.NullInt64  `db:"gen_1"`
	Gen2         sql.NullInt64  `db:"gen_2"`
	Gen3         sql.NullInt64  `db:"gen_3"`
	Gen4         sql.NullInt64  `db:"gen_4"`
	Gen5         sql.NullInt64  `db:"gen_5"`
	GenTotal     int            `db:"gen_total"`
	Consumption  int            `db:"consumption"`
	Selling      int            `db:"selling"`
	Buying       int            `db:"buying"`
	FetchedAt    time.Time      `db:"fetched_at"`
	RetriedAt    sql.NullTime   `db:"retried_at"`
	CorrectedAt  sql.NullTime   `db:"corrected_at"`
}

// FetchStatus は fetch_status テーブルのレコード
type FetchStatus struct {
	Date         string         `db:"date"`
	Hour         int            `db:"hour"`
	IsFailed     bool           `db:"is_failed"`
	ErrorMessage sql.NullString `db:"error_message"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

// HourlyRecord はオムロンAPIから取得した1時間分のデータ
type HourlyRecord struct {
	Hour        int
	Gen1        int
	Gen2        int
	GenTotal    int
	Selling     int
	Buying      int
	Consumption int
}
