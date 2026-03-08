# OMRON 太陽光・分電盤計測システム連携バッチ

本プロジェクトは、オムロン製計測ユニット（MCSM-P03）から電力データを自動取得し、MariaDBへ保存・Grafanaで可視化するためのシステムです。

## 1. システム構成

- **言語:** Go 1.21+
- **データベース:** MariaDB 10.11
- **可視化ツール:** Grafana
- **インフラ:** Docker / Docker Compose (Synology NAS 等のコンテナ環境を想定)

### アーキテクチャ図
```
[オムロン計測ユニット] <--- (TCP/CGI API) --- [Go バッチコンテナ]
                                              |
                                              v
[Grafana (可視化)] <--- (SQL) --- [MariaDB (蓄積)]
```

## 2. 主な機能

- **自動履歴取得**: 毎日 AM 1:00 に起動し、前日分の1時間ごとの電力データ（発電量、消費量、売電量、買電量）を取得します。
- **自動リトライ**: ネットワークエラー等で取得に失敗した場合、次回のバッチ実行時に未取得分を自動的に再取得します。
- **データ永続化**: 計測ユニット内のデータ消去に備え、MariaDBに長期保存します。
- **可視化**: Grafana を使用して、日別・月別のグラフ表示が可能です。

## 3. ディレクトリ構成

- `batch/`: Go言語によるバッチ処理のソースコード
  - `main.go`: エントリポイント、リトライロジック
  - `omron.go`: 計測ユニットとの通信（非標準HTTP/TCP通信）
  - `db.go`: データベース操作
- `docker/`: 実行環境設定
  - `compose.yml`: マルチコンテナ（DB, Grafana, Batch）の定義
  - `Dockerfile`: バッチ実行用コンテナのビルド設定
  - `init.sql`: データベース初期化スクリプト

## 4. データベース仕様

### energy_data テーブル
電力の生データを1時間単位で保存します。
- `date`, `hour`: 主キー（日付、時間）
- `gen_1`〜`gen_2`: 各パワーコントローラの発電量(Wh)
- `gen_total`: 合計発電量(Wh)
- `consumption`: 消費電力量(Wh)
- `selling`, `buying`: 売電量、買電量(Wh)

### fetch_status テーブル
取得の成否を管理し、リトライ対象を判定します。

## 5. セットアップと実行

### 環境変数の設定
`docker/compose.yml` 内の以下の値を環境に合わせて変更してください。
- `OMRON_IP`: 計測ユニットのIPアドレス（デフォルト: `192.168.50.25`）
- `OMRON_DEVICE_ID`: デバイスID（デフォルト: `00168978`）

### 起動方法
```bash
cd docker
# DBとGrafanaを常時起動
docker-compose up -d db grafana

# バッチを手動実行（前日分を取得）
docker-compose build batch
docker-compose run --rm batch
```

### 定期実行（Synology NAS）
Synology NAS の「タスクスケジューラ」にて、以下のユーザー指定スクリプトを毎日 AM 1:00 に実行するように設定してください。
```bash
cd /volume1/docker/hems-omron/docker
/usr/local/bin/docker-compose run --rm batch
```

## 6. Grafana での可視化

1. `http://(NASのIP):3000` にアクセス（初期ID/PW: `admin/admin`）。
2. Data Source に `MySQL` を追加。
   - Host: `db:3306`
   - Database: `omron_energy`
   - User/PW: `omron_user/omron_password`
3. ダッシュボードを作成し、SQLでグラフを描画します。

## 7. 開発・保守

- **手動リトライ**: 特定の日付を再取得したい場合は、以下のコマンドを実行してください。
  ```bash
  docker-compose run --rm batch /omron_batch -date 20260301
  ```
- **特殊仕様**: オムロンのCGI APIは標準的なHTTPヘッダーを返さないため、Goの `net/http` ではなく `net.Dial` による生のTCP通信でデータを取得しています。
