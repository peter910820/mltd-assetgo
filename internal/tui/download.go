package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"mltdassetgo/internal/asset"
)

const recentLimit = 8

// 下載畫面所使用的訊息

type versionMsg asset.Version
type manifestMsg []asset.File
type errMsg struct{ err error }

// result 為每個下載完成的檔案，由下載 worker 送出。
type result struct {
	file  asset.File
	bytes int64
	err   error
}

func (m Model) fetchVersion() tea.Cmd {
	return func() tea.Msg {
		v, err := m.client.LatestVersion(m.ctx)
		if err != nil {
			return errMsg{err}
		}
		return versionMsg(v)
	}
}

func (m Model) fetchManifest() tea.Cmd {
	return func() tea.Msg {
		files, err := m.client.Manifest(m.ctx, m.version)
		if err != nil {
			return errMsg{err}
		}
		return manifestMsg(files)
	}
}

// startDownloads 啟動一組 worker pool，並透過 channel 串流回傳結果。
func (m *Model) startDownloads() {
	jobs := make(chan asset.File)
	go func() {
		for _, f := range m.files {
			jobs <- f
		}
		close(jobs)
	}()

	for i := 0; i < m.conc; i++ {
		go func() {
			for f := range jobs {
				n, err := m.client.Download(m.ctx, m.version.Asset.Version, f)
				m.results <- result{file: f, bytes: n, err: err}
			}
		}()
	}
}

func listen(ch chan result) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// updateDownload 推進「版本 -> 資源清單 -> 下載」的狀態機。
func (m Model) updateDownload(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		pm, cmd := m.progress.Update(msg)
		m.progress = pm.(progress.Model)
		return m, cmd

	case versionMsg:
		m.version = asset.Version(msg)
		m.state = stateManifest
		return m, m.fetchManifest()

	case manifestMsg:
		m.files = msg
		if m.limit > 0 && m.limit < len(m.files) {
			m.files = m.files[:m.limit]
		}
		m.total = len(m.files)
		if m.total == 0 {
			m.state = stateDone
			return m, nil
		}
		m.state = stateDownloading
		m.results = make(chan result, m.conc*2)
		m.startDownloads()
		return m, listen(m.results)

	case result:
		if msg.err != nil {
			m.failed++
			m.recent = appendRecent(m.recent, errStyle.Render("✗ ")+faintStyle.Render(msg.file.Name))
		} else {
			m.completed++
			m.bytes += msg.bytes
			m.recent = appendRecent(m.recent, okStyle.Render("✓ ")+valueStyle.Render(msg.file.Name))
		}

		if m.completed+m.failed >= m.total {
			m.state = stateDone
			return m, tea.Sequence(m.progress.SetPercent(1), tea.Quit)
		}
		pct := float64(m.completed+m.failed) / float64(m.total)
		return m, tea.Batch(m.progress.SetPercent(pct), listen(m.results))

	case errMsg:
		m.err = msg.err
		m.state = stateError
		return m, tea.Quit
	}

	return m, nil
}

// viewDownload 繪製版本、資源清單與下載進度等畫面。
func (m Model) viewDownload() string {
	var b strings.Builder

	header := func() {
		fmt.Fprintf(&b, "%s 版本 %s  %s\n",
			labelStyle.Render("版本："),
			valueStyle.Render(fmt.Sprintf("%d", m.version.Asset.Version)),
			faintStyle.Render(m.version.Asset.UpdateAt.In(time.Local).Format("2006-01-02")))
	}

	switch m.state {
	case stateVersion:
		fmt.Fprintf(&b, "%s 正在取得最新版本…\n", m.spinner.View())

	case stateManifest:
		header()
		fmt.Fprintf(&b, "%s 正在下載並解析資源清單…\n", m.spinner.View())

	case stateDownloading, stateDone:
		header()
		// 下載中使用動畫繪製（Harmonica），完成時直接顯示滿格。
		bar := m.progress.View()
		if m.state == stateDone {
			bar = m.progress.ViewAs(1)
		}
		fmt.Fprintf(&b, "%s\n\n", bar)
		fmt.Fprintf(&b, "%s %d/%d   %s %d   %s %s\n",
			labelStyle.Render("完成："), m.completed+m.failed, m.total,
			labelStyle.Render("失敗："), m.failed,
			labelStyle.Render("已下載："), humanBytes(m.bytes))

		if len(m.recent) > 0 {
			fmt.Fprintf(&b, "\n%s\n", faintStyle.Render("最近："))
			for _, line := range m.recent {
				fmt.Fprintf(&b, "  %s\n", line)
			}
		}
		if m.state == stateDone {
			fmt.Fprintf(&b, "\n%s\n", okStyle.Render("全部完成！"))
		}

	case stateError:
		fmt.Fprintf(&b, "%s\n", errStyle.Render("錯誤："+m.err.Error()))
	}

	return b.String()
}

// appendRecent 將一行訊息加入最近完成清單，並保留最後 recentLimit 筆。
func appendRecent(recent []string, line string) []string {
	recent = append(recent, line)
	if len(recent) > recentLimit {
		recent = recent[len(recent)-recentLimit:]
	}
	return recent
}
