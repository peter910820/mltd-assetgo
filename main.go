// 指令 mltd-assetgo 用於下載 MLTD（偶像大師 百萬人演唱會！劇場時光）的遊戲資源，
// 並以 Bubble Tea 終端機介面即時顯示下載進度。
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"mltdassetgo/internal/tui"
)

func main() {
	if _, err := tea.NewProgram(tui.New()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "錯誤：", err)
		os.Exit(1)
	}
}
