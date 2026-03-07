CREATE DATABASE IF NOT EXISTS omron_energy;
USE omron_energy;

-- 太陽光発電送信ユニットから取得した生データを保存する
CREATE TABLE IF NOT EXISTS energy_data (
    date DATE NOT NULL,
    hour TINYINT NOT NULL,
    gen_1 INT DEFAULT NULL COMMENT 'パワーコントローラ1発電量',
    gen_2 INT DEFAULT NULL COMMENT 'パワーコントローラ2発電量',
    gen_3 INT DEFAULT NULL COMMENT '将来用3',
    gen_4 INT DEFAULT NULL COMMENT '将来用4',
    gen_5 INT DEFAULT NULL COMMENT '将来用5',
    gen_total INT NOT NULL COMMENT '1-5の合計発電量',
    consumption INT NOT NULL COMMENT '消費電力量',
    selling INT NOT NULL COMMENT '売電量',
    buying INT NOT NULL COMMENT '買電量',
    fetched_at DATETIME NOT NULL COMMENT 'APIから取得しDBへ保存した日時',
    retried_at DATETIME DEFAULT NULL COMMENT 'リトライ等で成功した再取得日時',
    corrected_at DATETIME DEFAULT NULL COMMENT '手動補正日時',
    PRIMARY KEY (date, hour)
);

-- データ取得の状態を管理する（リトライ対象の判定用）
CREATE TABLE IF NOT EXISTS fetch_status (
    date DATE NOT NULL,
    hour TINYINT NOT NULL,
    is_failed BOOLEAN NOT NULL DEFAULT FALSE,
    error_message TEXT,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (date, hour)
);
