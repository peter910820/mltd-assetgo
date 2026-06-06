// Package asset 實作 MLTD（偶像大師 百萬人演唱會！劇場時光）的資源下載邏輯：
// 取得最新版本、解析 msgpack 資源清單，以及下載個別資源檔。
package asset

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

const (
	// versionAPI 會回傳所有已知資源版本的清單。
	versionAPI = "https://api.matsurihi.me/api/mltd/v2/version/latest"
	// assetBaseURL 是 Android 正式資源的 CDN 根網址。
	assetBaseURL = "https://td-assets.bn765.com"
)

// Version 描述版本 API 回傳的單一版本項目。
type Version struct {
	App   VersionApp   `json:"app"`
	Asset VersionAsset `json:"asset"`
}

type VersionApp struct {
	Version  string    `json:"version"`
	UpdateAt time.Time `json:"updatedAt"`
	Revision int64     `json:"revision"`
}

type VersionAsset struct {
	Version   int64     `json:"version"`
	UpdateAt  time.Time `json:"updatedAt"`
	IndexName string    `json:"indexName"`
}

// File 是從資源清單中取出的單一可下載資源。
type File struct {
	// Name 為人類可讀的資源名稱（資源清單的鍵）。
	Name string
	// Route 為用來組合下載網址的雜湊檔名。
	Route string
}

// Client 負責從官方 CDN 下載 MLTD 資源。
type Client struct {
	HTTP    *http.Client
	OutDir  string
	BaseURL string
}

// NewClient 回傳一個帶有合理預設值的 Client。
func NewClient(outDir string) *Client {
	return &Client{
		HTTP: &http.Client{
			Timeout: 5 * time.Minute,
		},
		OutDir:  outDir,
		BaseURL: assetBaseURL,
	}
}

// get 發送 GET 請求，狀態碼為 2xx 時回傳 response。
// 呼叫端必須負責關閉 response body。
func (c *Client) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s：狀態碼 %d", url, resp.StatusCode)
	}
	return resp, nil
}

// assetURL 依版本與檔案路由組合出 CDN 下載網址。
func (c *Client) assetURL(version int64, route string) string {
	return fmt.Sprintf("%s/%d/production/2018/Android/%s", c.BaseURL, version, route)
}

// LatestVersion 取得版本清單並回傳最新一筆（陣列最後一個元素）。
func (c *Client) LatestVersion(ctx context.Context) (Version, error) {
	resp, err := c.get(ctx, versionAPI)
	if err != nil {
		return Version{}, err
	}
	defer resp.Body.Close()

	var versions Version
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return Version{}, fmt.Errorf("解析版本清單失敗：%w", err)
	}

	return versions, nil
}

// Manifest 下載並解析指定版本的 msgpack 資源清單，
// 回傳清單第一個區段中所包含的資源檔列表。
func (c *Client) Manifest(ctx context.Context, v Version) ([]File, error) {
	resp, err := c.get(ctx, c.assetURL(v.Asset.Version, v.Asset.IndexName))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("讀取資源清單失敗：%w", err)
	}
	return parseManifest(body)
}

// parseManifest 解析 msgpack 資源清單內容。清單是一個陣列，
// 第一個元素為「資源名稱 -> 元組」的對應表，元組索引 1 即為雜湊下載路由。
func parseManifest(body []byte) ([]File, error) {
	var root []any
	if err := msgpack.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("解包資源清單失敗：%w", err)
	}
	if len(root) == 0 {
		return nil, fmt.Errorf("資源清單沒有任何區段")
	}

	section, ok := root[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("資源清單格式不符：第一個區段為 %T", root[0])
	}

	files := make([]File, 0, len(section))
	for name, raw := range section {
		tuple, ok := raw.([]any)
		if !ok || len(tuple) < 2 {
			continue
		}
		route, ok := tuple[1].(string)
		if !ok {
			continue
		}
		files = append(files, File{Name: name, Route: route})
	}

	return files, nil
}

// Download 下載單一資源檔並寫入輸出資料夾，回傳寫入的位元組數。
func (c *Client) Download(ctx context.Context, version int64, f File) (int64, error) {
	if err := os.MkdirAll(c.OutDir, 0o755); err != nil {
		return 0, fmt.Errorf("建立輸出資料夾失敗：%w", err)
	}

	resp, err := c.get(ctx, c.assetURL(version, f.Route))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath.Join(c.OutDir, f.Route))
	if err != nil {
		return 0, err
	}
	defer out.Close()

	return io.Copy(out, resp.Body)
}
