# OMRON 太陽光・分電盤計測システム連携バッチ

本プロジェクトは、オムロン製計測ユニット（MCSM-P03）から電力データを自動取得し、MariaDBへ保存・Grafanaで可視化するためのシステムです。

## 1. システム構成

- **言語:** Go 1.25+
- **データベース:** MariaDB 10.11
- **可視化ツール:** Grafana
- **インフラ:** Docker / Docker Compose (Synology NAS 等のコンテナ環境を想定)
- **タイムゾーン:** すべてのコンテナおよびプログラムで `Asia/Tokyo` (JST) を使用

### アーキテクチャ図
```
[オムロン計測ユニット] <--- (TCP/CGI API) --- [Go バッチコンテナ]
                                              |
                                              v
[Grafana (可視化)] <--- (SQL) --- [MariaDB (蓄積)]
```

## 2. 主な機能

- **自動履歴取得**: 毎日 AM 1:00 に起動し、前日分の1時間ごとの電力データ（発電量、消費量、売電量、買電量）を取得します。
- **一括取得（バックフィル）**: 過去（2017/09/01〜）のデータを一括で取り込むことが可能です。
- **自動スキップ & リトライ**: すでに正常取得済みのデータはスキップし、過去に失敗したデータ（`is_failed=TRUE`）のみを自動的に再取得します。
- **データ永続化**: 計測ユニット内のデータ消去に備え、MariaDBに長期保存します。
- **可視化**: Grafana を使用して、時系列グラフや統計表示が可能です。

## 3. ディレクトリ構成

- `batch/`: Go言語によるバッチ処理のソースコード
  - `main.go`: エントリポイント、設定読み込み
  - `processor.go`: メインロジック（バックフィル、リトライ、個別処理）
  - `db.go`: データベース操作（CRUD、JST処理）
  - `omron.go`: 計測ユニットとの通信（非標準HTTP/TCP通信）
  - `models.go`: データ構造体の定義
- `docker/`: 実行環境設定
  - `compose.yml`: マルチコンテナ（DB, Grafana, Batch）の定義
  - `Dockerfile`: バッチ実行用コンテナのビルド設定
  - `init.sql`: データベース初期化スクリプト
  - `.env`: 環境変数設定（認証情報、接続先IP等）

## 4. データベース仕様 (`energy_data` テーブル)

本システムでは、すべてのデータと取得状態を単一のテーブルで管理します。
- `date`, `hour`: 主キー（日付、時間）
- `gen_1`〜`gen_2`: 各パワーコントローラの発電量(Wh)
- `gen_total`: 合計発電量(Wh)
- `consumption`: 消費電力量(Wh)
- `selling`, `buying`: 売電量、買電量(Wh)
- `is_failed`: 取得失敗フラグ（成功時は `FALSE`, 失敗時は `TRUE`）
- `error_message`: 失敗時のエラー内容（成功/復旧時は `recovered`）

## 5. セットアップと実行

### 5.1 環境設定
`docker/.env` ファイルを作成（または編集）し、各設定値を入力してください。
```env
MARIADB_ROOT_PASSWORD=<YOUR_ROOT_PASSWORD>
MARIADB_DATABASE=omron_energy
MARIADB_USER=<YOUR_DB_USER>
MARIADB_PASSWORD=<YOUR_DB_PASSWORD>

OMRON_IP=<YOUR_OMRON_IP>
OMRON_DEVICE_ID=<YOUR_DEVICE_ID>

GRAFANA_ADMIN_PASSWORD=<YOUR_GRAFANA_PASSWORD>
```

### 5.2 起動方法
```bash
cd docker
# DBとGrafanaを常時起動
docker compose up -d

# バッチをビルド
docker compose build batch
```

### 5.3 データの取得
```bash
# 前日分を取得（通常の定期実行用）
docker compose run --rm batch

# 過去データを一括取得（すでに取得済みの日は自動スキップされます）
docker compose run --rm batch /omron_batch -start 20170901
```

## 6. Grafana での可視化

1. `http://(サーバーのIP):3000` にアクセス（初期ID/PW: `admin/(設定したPW)`）。
2. Data Source に `MySQL` を追加。
   - Host: `db:3306`
   - Database: `omron_energy`
   - User/PW: `omron_user/omron_password`
3. ダッシュボードを作成し、以下のSQL等でグラフを描画します。

```sql
SELECT
  time,
  gen_total AS "発電量(Wh)",
  consumption AS "消費量(Wh)",
  buying AS "買電(Wh)",
  selling AS "売電(Wh)"
FROM (
  SELECT
    date + INTERVAL hour HOUR AS time,
    gen_total, consumption, buying, selling
  FROM energy_data
) AS sub
WHERE $__timeFilter(time)
ORDER BY time
```

## 7. 免責事項

- **リバースエンジニアリング**: 本ソフトウェアは、オムロン製計測ユニット（MCSM-P03）の非公開APIおよび通信プロトコルをリバースエンジニアリングして開発されたものです。
- **無保証**: 本ソフトウェアの使用により生じたいかなる損害（データの消失、機器の故障、電気料金の計算不一致等）について、開発者は一切の責任を負いません。
- **メーカー非公式**: 本プロジェクトはオムロン株式会社とは一切関係のない個人の開発物です。メーカーのアップデートにより動作しなくなる可能性があります。

## 8. 保守・特殊仕様

- **通信プロトコル**: オムロンのCGI APIは標準的なHTTPヘッダーを返さないため、Goの `net/http` ではなく `net.Dial` による生のTCP通信でデータを取得しています。
- **リトライロジック**: バッチ実行時に `is_failed=TRUE` のレコードを自動検出し、再取得を試みます。
