package ui

import (
	"context"
	"fmt"
	"os"
	"scrapbtc/internal/processor"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ProgressModel struct {
	startHeight     int64
	endHeight       int64
	currentHeight   int64
	totalBlocks     int64
	processedBlocks int64
	failedBlocks    int64
	totalTxs        int64
	currentBlockTxs int
	startTime       time.Time
	lastUpdate      time.Time
	status          string
	errors          []string
	progressChan    <-chan processor.ProgressUpdate
	done            bool
}

type ProgressMsg processor.ProgressUpdate

func NewProgressModel(startHeight, endHeight int64, progressChan <-chan processor.ProgressUpdate) ProgressModel {
	return ProgressModel{
		startHeight:   startHeight,
		endHeight:     endHeight,
		totalBlocks:   endHeight - startHeight + 1,
		startTime:     time.Now(),
		lastUpdate:    time.Now(),
		status:        "Starting...",
		progressChan:  progressChan,
		errors:        make([]string, 0),
	}
}

func (m ProgressModel) Init() tea.Cmd {
	return m.waitForActivity()
}

func (m *ProgressModel) waitForActivity() tea.Cmd {
	return func() tea.Msg {
		select {
		case update, ok := <-m.progressChan:
			if !ok {
				return tea.Quit()
			}
			return ProgressMsg(update)
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	}
}

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case ProgressMsg:
		m.lastUpdate = time.Now()
		
		if msg.Error != nil {
			m.failedBlocks++
			m.errors = append(m.errors, fmt.Sprintf("Block %d: %s", msg.BlockHeight, msg.Error.Error()))
			if len(m.errors) > 5 {
				m.errors = m.errors[1:]
			}
		} else if msg.Status == "completed" {
			m.processedBlocks++
			m.totalTxs += int64(msg.TxCount)
			m.currentHeight = msg.BlockHeight
			m.currentBlockTxs = msg.TxCount
		}

		if msg.Status == "All blocks already processed" {
			m.status = "All blocks already processed"
			m.done = true
			return m, tea.Quit
		}

		return m, m.waitForActivity()

	case tea.QuitMsg:
		return m, nil
	}

	return m, nil
}

func (m ProgressModel) View() string {
	if m.done && m.status == "All blocks already processed" {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("2")).
			Render("✓ All blocks already processed\n")
	}

	elapsed := time.Since(m.startTime)
	progress := float64(m.processedBlocks) / float64(m.totalBlocks) * 100
	
	var eta time.Duration
	if m.processedBlocks > 0 {
		avgTimePerBlock := elapsed / time.Duration(m.processedBlocks)
		remainingBlocks := m.totalBlocks - m.processedBlocks
		eta = avgTimePerBlock * time.Duration(remainingBlocks)
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		MarginBottom(1)

	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7"))

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("1"))

	progressBar := m.renderProgressBar(progress)

	header := headerStyle.Render("🚀 Bitcoin Blockchain Scraper")
	
	stats := statsStyle.Render(fmt.Sprintf(
		"📊 Range: %d - %d | Current: %d\n"+
		"✅ Processed: %d/%d blocks (%.1f%%)\n"+
		"📈 Transactions: %d total | %d in current block\n"+
		"⏱️  Elapsed: %s | ETA: %s\n"+
		"❌ Failed: %d blocks",
		m.startHeight, m.endHeight, m.currentHeight,
		m.processedBlocks, m.totalBlocks, progress,
		m.totalTxs, m.currentBlockTxs,
		elapsed.Truncate(time.Second), eta.Truncate(time.Second),
		m.failedBlocks))

	var errorSection string
	if len(m.errors) > 0 {
		errorSection = "\n\n" + errorStyle.Render("Recent Errors:") + "\n"
		for _, err := range m.errors {
			errorSection += errorStyle.Render("• " + err) + "\n"
		}
	}

	return fmt.Sprintf("%s\n\n%s\n\n%s%s\n\nPress 'q' or Ctrl+C to quit",
		header, progressBar, stats, errorSection)
}

func (m ProgressModel) renderProgressBar(progress float64) string {
	width := 50
	filled := int(progress / 100 * float64(width))
	
	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("2"))
	
	return style.Render(fmt.Sprintf("[%s] %.1f%%", bar, progress))
}

func RunProgressUI(ctx context.Context, startHeight, endHeight int64, progressChan <-chan processor.ProgressUpdate) error {
	// Check if we have a TTY, if not use simple console output
	if !isInteractiveTerminal() {
		return runSimpleProgress(ctx, startHeight, endHeight, progressChan)
	}
	
	model := NewProgressModel(startHeight, endHeight, progressChan)
	
	p := tea.NewProgram(model, tea.WithAltScreen())
	
	go func() {
		<-ctx.Done()
		p.Quit()
	}()
	
	_, err := p.Run()
	return err
}

func isInteractiveTerminal() bool {
	// Simple check - if we can't open /dev/tty, we're not interactive
	file, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	file.Close()
	return true
}

func runSimpleProgress(ctx context.Context, startHeight, endHeight int64, progressChan <-chan processor.ProgressUpdate) error {
	totalBlocks := endHeight - startHeight + 1
	var processedBlocks, failedBlocks int64
	var totalTxs int64
	startTime := time.Now()
	
	fmt.Printf("Processing blocks from %d to %d (%d blocks total)\n", startHeight, endHeight, totalBlocks)
	
	for {
		select {
		case update, ok := <-progressChan:
			if !ok {
				elapsed := time.Since(startTime)
				fmt.Printf("\nProcessing completed!\n")
				fmt.Printf("Processed: %d blocks\n", processedBlocks)
				fmt.Printf("Failed: %d blocks\n", failedBlocks)
				fmt.Printf("Total transactions: %d\n", totalTxs)
				fmt.Printf("Total time: %s\n", elapsed.Truncate(time.Second))
				return nil
			}
			
			if update.Error != nil {
				failedBlocks++
				fmt.Printf("Error processing block %d: %s\n", update.BlockHeight, update.Error.Error())
			} else if update.Status == "completed" {
				processedBlocks++
				totalTxs += int64(update.TxCount)
				progress := float64(processedBlocks) / float64(totalBlocks) * 100
				fmt.Printf("Processed block %d (%d txs) - Progress: %.1f%% (%d/%d)\n", 
					update.BlockHeight, update.TxCount, progress, processedBlocks, totalBlocks)
			} else if update.Status == "All blocks already processed" {
				fmt.Println("All blocks already processed")
				return nil
			}
			
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
