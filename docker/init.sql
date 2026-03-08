CREATE DATABASE IF NOT EXISTS omron_energy;
USE omron_energy;

-- 太陽光発電送信ユニットから取得した生データを保存する
CREATE TABLE IF NOT EXISTS energy_data (
    date DATE NOT NULL,
    hour TINYINT NOT NULL,
    gen_1 INT NOT NULL DEFAULT 0 COMMENT 'パワーコントローラ1発電量',
    gen_2 INT NOT NULL DEFAULT 0 COMMENT 'パワーコントローラ2発電量',
    gen_3 INT NOT NULL DEFAULT 0 COMMENT '将来用3',
    gen_4 INT NOT NULL DEFAULT 0 COMMENT '将来用4',
    gen_5 INT NOT NULL DEFAULT 0 COMMENT '将来用5',
    gen_total INT NOT NULL DEFAULT 0 COMMENT '1-5の合計発電量',
    consumption INT NOT NULL DEFAULT 0 COMMENT '消費電力量',
    selling INT NOT NULL DEFAULT 0 COMMENT '売電量',
    buying INT NOT NULL DEFAULT 0 COMMENT '買電量',
    is_failed BOOLEAN NOT NULL DEFAULT FALSE COMMENT '取得失敗フラグ',
    error_message TEXT DEFAULT NULL COMMENT 'エラー内容',
    fetched_at DATETIME NOT NULL COMMENT '保存日時',
    retried_at DATETIME DEFAULT NULL COMMENT '再取得成功日時',
    corrected_at DATETIME DEFAULT NULL COMMENT '手動補正日時',
    PRIMARY KEY (date, hour)
);
