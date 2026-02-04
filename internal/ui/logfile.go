package ui

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func (a *App) openLogFileCommand() {
	path := a.resolveLogPath()
	if path == "" {
		a.setStatusMessage("Log file not found")
		return
	}
	if _, err := os.Stat(path); err != nil {
		a.setStatusMessage("Log file not found")
		return
	}
	if err := openWithSystem(path); err == nil {
		a.setStatusMessage("Opened log file")
		return
	}
	a.showLogFileDialog(path)
}

func openWithSystem(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", "", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

func (a *App) resolveLogPath() string {
	if path := strings.TrimSpace(os.Getenv("TXP_LOG_PATH")); path != "" {
		return path
	}
	if a.ConfigPath != "" {
		return filepath.Join(filepath.Dir(a.ConfigPath), "txp.log")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".txp", "txp.log")
}

func (a *App) showLogFileDialog(path string) {
	if a.LogFileDialog == nil {
		return
	}
	a.LogFileDialog.Path = path
	a.LogFileDialog.Active = true
	a.LogFileDialog.HasNewLines = false
	a.LogFileDialog.StopCh = make(chan struct{})
	lines, start, end, total, offset := readTailLines(path, 500)
	a.LogFileDialog.Lines = lines
	a.LogFileDialog.StartLine = start
	a.LogFileDialog.EndLine = end
	a.LogFileDialog.TotalLines = total
	a.LogFileDialog.LastSize = logFileSize(path)
	a.LogFileDialog.LastOffset = offset
	a.LogFileDialog.TailBuffer = ""
	a.LogFileDialog.View.SetText(strings.Join(lines, "\n"))
	a.LogFileDialog.View.ScrollToEnd()
	a.updateLogFooter()
	a.pushOverlay("logfile", nil, false, nil)
	a.setPanelStyle(a.LogFileDialog.Root, "Log File", true)
	a.App.SetFocus(a.LogFileDialog.View)
	go a.watchLogFileUpdates()
}

func (a *App) hideLogFileDialog() {
	if a.LogFileDialog == nil {
		return
	}
	a.LogFileDialog.Active = false
	if a.LogFileDialog.StopCh != nil {
		close(a.LogFileDialog.StopCh)
		a.LogFileDialog.StopCh = nil
	}
	if entry, ok := a.popOverlay("logfile"); ok {
		a.restoreOverlayFocus(entry)
	}
	a.setPanelStyle(a.LogFileDialog.Root, "Log File", false)
	a.updateBottomHints()
}

func (a *App) watchLogFileUpdates() {
	if a.LogFileDialog == nil || a.LogFileDialog.StopCh == nil {
		return
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-a.LogFileDialog.StopCh:
			return
		case <-ticker.C:
			size := logFileSize(a.LogFileDialog.Path)
			if size < a.LogFileDialog.LastOffset {
				lines, start, end, total, offset := readTailLines(a.LogFileDialog.Path, 500)
				a.LogFileDialog.Lines = lines
				a.LogFileDialog.StartLine = start
				a.LogFileDialog.EndLine = end
				a.LogFileDialog.TotalLines = total
				a.LogFileDialog.LastOffset = offset
				a.LogFileDialog.TailBuffer = ""
				a.LogFileDialog.LastSize = size
				a.LogFileDialog.HasNewLines = false
				a.App.QueueUpdateDraw(func() {
					a.LogFileDialog.View.SetText(strings.Join(lines, "\n"))
					a.LogFileDialog.View.ScrollToEnd()
					a.updateLogFooter()
				})
				continue
			}
			if size > a.LogFileDialog.LastSize {
				a.LogFileDialog.LastSize = size
				a.LogFileDialog.HasNewLines = true
				a.App.QueueUpdateDraw(func() {
					a.updateLogFooter()
				})
			}
		}
	}
}

func (a *App) updateLogFooter() {
	if a.LogFileDialog == nil {
		return
	}
	if a.LogFileDialog.HasNewLines {
		a.LogFileDialog.Footer.SetText("New log lines available (F5)")
		return
	}
	a.LogFileDialog.Footer.SetText("")
}

func (a *App) logViewAtBottom() bool {
	if a.LogFileDialog == nil {
		return false
	}
	row, _ := a.LogFileDialog.View.GetScrollOffset()
	_, _, _, height := a.LogFileDialog.View.GetInnerRect()
	if height <= 0 {
		height = 1
	}
	return row+height >= len(a.LogFileDialog.Lines)
}

func (a *App) appendLogLines() {
	if a.LogFileDialog == nil || a.LogFileDialog.Path == "" {
		return
	}
	lines, tail, offset := readAppendedLines(a.LogFileDialog.Path, a.LogFileDialog.LastOffset, a.LogFileDialog.TailBuffer)
	if len(lines) == 0 {
		return
	}
	end := a.LogFileDialog.EndLine + len(lines)
	if a.LogFileDialog.EndLine < 0 {
		end = len(lines) - 1
	}
	a.LogFileDialog.Lines = append(a.LogFileDialog.Lines, lines...)
	a.LogFileDialog.EndLine = end
	a.LogFileDialog.TotalLines += len(lines)
	a.LogFileDialog.LastOffset = offset
	a.LogFileDialog.TailBuffer = tail
	a.LogFileDialog.LastSize = logFileSize(a.LogFileDialog.Path)
	a.LogFileDialog.HasNewLines = false
	a.LogFileDialog.View.SetText(strings.Join(a.LogFileDialog.Lines, "\n"))
	a.updateLogFooter()
}

func (a *App) prependLogLines() {
	if a.LogFileDialog == nil || a.LogFileDialog.Path == "" {
		return
	}
	start := a.LogFileDialog.StartLine - 500
	if start < 0 {
		start = 0
	}
	limit := a.LogFileDialog.StartLine - start
	if limit <= 0 {
		return
	}
	lines, newStart, _, total := readLogRange(a.LogFileDialog.Path, start, limit)
	if len(lines) == 0 {
		return
	}
	currentRow, _ := a.LogFileDialog.View.GetScrollOffset()
	a.LogFileDialog.Lines = append(lines, a.LogFileDialog.Lines...)
	a.LogFileDialog.StartLine = newStart
	a.LogFileDialog.TotalLines = total
	a.LogFileDialog.LastSize = logFileSize(a.LogFileDialog.Path)
	a.LogFileDialog.View.SetText(strings.Join(a.LogFileDialog.Lines, "\n"))
	a.LogFileDialog.View.ScrollTo(currentRow+len(lines), 0)
}

func readTailLines(path string, limit int) ([]string, int, int, int, int64) {
	file, err := os.Open(path)
	if err != nil {
		return []string{}, 0, -1, 0, 0
	}
	defer func() {
		_ = file.Close()
	}()
	info, err := file.Stat()
	if err != nil {
		return []string{}, 0, -1, 0, 0
	}
	reader := bufio.NewReader(file)
	lines := []string{}
	total := 0
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimRight(line, "\n")
			line = strings.TrimRight(line, "\r")
			total++
			if limit > 0 && len(lines) >= limit {
				lines = append(lines[1:], line)
			} else {
				lines = append(lines, line)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}
	if total == 0 {
		return []string{}, 0, -1, 0, info.Size()
	}
	start := total - len(lines)
	if start < 0 {
		start = 0
	}
	end := total - 1
	return lines, start, end, total, info.Size()
}

func readAppendedLines(path string, offset int64, tail string) ([]string, string, int64) {
	file, err := os.Open(path)
	if err != nil {
		return nil, tail, offset
	}
	defer func() {
		_ = file.Close()
	}()
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, tail, offset
	}
	data, err := io.ReadAll(file)
	if err != nil || len(data) == 0 {
		return nil, tail, offset
	}
	text := tail + string(data)
	lines, nextTail := splitLogLines(text)
	return lines, nextTail, offset + int64(len(data))
}

func splitLogLines(text string) ([]string, string) {
	if text == "" {
		return nil, ""
	}
	parts := strings.Split(text, "\n")
	if strings.HasSuffix(text, "\n") {
		return trimLogLines(parts[:len(parts)-1]), ""
	}
	if len(parts) == 1 {
		return nil, parts[0]
	}
	return trimLogLines(parts[:len(parts)-1]), parts[len(parts)-1]
}

func trimLogLines(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		cleaned = append(cleaned, strings.TrimRight(line, "\r"))
	}
	return cleaned
}

func readLogRange(path string, start int, limit int) ([]string, int, int, int) {
	file, err := os.Open(path)
	if err != nil {
		return nil, start, start - 1, 0
	}
	defer func() {
		_ = file.Close()
	}()

	result := []string{}
	reader := bufio.NewReader(file)
	index := 0
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimRight(line, "\n")
			if limit == -1 || (index >= start && index < start+limit) {
				result = append(result, line)
			}
			index++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}
	end := start + len(result) - 1
	return result, start, end, index
}

func logFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
