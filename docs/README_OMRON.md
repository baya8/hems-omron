# OMRON 太陽光発電・分電盤計測システム 連携ガイド

オムロン製計測ユニット（MCSM-P03）から、リアルタイム電力および過去の履歴データを取得するための仕様書です。

## 1. システム構成
- **デバイスIP:** `192.168.50.25`
- **デバイスID:** `00168978` (HTTPリクエストに必要)
- **ECHONET Liteポート:** `3610` (UDP)
- **Web APIポート:** `80` (TCP/HTTP)

## 2. データ取得仕様

### リアルタイム監視 (ECHONET Lite)
家全体の瞬時電力は、標準プロパティではなく独自回路プロパティから取得します。

| クラス | インスタンス | EPC | 内容 | 仕様 |
| :--- | :--- | :--- | :--- | :--- |
| `0x0279` (太陽光) | `0x01` | `0xE0` | **発電電力** | 2バイト (W) |
| `0x0287` (分電盤) | `0x01` | `0xC6` | **主幹電力** | 後半2バイト (W) ※買電は正、売電は負 |
| `0x0287` (分電盤) | `0x01` | `0xC8` | **電圧** | 前半L1 / 後半L2 (0.1V単位) |

**計算式:** `消費電力 ＝ 発電電力 ＋ 主幹電力`

### 過去履歴の取得 (HTTP Web API)
履歴データ（日別合計値）は、ECHONET LiteではなくWebサーバー(CGI)から取得します。

| パラメータ | 内容 | 備考 |
| :--- | :--- | :--- |
| **`IG0`** | **発電量** | 1日の総発電量 (Wh) |
| **`ISI`** | **売電量** | 1日の総売電量 (Wh) |
| **`IBI`** | **買電量** | 1日の総買電量 (Wh) |

**URL形式:** `http://[IP]/getinfo.cgi?[ID]&0&[開始日]&[終了日]&IG0&ISI&IBI`

---

## 3. ツール・プログラム

### メインツール: `omron_energy_tool.go`
これ1つでリアルタイム監視とCSV保存が可能です。

- **リアルタイム監視モード:**
  ```bash
  go run omron_energy_tool.go monitor
  ```
- **CSVエクスポートモード (例: 2026年3月分):**
  ```bash
  go run omron_energy_tool.go export 202603
  ```

### 補助ツール
- `discover.go`: ネットワーク内のECHONET Lite機器を再検索する場合に使用。
- `check_instances.go`: デバイスの機能一覧を確認する場合に使用。
- `get_properties.go`: 特定の機能がサポートする項目(EPC)を調べる場合に使用。

---

## 4. WSL2環境での実行に関する注意
WindowsファイアウォールがUDP通信をブロックすることがあります。通信がタイムアウトする場合は、管理者権限のPowerShellで以下のコマンドを実行してください：
```powershell
New-NetFirewallRule -DisplayName "WSL2 ECHONET Lite" -Direction Inbound -Action Allow -Protocol UDP -LocalPort 3610
```
