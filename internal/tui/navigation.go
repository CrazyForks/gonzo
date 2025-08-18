package tui

import (
	"fmt"
	"regexp"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// Use tea.Quit directly instead of custom quit message

// handleKeyPress processes keyboard input
func (m *DashboardModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// HIGHEST PRIORITY: Filter input (must come before ANY other handlers)
	if m.filterActive {
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "escape", "esc":
			m.filterActive = false
			m.filterInput.Blur()
			// Clear filter and regenerate view
			m.filterInput.SetValue("")
			m.filterRegex = nil
			m.updateFilteredView()
			// Reset to a valid section for navigation
			if m.activeSection == SectionFilter {
				m.activeSection = SectionWords
			}
			return m, nil
		case "enter":
			// Exit filter input mode but keep filter applied
			m.filterActive = false  // Exit input mode to allow other keys
			m.filterInput.Blur()
			// Make sure filtered view is up to date
			m.updateFilteredView()
			// Switch to log viewer to allow navigation
			m.activeSection = SectionLogs
			return m, nil
		default:
			// ALL other keys (including 'q') go to filter input
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)

			// Update filter regex and regenerate filtered view
			oldRegex := m.filterRegex
			if m.filterInput.Value() != "" {
				if regex, err := regexp.Compile(m.filterInput.Value()); err == nil {
					m.filterRegex = regex
				}
			} else {
				m.filterRegex = nil
			}

			// Update filtered view if regex changed
			if (oldRegex == nil) != (m.filterRegex == nil) ||
				(oldRegex != nil && m.filterRegex != nil && oldRegex.String() != m.filterRegex.String()) {
				m.updateFilteredView()
			}

			return m, cmd
		}
	}

	// THIRD HIGHEST PRIORITY: Search input (must come before ANY other handlers)
	if m.searchActive {
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "escape", "esc":
			m.searchActive = false
			m.searchInput.Blur()
			// Clear search
			m.searchInput.SetValue("")
			m.searchTerm = ""
			// Reset to a valid section for navigation
			if m.activeSection == SectionFilter {
				m.activeSection = SectionWords
			}
			return m, nil
		case "enter":
			// Exit search input mode but keep search applied
			m.searchActive = false  // Exit input mode to allow other keys
			m.searchInput.Blur()
			// Update search term
			m.searchTerm = m.searchInput.Value()
			// Switch to log viewer to allow navigation
			m.activeSection = SectionLogs
			return m, nil
		default:
			// ALL other keys (including 'q') go to search input
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)

			// Update search term in real-time
			m.searchTerm = m.searchInput.Value()

			return m, cmd
		}
	}

	// FIRST PRIORITY: Handle help modal if active
	if m.showHelp {
		switch msg.String() {
		case "up", "k":
			m.infoViewport.ScrollUp(1)
			return m, nil
		case "down", "j":
			m.infoViewport.ScrollDown(1)
			return m, nil
		case "pgup":
			m.infoViewport.HalfPageUp()
			return m, nil
		case "pgdown":
			m.infoViewport.HalfPageDown()
			return m, nil
		case "?", "h", "escape", "esc":
			m.showHelp = false
			return m, nil
		}

		// Update viewport with any other keys
		var cmd tea.Cmd
		m.infoViewport, cmd = m.infoViewport.Update(msg)
		return m, cmd
	}

	// SECOND PRIORITY: Handle chat input if active - bypass ALL other shortcuts
	if m.showModal && m.chatActive {
		switch msg.String() {
		case "tab":
			// Tab exits chat mode and switches to info section
			m.chatActive = false
			m.chatInput.Blur()
			m.modalActiveSection = "info"
			return m, nil
		case "escape", "esc":
			m.chatActive = false
			m.chatInput.Blur()
			m.chatInput.SetValue("")
			return m, nil
		case "enter":
			if m.chatInput.Value() != "" && m.currentLogEntry != nil && m.aiClient != nil {
				question := m.chatInput.Value()
				m.chatHistory = append(m.chatHistory, fmt.Sprintf("You: %s", question))
				
				// Add working indicator to chat history
				m.chatHistory = append(m.chatHistory, fmt.Sprintf("AI: %s Working on it...", m.getChatSpinner()))
				m.chatAutoScroll = true  // Enable auto-scroll for new messages
				
				m.chatInput.SetValue("")
				// Keep chat mode active and focused after sending
				m.chatInput.Focus()
				m.chatAiAnalyzing = true  // Use chat-specific AI flag

				// Continue conversation with context
				return m, func() tea.Msg {
					result, err := m.aiClient.AnalyzeLogWithContext(
						m.currentLogEntry.Message,
						m.currentLogEntry.Severity,
						m.currentLogEntry.Timestamp.Format("2006-01-02 15:04:05.000"),
						m.currentLogEntry.Attributes,
						m.aiAnalysisResult,
						question,
					)
					return AIAnalysisMsg{Result: result, Error: err, IsChat: true}
				}
			}
			return m, nil
		case "ctrl+c":
			// Allow ctrl+c to quit even in chat mode
			return m, tea.Quit
		case "up", "k":
			// Allow scrolling in chat viewport when in chat mode
			m.chatViewport.ScrollUp(1)
			return m, nil
		case "down", "j":
			// Allow scrolling in chat viewport when in chat mode
			m.chatViewport.ScrollDown(1)
			return m, nil
		case "pgup":
			// Allow page scrolling in chat viewport when in chat mode
			m.chatViewport.HalfPageUp()
			return m, nil
		case "pgdown":
			// Allow page scrolling in chat viewport when in chat mode
			m.chatViewport.HalfPageDown()
			return m, nil
		default:
			// ALL other keys go to the text input - no shortcuts processed at all
			var cmd tea.Cmd
			m.chatInput, cmd = m.chatInput.Update(msg)
			return m, cmd
		}
	}

	// Critical keys that always work
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "escape", "esc":
		if m.showModelSelectionModal {
			m.showModelSelectionModal = false
			return m, nil
		}
		if m.showPatternsModal {
			m.showPatternsModal = false
			return m, nil
		}
		if m.showStatsModal {
			m.showStatsModal = false
			return m, nil
		}
		if m.showCountsModal {
			m.showCountsModal = false
			return m, nil
		}
		if m.showModal {
			m.showModal = false
			m.modalContent = ""
			// Reset viewport scroll position for next modal
			m.infoViewport.GotoTop()
			m.chatViewport.GotoTop()
			// Don't unpause if we're still in the log section
			// User needs to tab out to resume updates
			return m, nil
		}
		if m.filterActive {
			m.filterActive = false
			m.filterInput.Blur()
			// Clear filter and regenerate view
			m.filterInput.SetValue("")
			m.filterRegex = nil
			m.updateFilteredView()
			// Reset to a valid section for navigation
			if m.activeSection == SectionFilter {
				m.activeSection = SectionWords
			}
			return m, nil
		}
		if m.searchActive {
			m.searchActive = false
			m.searchInput.Blur()
			// Clear search
			m.searchInput.SetValue("")
			m.searchTerm = ""
			// Reset to a valid section for navigation
			if m.activeSection == SectionFilter {
				m.activeSection = SectionWords
			}
			return m, nil
		}
		// Clear applied filter/search even when not in input mode
		if m.filterRegex != nil || m.filterInput.Value() != "" || m.searchTerm != "" || m.searchInput.Value() != "" {
			// Clear all filter and search state
			m.filterActive = false
			m.searchActive = false
			m.filterInput.Blur()
			m.searchInput.Blur()
			m.filterInput.SetValue("")
			m.searchInput.SetValue("")
			m.filterRegex = nil
			m.searchTerm = ""
			m.updateFilteredView()
			// Reset to a valid section for navigation
			if m.activeSection == SectionFilter {
				m.activeSection = SectionWords
			}
			return m, nil
		}
	}

	// Global shortcuts (now handled after filter/search input)
	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "?", "h":
		m.showHelp = !m.showHelp
		return m, nil

	case "/":
		if !m.showModal && !m.searchActive {
			// Check if filter is already applied (not just active input)
			if m.filterRegex != nil || m.filterInput.Value() != "" {
				// Re-enter filter editing mode
				m.activeSection = SectionFilter
				m.filterActive = true
				m.filterInput.Focus()
			} else {
				// Start new filter
				m.activeSection = SectionFilter
				m.filterActive = true
				m.filterInput.SetValue("") // Clear any existing content
				m.filterRegex = nil        // Clear regex filter
				m.updateFilteredView()     // Update view with no filter
				m.filterInput.Focus()
			}
			return m, nil
		}

	case "s":
		if !m.showModal && !m.filterActive {
			// Check if search is already applied (not just active input)
			if m.searchTerm != "" || m.searchInput.Value() != "" {
				// Re-enter search editing mode
				m.activeSection = SectionFilter // Use the same section for UI
				m.searchActive = true
				m.searchInput.Focus()
			} else {
				// Start new search
				m.activeSection = SectionFilter // Use the same section for UI
				m.searchActive = true
				m.searchInput.SetValue("") // Clear any existing content
				m.searchTerm = ""          // Clear search term
				m.searchInput.Focus()
			}
			return m, nil
		}

	case "r":
		// Manual reset of frequency data and patterns
		if !m.showModal && !m.filterActive && !m.searchActive {
			// Reset drain3 tracking as well
			m.drain3LastProcessed = 0
			// Send manual reset message to trigger reset in app immediately
			return m, func() tea.Msg {
				return ManualResetMsg{}
			}
		}

	case "c":
		// Toggle Host/Service columns in log view
		if !m.showModal && !m.filterActive && !m.searchActive {
			m.showColumns = !m.showColumns
			return m, nil
		}
		
	case "i":
		// Toggle statistics modal
		if !m.showModal && !m.filterActive && !m.searchActive && !m.showHelp && !m.showPatternsModal && !m.showModelSelectionModal {
			m.showStatsModal = !m.showStatsModal
			return m, nil
		}

	case "m":
		// Model selection modal
		if !m.showModal && !m.filterActive && !m.searchActive && !m.showHelp && !m.showPatternsModal && !m.showStatsModal {
			if m.aiClient != nil && len(m.availableModelsList) > 0 {
				m.showModelSelectionModal = true
				m.selectedModelIndex = 0
				// Find current model in the list to pre-select it
				for i, model := range m.availableModelsList {
					if model == m.aiModelName {
						m.selectedModelIndex = i
						break
					}
				}
				return m, nil
			}
		}

	case " ":
		// Spacebar: Global pause/unpause toggle for entire UI
		if !m.showModal && !m.filterActive && !m.searchActive {
			wasPaused := m.viewPaused
			m.viewPaused = !m.viewPaused
			
			// If unpausing, process any accumulated logs
			if wasPaused && !m.viewPaused {
				// Process unprocessed logs through drain3
				if m.drain3Manager != nil {
					// Process all logs that haven't been processed yet
					for i := m.drain3LastProcessed; i < len(m.allLogEntries); i++ {
						m.drain3Manager.AddLogMessage(m.allLogEntries[i].Message)
					}
					m.drain3LastProcessed = len(m.allLogEntries)
				}
				
				// Update the filtered view with all accumulated logs
				m.updateFilteredView()
			}
			return m, nil
		}

	case "u":
		// Cycle to next update interval (forward)
		if !m.showModal && !m.filterActive && !m.searchActive {
			m.currentIntervalIdx = (m.currentIntervalIdx + 1) % len(m.availableIntervals)
			newInterval := m.availableIntervals[m.currentIntervalIdx]
			m.updateInterval = newInterval

			// Show feedback to user about new interval
			intervalStr := m.formatDuration(newInterval)
			m.modalContent = fmt.Sprintf("Update Interval Changed\n\nNew interval: %s\n\nPress 'u' for next, 'U' for previous interval.\nThis controls how often the dashboard refreshes.", intervalStr)
			m.showModal = true

			// Return message to update the main model's interval
			return m, func() tea.Msg {
				return UpdateIntervalMsg(newInterval)
			}
		}

	case "U":
		// Cycle to previous update interval (backward)
		if !m.showModal && !m.filterActive && !m.searchActive {
			m.currentIntervalIdx = (m.currentIntervalIdx - 1 + len(m.availableIntervals)) % len(m.availableIntervals)
			newInterval := m.availableIntervals[m.currentIntervalIdx]
			m.updateInterval = newInterval

			// Show feedback to user about new interval
			intervalStr := m.formatDuration(newInterval)
			m.modalContent = fmt.Sprintf("Update Interval Changed\n\nNew interval: %s\n\nPress 'u' for next, 'U' for previous interval.\nThis controls how often the dashboard refreshes.", intervalStr)
			m.showModal = true

			// Return message to update the main model's interval
			return m, func() tea.Msg {
				return UpdateIntervalMsg(newInterval)
			}
		}
	}

	// Patterns modal shortcuts
	if m.showPatternsModal {
		switch msg.String() {
		case "up", "k":
			m.infoViewport.ScrollUp(1)
			return m, nil
		case "down", "j":
			m.infoViewport.ScrollDown(1)
			return m, nil
		case "pgup":
			m.infoViewport.HalfPageUp()
			return m, nil
		case "pgdown":
			m.infoViewport.HalfPageDown()
			return m, nil
		case "escape", "esc":
			m.showPatternsModal = false
			return m, nil
		}

		// Update patterns modal viewport with scroll messages
		var cmd tea.Cmd
		m.infoViewport, cmd = m.infoViewport.Update(msg)
		return m, cmd
	}
	
	// Statistics modal shortcuts
	if m.showStatsModal {
		switch msg.String() {
		case "up", "k":
			m.infoViewport.ScrollUp(1)
			return m, nil
		case "down", "j":
			m.infoViewport.ScrollDown(1)
			return m, nil
		case "pgup":
			m.infoViewport.HalfPageUp()
			return m, nil
		case "pgdown":
			m.infoViewport.HalfPageDown()
			return m, nil
		case "i":
			// Allow 'i' to toggle stats modal off
			m.showStatsModal = false
			return m, nil
		case "escape", "esc":
			m.showStatsModal = false
			return m, nil
		}

		// Update statistics modal viewport with scroll messages
		var cmd tea.Cmd
		m.infoViewport, cmd = m.infoViewport.Update(msg)
		return m, cmd
	}

	// Counts modal keyboard navigation
	if m.showCountsModal {
		switch msg.String() {
		case "up", "k":
			m.infoViewport.ScrollUp(1)
			return m, nil
		case "down", "j":
			m.infoViewport.ScrollDown(1)
			return m, nil
		case "pgup":
			m.infoViewport.HalfPageUp()
			return m, nil
		case "pgdown":
			m.infoViewport.HalfPageDown()
			return m, nil
		case "escape", "esc":
			m.showCountsModal = false
			return m, nil
		}

		// Update counts modal viewport with scroll messages
		var cmd tea.Cmd
		m.infoViewport, cmd = m.infoViewport.Update(msg)
		return m, cmd
	}
	
	// Model selection modal shortcuts
	if m.showModelSelectionModal {
		switch msg.String() {
		case "up", "k":
			if m.selectedModelIndex > 0 {
				m.selectedModelIndex--
			}
			return m, nil
		case "down", "j":
			if m.selectedModelIndex < len(m.availableModelsList)-1 {
				m.selectedModelIndex++
			}
			return m, nil
		case "pgup":
			// Page up - move up 10 models
			m.selectedModelIndex = max(0, m.selectedModelIndex-10)
			return m, nil
		case "pgdown":
			// Page down - move down 10 models
			m.selectedModelIndex = min(len(m.availableModelsList)-1, m.selectedModelIndex+10)
			return m, nil
		case "home":
			// Go to first model
			m.selectedModelIndex = 0
			return m, nil
		case "end":
			// Go to last model
			m.selectedModelIndex = len(m.availableModelsList) - 1
			return m, nil
		case "enter":
			// Switch to selected model
			if m.selectedModelIndex >= 0 && m.selectedModelIndex < len(m.availableModelsList) {
				newModel := m.availableModelsList[m.selectedModelIndex]
				return m.switchToModel(newModel)
			}
			return m, nil
		case "escape", "esc":
			m.showModelSelectionModal = false
			return m, nil
		}
		return m, nil
	}

	// Modal view shortcuts (chat mode handled above at function start)
	if m.showModal {
		// Check if this is a log details modal (split layout) or single modal

		if m.currentLogEntry != nil {
			// Handle split modal navigation and scrolling
			switch msg.String() {
			case "tab":
				// Always allow tab navigation between panes
				// Switch between info and chat sections in modal
				if m.modalActiveSection == "info" {
					m.modalActiveSection = "chat"
					// Check if AI is configured before enabling chat
					if !m.aiConfigured {
						// Show error in chat area instead of enabling chat
						chatError := fmt.Sprintf("AI Chat Not Available\n\nError: %s\n\nTo configure AI:\n• Set OPENAI_API_KEY environment variable\n• For local AI: Set OPENAI_API_BASE\n• Use --ai-model flag to specify model", m.aiErrorMessage)
						m.chatHistory = []string{fmt.Sprintf("System: %s", chatError)}
						m.chatAutoScroll = true  // Enable auto-scroll for error message
						return m, nil
					}
					// Automatically enter chat mode when switching to chat pane
					if m.currentLogEntry != nil && m.aiClient != nil {
						m.chatActive = true
						m.chatInput.Focus()
						return m, textarea.Blink
					}
				} else {
					m.modalActiveSection = "info"
					// Exit chat mode when switching away from chat pane
					if m.chatActive {
						m.chatActive = false
						m.chatInput.Blur()
					}
				}
				return m, nil
			case "up", "k":
				// Handle scrolling based on active section
				if m.modalActiveSection == "info" {
					m.infoViewport.ScrollUp(1)
					return m, nil
				} else if m.modalActiveSection == "chat" {
					// Always allow scrolling in chat section (single-line input doesn't use up/down)
					m.chatViewport.ScrollUp(1)
					return m, nil
				}
			case "down", "j":
				// Handle scrolling based on active section
				if m.modalActiveSection == "info" {
					m.infoViewport.ScrollDown(1)
					return m, nil
				} else if m.modalActiveSection == "chat" {
					// Always allow scrolling in chat section (single-line input doesn't use up/down)
					m.chatViewport.ScrollDown(1)
					return m, nil
				}
			case "pgup":
				// Handle page navigation based on active section
				if m.modalActiveSection == "info" {
					m.infoViewport.HalfPageUp()
				} else if m.modalActiveSection == "chat" {
					m.chatViewport.HalfPageUp()
				}
				return m, nil
			case "pgdown":
				// Handle page navigation based on active section
				if m.modalActiveSection == "info" {
					m.infoViewport.HalfPageDown()
				} else if m.modalActiveSection == "chat" {
					m.chatViewport.HalfPageDown()
				}
				return m, nil
			case "i":
				// Only handle AI analysis if not actively typing in chat
				if !m.chatActive || m.modalActiveSection == "info" {
					// AI analysis only available when viewing log details and AI client is available
					if m.currentLogEntry != nil && m.aiClient != nil && !m.aiAnalyzing {
						m.aiAnalyzing = true
						m.aiAnalysisResult = "Analyzing..."

						// Start AI analysis in background
						return m, func() tea.Msg {
							result, err := m.aiClient.AnalyzeLog(
								m.currentLogEntry.Message,
								m.currentLogEntry.Severity,
								m.currentLogEntry.Timestamp.Format("2006-01-02 15:04:05.000"),
								m.currentLogEntry.Attributes,
							)
							return AIAnalysisMsg{Result: result, Error: err, IsChat: false}
						}
					}
				}
			case "m":
				// Model selection modal - only when not in chat mode
				if !m.chatActive && m.aiClient != nil && len(m.availableModelsList) > 0 {
					m.showModelSelectionModal = true
					m.selectedModelIndex = 0
					// Find current model in the list to pre-select it
					for i, model := range m.availableModelsList {
						if model == m.aiModelName {
							m.selectedModelIndex = i
							break
						}
					}
					return m, nil
				}
			case "escape", "esc": // escape to close modal (only if not in chat mode)
				m.showModal = false
				m.modalContent = ""
				m.currentLogEntry = nil // Clear current log entry when closing modal
				// Reset viewport scroll position for next modal
				m.infoViewport.GotoTop()
				m.chatViewport.GotoTop()
				m.aiAnalysisResult = ""
				m.chatHistory = []string{}
				m.chatActive = false
				m.chatAiAnalyzing = false  // Reset chat AI state
				m.chatInput.SetValue("")
				return m, nil
			}

			// Update active viewport with scroll messages
			var cmd tea.Cmd
			if m.modalActiveSection == "info" {
				m.infoViewport, cmd = m.infoViewport.Update(msg)
			} else {
				m.chatViewport, cmd = m.chatViewport.Update(msg)
			}
			return m, cmd
		} else {
			// Handle single modal scrolling and shortcuts
			switch msg.String() {
			case "up", "k":
				m.infoViewport.ScrollUp(1)
				return m, nil
			case "down", "j":
				m.infoViewport.ScrollDown(1)
				return m, nil
			case "pgup":
				m.infoViewport.HalfPageUp()
				return m, nil
			case "pgdown":
				m.infoViewport.HalfPageDown()
				return m, nil
			case "escape", "esc":
				m.showModal = false
				m.modalContent = ""
				return m, nil
			}

			// Update single viewport with scroll messages
			var cmd tea.Cmd
			m.infoViewport, cmd = m.infoViewport.Update(msg)
			return m, cmd
		}
	}


	// Navigation shortcuts
	switch msg.String() {
	case "tab":
		m.nextSection()
		return m, nil

	case "shift+tab":
		m.prevSection()
		return m, nil

	case "up", "k":
		m.moveSelection(-1)
		return m, nil

	case "down", "j":
		m.moveSelection(1)
		return m, nil

	case "home":
		// Home key: In log viewer section, scroll to top and stop auto-scroll
		if m.activeSection == SectionLogs {
			m.selectedLogIndex = 0
			m.logAutoScroll = false // Stop auto-scrolling when at top
			return m, nil
		}

	case "end":
		// End key: In log viewer section, scroll to latest and resume auto-scroll
		if m.activeSection == SectionLogs {
			m.selectedLogIndex = max(0, len(m.logEntries)-1)
			m.logAutoScroll = true // Resume auto-scrolling
			return m, nil
		}

	case "pgup":
		// Page Up: In log viewer section, move up by page
		if m.activeSection == SectionLogs {
			pageSize := 10 // Move by 10 entries
			m.selectedLogIndex = max(0, m.selectedLogIndex-pageSize)
			if m.selectedLogIndex == 0 {
				m.logAutoScroll = false // Stop auto-scroll when at top
			}
			return m, nil
		}

	case "pgdown", "pagedown":
		// Page Down: In log viewer section, move down by page
		if m.activeSection == SectionLogs {
			pageSize := 10 // Move by 10 entries
			maxIndex := max(0, len(m.logEntries)-1)
			m.selectedLogIndex = min(maxIndex, m.selectedLogIndex+pageSize)
			if m.selectedLogIndex == maxIndex {
				m.logAutoScroll = true // Resume auto-scroll when at bottom
			}
			return m, nil
		}

	case "i":
		// If in log section, open modal and handle AI analysis
		if m.activeSection == SectionLogs {
			if m.selectedLogIndex >= 0 && m.selectedLogIndex < len(m.logEntries) {
				entry := m.logEntries[m.selectedLogIndex]
				m.currentLogEntry = &entry
				m.modalContent = m.formatLogDetails(entry, 60)
				m.showModal = true
				m.modalReady = false // Reset viewport
				// Explicitly reset viewport scroll position
				m.infoViewport.GotoTop()
				m.chatViewport.GotoTop()
				
				// Clear any previous AI analysis result - user must press 'i' to analyze
				m.aiAnalysisResult = ""
			}
		}
		return m, nil

	case "enter":
		return m.showDetails()
	}

	return m, nil
}

// nextSection moves to the next section
func (m *DashboardModel) nextSection() {
	sections := []Section{SectionWords, SectionAttributes, SectionDistribution, SectionCounts, SectionLogs}

	// If current section is not in the list (e.g., SectionFilter), start from the first section
	if m.activeSection == SectionFilter {
		m.activeSection = SectionWords
		return
	}

	// Find current section and move to next
	for i, section := range sections {
		if section == m.activeSection {
			m.activeSection = sections[(i+1)%len(sections)]
			// No longer pause when entering log section - logs keep streaming
			break
		}
	}
}

// prevSection moves to the previous section
func (m *DashboardModel) prevSection() {
	sections := []Section{SectionWords, SectionAttributes, SectionDistribution, SectionCounts, SectionLogs}

	// If current section is not in the list (e.g., SectionFilter), start from the last section
	if m.activeSection == SectionFilter {
		m.activeSection = SectionLogs
		return
	}

	// Find current section and move to previous
	for i, section := range sections {
		if section == m.activeSection {
			m.activeSection = sections[(i-1+len(sections))%len(sections)]
			// No longer pause when entering log section - logs keep streaming
			break
		}
	}
}

// moveSelection moves the selection within the active section
func (m *DashboardModel) moveSelection(delta int) {
	// Special handling for log section
	if m.activeSection == SectionLogs {
		maxItems := len(m.logEntries)
		if maxItems == 0 {
			return
		}

		newIndex := m.selectedLogIndex + delta

		// Constrain to bounds without wrapping
		if newIndex < 0 {
			newIndex = 0
		} else if newIndex >= maxItems {
			newIndex = maxItems - 1
		}

		m.selectedLogIndex = newIndex

		// Update auto-scroll based on position
		if m.selectedLogIndex == 0 {
			// At top - disable auto-scroll
			m.logAutoScroll = false
		} else if m.selectedLogIndex == maxItems-1 {
			// At bottom - enable auto-scroll
			m.logAutoScroll = true
		}
		// For positions in between, keep current auto-scroll state
		
		return
	}

	if m.snapshot == nil {
		return
	}

	var maxItems int
	switch m.activeSection {
	case SectionWords:
		// Limit to 10 visible items in words chart - use lifetime data
		lifetimeWords := m.getLifetimeWordEntries()
		maxItems = min(len(lifetimeWords), 10)
	case SectionAttributes:
		// Limit to 10 visible items in attributes chart - use lifetime data
		lifetimeAttrs := m.getLifetimeAttributeEntries()
		maxItems = min(len(lifetimeAttrs), 10)
	case SectionDistribution:
		maxItems = 7 // Fixed number of distribution ranges
	case SectionCounts:
		maxItems = len(m.countsHistory)
	default:
		return
	}

	if maxItems == 0 {
		return
	}

	current := m.selectedIndex[m.activeSection]
	newIndex := current + delta

	// Constrain to bounds without wrapping
	if newIndex < 0 {
		newIndex = 0
	} else if newIndex >= maxItems {
		newIndex = maxItems - 1
	}

	m.selectedIndex[m.activeSection] = newIndex
}

// showDetails shows details for the selected item
func (m *DashboardModel) showDetails() (tea.Model, tea.Cmd) {
	// Special handling for log details
	if m.activeSection == SectionLogs {
		if m.selectedLogIndex >= 0 && m.selectedLogIndex < len(m.logEntries) {
			entry := m.logEntries[m.selectedLogIndex]
			m.currentLogEntry = &entry // Store current log entry for AI analysis
			m.modalContent = m.formatLogDetails(entry, 60)
			m.showModal = true
			m.modalReady = false       // Reset viewport
			// Explicitly reset viewport scroll position
			m.infoViewport.GotoTop()
			m.chatViewport.GotoTop()
			m.aiAnalysisResult = ""    // Clear previous analysis
			m.chatHistory = []string{} // Clear chat history
			m.chatAiAnalyzing = false  // Reset chat AI state
		}
		return m, nil
	}

	// For lifetime data, we don't need the snapshot check since lifetime data is always available

	selectedIdx := m.selectedIndex[m.activeSection]
	var content string

	switch m.activeSection {
	case SectionWords:
		lifetimeWords := m.getLifetimeWordEntries()
		if selectedIdx < len(lifetimeWords) {
			entry := lifetimeWords[selectedIdx]
			// Toggle word highlighting: if same word is already being searched, clear it; otherwise apply it
			if m.searchTerm == entry.Term {
				// Clear search highlighting if same word is already being searched
				m.searchTerm = ""
			} else {
				// Apply the selected word for search highlighting (like 's' command)
				m.searchTerm = entry.Term
			}
			// Don't show modal, just return
			return m, nil
		}

	case SectionAttributes:
		lifetimeAttrs := m.getLifetimeAttributeEntries()
		if selectedIdx < len(lifetimeAttrs) {
			entry := lifetimeAttrs[selectedIdx]
			// Use full available content width for the modal
			contentWidth := m.width - 16 // Modal margins and borders
			if contentWidth < 60 {
				contentWidth = 60 // Minimum reasonable width
			}
			content = m.formatAttributeValuesModal(entry, contentWidth)
			// Clear log entry to ensure single modal layout for attributes
			m.currentLogEntry = nil
		}

	case SectionDistribution:
		// Show patterns modal with all patterns
		if m.drain3Manager != nil {
			m.showPatternsModal = true
			// Clear log entry to ensure single modal layout for patterns
			m.currentLogEntry = nil
			return m, nil
		}
		
	case SectionCounts:
		// Show counts modal with heatmap and analysis
		m.showCountsModal = true
		// Clear log entry to ensure single modal layout for counts
		m.currentLogEntry = nil
		return m, nil
	}

	if content != "" {
		m.modalContent = content
		m.showModal = true
	}

	return m, nil
}
