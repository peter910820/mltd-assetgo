package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// 啟動表單的欄位索引。
const (
	fieldPath = iota
	fieldLimit
	fieldConc
)

// updatePath 處理啟動設定表單。可用 tab／方向鍵在欄位間切換，
// 按 enter 確認所有設定並啟動第一個網路請求；其餘輸入交給目前聚焦的欄位。
func (m Model) updatePath(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyEnter:
			return m.applySetup()
		case tea.KeyTab, tea.KeyDown:
			m.focus = (m.focus + 1) % len(m.inputs)
			return m, m.refocus()
		case tea.KeyShiftTab, tea.KeyUp:
			m.focus = (m.focus - 1 + len(m.inputs)) % len(m.inputs)
			return m, m.refocus()
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	return m, cmd
}

// refocus 將焦點切到目前欄位（其餘失焦），並回傳游標閃爍指令。
func (m *Model) refocus() tea.Cmd {
	for i := range m.inputs {
		if i == m.focus {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	return textinput.Blink
}

// applySetup 讀取表單欄位、套用設定（並對並行數做範圍限制），接著進入下載流程。
// 欄位留空時採用 placeholder 所示的建議值。
func (m Model) applySetup() (tea.Model, tea.Cmd) {
	path := strings.TrimSpace(m.inputs[fieldPath].Value())
	if path == "" {
		path = "file"
	}
	m.client.OutDir = path

	limitStr := strings.TrimSpace(m.inputs[fieldLimit].Value())
	if limitStr == "" {
		m.limit = 0
	} else if v, err := strconv.Atoi(limitStr); err == nil && v >= 0 {
		m.limit = v
	}

	concStr := strings.TrimSpace(m.inputs[fieldConc].Value())
	conc := 8
	if concStr != "" {
		if v, err := strconv.Atoi(concStr); err == nil && v >= 1 {
			conc = v
		}
	}
	if conc > maxConc {
		conc = maxConc
	}
	m.conc = conc

	m.state = stateVersion
	return m, tea.Batch(m.spinner.Tick, m.fetchVersion())
}

// viewPath 繪製啟動設定表單。
func (m Model) viewPath() string {
	var b strings.Builder

	field := func(label, hint string, in textinput.Model) {
		fmt.Fprintf(&b, "%s  %s\n%s\n\n",
			labelStyle.Render(label), faintStyle.Render(hint), in.View())
	}

	field("下載資料夾", "留空則使用 file，可用相對路徑", m.inputs[fieldPath])
	field("下載數量上限", "留空則 0（全部）", m.inputs[fieldLimit])
	field("並行下載數", fmt.Sprintf("留空則 8（範圍 1-%d）", maxConc), m.inputs[fieldConc])

	return b.String()
}
