// Package tui 以 Bubble Tea 終端機介面呈現資源下載器。
//
// 介面分成兩個畫面，各自實作於獨立檔案：
//   - path.go     ：選擇下載資料夾（statePath）
//   - download.go ：取得版本／資源清單並下載資源（其餘狀態）
//
// 本檔案存放共用的 model、樣式，以及將工作分派給對應畫面的
// 最上層 Update／View 分派器。
package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"mltdassetgo/internal/asset"
)

type state int

const (
	statePath state = iota
	stateVersion
	stateManifest
	stateDownloading
	stateDone
	stateError
)

// maxConc 為並行下載數的上限，也用於啟動表單的建議文字。
const maxConc = 64

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	faintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// Model 是驅動下載器介面的 Bubble Tea model。
type Model struct {
	ctx    context.Context
	client *asset.Client
	limit  int
	conc   int

	state    state
	spinner  spinner.Model
	progress progress.Model
	inputs   []textinput.Model
	focus    int

	version asset.Version
	files   []asset.File

	total     int
	completed int
	failed    int
	bytes     int64
	recent    []string

	results chan result
	err     error
}

// New 建立一個可直接交給 Bubble Tea 程式執行的 Model。
// 所有下載設定皆由啟動表單的使用者輸入決定。
func New() Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	pr := progress.New(progress.WithDefaultGradient())

	mkInput := func(placeholder string, width int) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = 256
		ti.Width = width
		return ti
	}

	inputs := []textinput.Model{
		mkInput("file", 40),
		mkInput("0", 12),
		mkInput("8", 12),
	}
	inputs[fieldPath].Focus()

	return Model{
		ctx:      context.Background(),
		client:   asset.NewClient("file"),
		state:    statePath,
		spinner:  sp,
		progress: pr,
		inputs:   inputs,
	}
}

// Init 讓路徑輸入框的游標開始閃爍；要等使用者確認下載資料夾後，
// 才會開始進行網路請求。
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update 先處理全域訊息，再依目前畫面分派後續處理。
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.progress.Width = min(msg.Width-4, 60)
		return m, nil
	}

	if m.state == statePath {
		return m.updatePath(msg)
	}
	return m.updateDownload(msg)
}

// View 繪製標題、目前畫面內容，以及依狀態變化的操作提示列。
func (m Model) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", titleStyle.Render("MLTD 資源下載器（Go / Bubble Tea）"))

	if m.state == statePath {
		b.WriteString(m.viewPath())
	} else {
		b.WriteString(m.viewDownload())
	}

	help := "按 q 或 ctrl+c 離開"
	if m.state == statePath {
		help = "tab：切換欄位 • enter：開始下載 • ctrl+c：離開"
	}
	fmt.Fprintf(&b, "\n%s", helpStyle.Render(help))
	return b.String()
}

// humanBytes 將位元組數轉為易讀的容量字串。
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
