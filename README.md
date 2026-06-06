# mltd-assetgo

使用 Go 開發的 MLTD（偶像大師 百萬人演唱會！劇場時光 / 日版）資源下載器，
並使用 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 打造終端機 UI 與即時下載進度。

## 功能

1. 根據 `Princess API(matsurihime)` 取得最新版本與 `indexName`
2. 下載並以 msgpack 解析資源清單（manifest），取出每個檔案的下載路由
3. 將所有資源下載到輸出資料夾

## 需求

- Go 1.26+

## 使用方式

```bash
go run .
```

或先編譯：

```bash
go build -o mltd-assetgo .
./mltd-assetgo
```

啟動後會顯示設定表單，請輸入以下項目後按 `Enter` 開始下載：

| 欄位 | 留空時的建議值 | 說明 |
| --- | --- | --- |
| 下載資料夾 | `file` | 可用相對路徑，例如 `./assets` |
| 下載數量上限 | `0` | `0` 代表全部 |
| 並行下載數 | `8` | 建議 8，範圍 1-64 |

可用 `tab` / `↑` / `↓` 切換欄位。下載過程中按 `q` 或 `ctrl+c` 離開。

## 專案結構

```
main.go                  進入點
internal/asset/asset.go  版本查詢、manifest 解析、下載邏輯
internal/tui/            Bubble Tea 終端機 UI
```

## 免責聲明

- **本專案為非官方、非營利的第三方開源工具，與 BANDAI NAMCO Entertainment、THE IDOLM@STER 系列或其相關權利人沒有任何關聯，亦未獲得授權或背書。**
- **使用本工具下載之遊戲資源，僅供個人研究、學習或除錯等合理使用目的。使用者必須於下載完成後 24 小時內，將所有相關檔案自本機完全刪除，不得長期保存、散布、公開分享或作為商業用途。**

