package ui

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"llama-manager/benchmark"
	"llama-manager/config"
	"llama-manager/hardware"
	"llama-manager/model"
	"llama-manager/profile"
	"llama-manager/runner"
)

type ServerUIStatus int

const (
	UIStatusStopped ServerUIStatus = iota
	UIStatusStarting
	UIStatusRunning
	UIStatusFailed
)

type ScreenMode int

const (
	ScreenBrowser ScreenMode = iota
	ScreenDashboard
	ScreenCollections
	ScreenBenchmarkProgress
	ScreenPerformanceDashboard
	ScreenServerMonitor
	ScreenSettings
	ScreenDownloader
	ScreenTagsEditor
	ScreenNotesEditor
)

type SidebarItemType int

const (
	ItemSectionHeader SidebarItemType = iota
	ItemFolderHeader
	ItemModelEntry
)

type SidebarItem struct {
	Type           SidebarItemType
	Label          string
	ModelIdx       int
	ModelPath      string
	CollectionName string
	Expanded       bool
}

type BrowserModel struct {
	config              *config.Config
	srvRunner           *runner.ServerRunner
	models              []*model.GGUFMetadata
	filtered            []int // indices in m.models
	selected            int   // index in m.sidebarItems
	scrollOffset        int
	loading             bool
	err                 error
	searchActive        bool
	searchInput         textinput.Model
	width, height       int

	serverUIStatus      ServerUIStatus
	serverErr           error
	runningModelPath    string
	hardwareSpecs       *hardware.HardwareSpecs
	profiles            []*profile.Profile
	screenMode          ScreenMode
	dashboard           *DashboardModel
	collectionsModel    *CollectionsModel
	editorModel         *EditorModel
	expandedCollections map[string]bool
	sidebarItems        []SidebarItem
	benchmarkProgress   *BenchmarkProgressModel
	perfDashboard       *PerformanceDashboardModel
	monitorModel        *MonitorModel
	lifecycleModel      *LifecycleModel
	downloaderModel     *DownloaderModel
	downloadQueue       *model.DownloadQueue
}

func NewBrowserModel(cfg *config.Config, srv *runner.ServerRunner) *BrowserModel {
	ti := textinput.New()
	ti.Placeholder = "Type to search..."
	ti.CharLimit = 156
	ti.Width = 30

	q := model.NewDownloadQueue(cfg.Paths.Models, cfg.HFToken)

	return &BrowserModel{
		config:              cfg,
		srvRunner:           srv,
		loading:             true,
		searchInput:         ti,
		serverUIStatus:      UIStatusStopped,
		screenMode:          ScreenBrowser,
		expandedCollections: make(map[string]bool),
		sidebarItems:        []SidebarItem{},
		monitorModel:        NewMonitorModel(srv),
		lifecycleModel:      NewLifecycleModel(cfg, srv),
		downloadQueue:       q,
		downloaderModel:     NewDownloaderModel(cfg, q),
	}
}

type discoverMsg struct {
	models []*model.GGUFMetadata
	err    error
}

func discoverCmd(modelsDir string) tea.Cmd {
	return func() tea.Msg {
		models, err := model.DiscoverModels(modelsDir)
		return discoverMsg{models: models, err: err}
	}
}

type startServerMsg struct {
	err error
}

func startServerCmd(srv *runner.ServerRunner, llamaCppDir string, modelPath string, ctxSize uint32, threads int, gpuLayers int, batchSize int, host string, port int) tea.Cmd {
	return func() tea.Msg {
		err := srv.Start(llamaCppDir, modelPath, ctxSize, threads, gpuLayers, batchSize, host, port)
		return startServerMsg{err: err}
	}
}

type healthCheckMsg struct {
	online bool
}

func checkHealthCmd(port int) tea.Cmd {
	return func() tea.Msg {
		client := http.Client{
			Timeout: 200 * time.Millisecond,
		}
		// Poll for health up to 10 times (5 seconds total)
		for i := 0; i < 10; i++ {
			time.Sleep(500 * time.Millisecond)
			resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return healthCheckMsg{online: true}
				}
			}
		}
		return healthCheckMsg{online: false}
	}
}

type hardwareDetectMsg struct {
	specs *hardware.HardwareSpecs
	err   error
}

func detectHardwareCmd() tea.Msg {
	specs, err := hardware.DetectHardware()
	return hardwareDetectMsg{specs: specs, err: err}
}

type profilesMsg struct {
	profiles []*profile.Profile
	err      error
}

func (m *BrowserModel) loadProfilesCmd() tea.Cmd {
	return func() tea.Msg {
		profs, err := profile.LoadAll(m.config.Paths.Profiles)
		return profilesMsg{profiles: profs, err: err}
	}
}

type benchmarkMsg struct {
	step BenchmarkProgressStep
	res  *benchmark.BenchmarkResult
	err  error
	ch   chan benchmarkMsg
}

func (m *BrowserModel) startBenchmark(targetModel *model.GGUFMetadata) tea.Cmd {
	ch := make(chan benchmarkMsg)

	go func() {
		res, err := benchmark.RunBenchmark(
			m.config.Paths.LlamaCPP,
			targetModel,
			m.hardwareSpecs,
			m.config,
			func(stepNum int) {
				var step BenchmarkProgressStep
				switch stepNum {
				case 0:
					step = StepBooting
				case 1:
					step = StepRunningPrompt
				case 2:
					step = StepSavingData
				}
				ch <- benchmarkMsg{step: step, ch: ch}
			},
		)
		if err != nil {
			ch <- benchmarkMsg{step: StepError, err: err, ch: ch}
			return
		}

		err = benchmark.SaveResult(m.config.Paths.Benchmarks, res)
		if err != nil {
			ch <- benchmarkMsg{step: StepError, err: fmt.Errorf("failed to save result: %w", err), ch: ch}
			return
		}

		ch <- benchmarkMsg{step: StepDone, res: res, ch: ch}
	}()

	return m.readBenchmarkChan(ch)
}

func (m *BrowserModel) readBenchmarkChan(ch chan benchmarkMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

type downloadQueueMsg struct {
	task *model.DownloadTask
}

func (m *BrowserModel) readDownloadQueueChan() tea.Cmd {
	return func() tea.Msg {
		qChan := m.downloadQueue.GetChan()
		task, ok := <-qChan
		if !ok {
			return nil
		}
		return downloadQueueMsg{task: task}
	}
}

type tickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m *BrowserModel) Init() tea.Cmd {
	return tea.Batch(
		discoverCmd(m.config.Paths.Models),
		m.loadProfilesCmd(),
		detectHardwareCmd,
		tickCmd(),
		m.readDownloadQueueChan(),
	)
}

func (m *BrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if (m.screenMode == ScreenTagsEditor || m.screenMode == ScreenNotesEditor) && m.editorModel != nil {
		if _, isSizeMsg := msg.(tea.WindowSizeMsg); !isSizeMsg {
			cmd, done := m.editorModel.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			if done {
				m.screenMode = ScreenBrowser
				m.editorModel = nil
			}
			return m, tea.Batch(cmds...)
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case discoverMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.models = msg.models
			m.filterModels()

			// Select last selected model if found
			if m.config.LastSelectedModel != "" {
				for i, item := range m.sidebarItems {
					if item.Type == ItemModelEntry && item.ModelPath == m.config.LastSelectedModel {
						m.selected = i
						break
					}
				}
			}
		}

	case hardwareDetectMsg:
		if msg.err == nil {
			m.hardwareSpecs = msg.specs
		}

	case profilesMsg:
		if msg.err == nil {
			m.profiles = msg.profiles
			m.rebuildSidebar()
		}

	case downloadQueueMsg:
		if m.downloaderModel != nil {
			_, cmd := m.downloaderModel.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if msg.task != nil && msg.task.Status == model.StatusCompleted {
			cmds = append(cmds, discoverCmd(m.config.Paths.Models))
		}
		cmds = append(cmds, m.readDownloadQueueChan())

	case updateMsg:
		if m.lifecycleModel != nil {
			_, cmd := m.lifecycleModel.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case benchmarkMsg:
		if m.benchmarkProgress == nil {
			break
		}
		if msg.err != nil {
			m.benchmarkProgress.Step = StepError
			m.benchmarkProgress.Err = msg.err
			break
		}

		m.benchmarkProgress.Step = msg.step
		if msg.step == StepDone {
			history, err := benchmark.LoadHistory(m.config.Paths.Benchmarks)
			if err == nil {
				m.perfDashboard = NewPerformanceDashboardModel(history)
			}
		} else {
			cmds = append(cmds, m.readBenchmarkChan(msg.ch))
		}

	case startServerMsg:
		if msg.err != nil {
			m.serverUIStatus = UIStatusFailed
			m.serverErr = msg.err
		}

	case healthCheckMsg:
		if msg.online {
			m.serverUIStatus = UIStatusRunning
		} else {
			// If it timed out, check if process is still running
			status, _, port := m.srvRunner.GetStatus()
			if status == runner.StatusRunning {
				m.serverUIStatus = UIStatusStarting
				// Retry health check
				cmds = append(cmds, checkHealthCmd(port))
			} else {
				m.serverUIStatus = UIStatusFailed
				m.serverErr = fmt.Errorf("server process terminated or failed to respond")
			}
		}

	case tickMsg:
		status, runModel, port := m.srvRunner.GetStatus()
		m.runningModelPath = runModel

		switch status {
		case runner.StatusRunning:
			if m.serverUIStatus == UIStatusStopped || m.serverUIStatus == UIStatusFailed {
				m.serverUIStatus = UIStatusStarting
				cmds = append(cmds, checkHealthCmd(port))
			}
		case runner.StatusFailed:
			m.serverUIStatus = UIStatusFailed
		case runner.StatusStopped:
			m.serverUIStatus = UIStatusStopped
		}

		cmds = append(cmds, tickCmd())

	case tea.KeyMsg:
		if m.screenMode == ScreenDashboard && m.dashboard != nil {
			switch msg.String() {
			case "left", "h":
				m.dashboard.CycleProfile(-1)
			case "right", "l":
				m.dashboard.CycleProfile(1)
			case "esc", "c", "C":
				m.screenMode = ScreenBrowser
			case "enter", "y", "Y":
				p := m.dashboard.ActiveProfile()
				if p != nil {
					// Persist profile selection
					m.config.ModelProfiles[m.dashboard.Model.FilePath] = p.Name
					
					// Record successful launch
					m.config.RecordLaunch(m.dashboard.Model.FilePath)
					
					_ = m.config.Save()
					m.rebuildSidebar()

					// Stop server and launch with profile settings
					m.serverUIStatus = UIStatusStarting
					m.serverErr = nil
					_ = m.srvRunner.Stop()

					cmds = append(cmds, startServerCmd(
						m.srvRunner,
						m.config.Paths.LlamaCPP,
						m.dashboard.Model.FilePath,
						p.Context,
						p.Threads,
						p.GPULayers,
						p.BatchSize,
						p.Host,
						p.Port,
					))
					cmds = append(cmds, checkHealthCmd(p.Port))
				}
				m.screenMode = ScreenBrowser
			}
		} else if m.screenMode == ScreenCollections && m.collectionsModel != nil {
			switch msg.String() {
			case "esc", "c", "C":
				if m.collectionsModel.mode == CollectionsCreateMode {
					cmd := m.collectionsModel.Update(msg)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				} else {
					m.screenMode = ScreenBrowser
					m.rebuildSidebar()
				}
			default:
				cmd := m.collectionsModel.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		} else if m.screenMode == ScreenBenchmarkProgress && m.benchmarkProgress != nil {
			switch msg.String() {
			case "esc", "enter", "c", "C":
				if m.benchmarkProgress.Step == StepDone {
					m.screenMode = ScreenPerformanceDashboard
				} else if m.benchmarkProgress.Step == StepError {
					m.screenMode = ScreenBrowser
				}
			}
		} else if m.screenMode == ScreenPerformanceDashboard && m.perfDashboard != nil {
			switch msg.String() {
			case "esc", "q", "ctrl+c":
				m.screenMode = ScreenBrowser
			}
		} else if m.screenMode == ScreenServerMonitor && m.monitorModel != nil {
			switch msg.String() {
			case "esc", "c", "C":
				m.screenMode = ScreenBrowser
				m.rebuildSidebar()
			default:
				cmd := m.monitorModel.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		} else if m.screenMode == ScreenSettings && m.lifecycleModel != nil {
			if m.lifecycleModel.tokenEditActive {
				_, cmd := m.lifecycleModel.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			} else {
				switch msg.String() {
				case "esc":
					m.screenMode = ScreenBrowser
					m.rebuildSidebar()
				case "c", "C":
					if m.lifecycleModel.state != StateChecking && m.lifecycleModel.state != StateDownloading && m.lifecycleModel.state != StateExtracting && m.lifecycleModel.state != StateVerifying && m.lifecycleModel.state != StateRollingBack {
						cmds = append(cmds, m.lifecycleModel.StartCheckOnly())
					}
				case "u", "U":
					if m.lifecycleModel.state != StateChecking && m.lifecycleModel.state != StateDownloading && m.lifecycleModel.state != StateExtracting && m.lifecycleModel.state != StateVerifying && m.lifecycleModel.state != StateRollingBack {
						cmds = append(cmds, m.lifecycleModel.StartUpdate())
					}
				case "r", "R":
					if m.lifecycleModel.hasBackup && m.lifecycleModel.state != StateChecking && m.lifecycleModel.state != StateDownloading && m.lifecycleModel.state != StateExtracting && m.lifecycleModel.state != StateVerifying && m.lifecycleModel.state != StateRollingBack {
						cmds = append(cmds, m.lifecycleModel.StartRollback())
					}
				case "t", "T":
					_, cmd := m.lifecycleModel.Update(msg)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			}
		} else if m.screenMode == ScreenDownloader && m.downloaderModel != nil {
			switch msg.String() {
			case "esc":
				m.downloaderModel.urlInput.Blur()
				m.downloaderModel.filenameInput.Blur()
				m.screenMode = ScreenBrowser
				m.rebuildSidebar()
			default:
				_, cmd := m.downloaderModel.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		} else if m.searchActive {
			switch msg.String() {
			case "enter":
				m.searchActive = false
				m.searchInput.Blur()
			case "esc":
				m.searchActive = false
				m.searchInput.SetValue("")
				m.searchInput.Blur()
				m.filterModels()
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				cmds = append(cmds, cmd)
				m.filterModels()
			}
		} else {
			switch msg.String() {
			case "q", "ctrl+c":
				_ = m.srvRunner.Stop()
				return m, tea.Quit

			case "up", "k":
				m.moveSelection(-1)

			case "down", "j":
				m.moveSelection(1)

			case "/":
				m.searchActive = true
				m.searchInput.Focus()
				m.searchInput.SetValue("")
				m.filterModels()

			case "s", "S":
				m.serverUIStatus = UIStatusStopped
				m.serverErr = nil
				_ = m.srvRunner.Stop()

			case "f", "F":
				if m.selected >= 0 && m.selected < len(m.sidebarItems) {
					item := m.sidebarItems[m.selected]
					if item.Type == ItemModelEntry {
						m.config.ToggleFavorite(item.ModelPath)
						_ = m.config.Save()
						m.rebuildSidebar()
					}
				}

			case "c", "C":
				if m.selected >= 0 && m.selected < len(m.sidebarItems) {
					item := m.sidebarItems[m.selected]
					if item.Type == ItemModelEntry {
						selectedModel := m.models[item.ModelIdx]
						m.collectionsModel = NewCollectionsModel(item.ModelPath, selectedModel.Name, m.config)
						m.screenMode = ScreenCollections
					}
				}

			case "t", "T":
				if m.selected >= 0 && m.selected < len(m.sidebarItems) {
					item := m.sidebarItems[m.selected]
					if item.Type == ItemModelEntry {
						selectedModel := m.models[item.ModelIdx]
						m.editorModel = NewEditorModel(item.ModelPath, selectedModel.Name, m.config, EditorTags)
						m.screenMode = ScreenTagsEditor
					}
				}

			case "n", "N":
				if m.selected >= 0 && m.selected < len(m.sidebarItems) {
					item := m.sidebarItems[m.selected]
					if item.Type == ItemModelEntry {
						selectedModel := m.models[item.ModelIdx]
						m.editorModel = NewEditorModel(item.ModelPath, selectedModel.Name, m.config, EditorNotes)
						m.screenMode = ScreenNotesEditor
					}
				}

			case "b", "B":
				if m.selected >= 0 && m.selected < len(m.sidebarItems) {
					item := m.sidebarItems[m.selected]
					if item.Type == ItemModelEntry {
						selectedModel := m.models[item.ModelIdx]
						m.benchmarkProgress = NewBenchmarkProgressModel(selectedModel.Name)
						m.screenMode = ScreenBenchmarkProgress
						_ = m.srvRunner.Stop()
						m.serverUIStatus = UIStatusStopped
						cmds = append(cmds, m.startBenchmark(selectedModel))
					}
				}

			case "v", "V":
				history, err := benchmark.LoadHistory(m.config.Paths.Benchmarks)
				if err == nil {
					m.perfDashboard = NewPerformanceDashboardModel(history)
					m.screenMode = ScreenPerformanceDashboard
				}

			case "m", "M":
				m.monitorModel.Refresh()
				m.screenMode = ScreenServerMonitor

			case "u", "U":
				m.lifecycleModel.RefreshLocalVersion()
				m.lifecycleModel.RefreshBackupStatus()
				m.screenMode = ScreenSettings
				cmds = append(cmds, m.lifecycleModel.StartCheckOnly())

			case "d", "D":
				m.downloaderModel.focus = FocusURL
				m.downloaderModel.urlInput.Focus()
				m.screenMode = ScreenDownloader

			case "space", "enter":
				if m.selected >= 0 && m.selected < len(m.sidebarItems) {
					item := m.sidebarItems[m.selected]
					if item.Type == ItemFolderHeader {
						m.expandedCollections[item.CollectionName] = !m.expandedCollections[item.CollectionName]
						m.rebuildSidebar()
					} else if item.Type == ItemModelEntry {
						selectedModel := m.models[item.ModelIdx]
						profName := m.config.ModelProfiles[selectedModel.FilePath]
						if profName == "" {
							profName = "Balanced"
						}
						m.dashboard = NewDashboardModel(selectedModel, m.hardwareSpecs, m.profiles, profName)
						m.screenMode = ScreenDashboard
					}
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *BrowserModel) filterModels() {
	query := strings.TrimSpace(strings.ToLower(m.searchInput.Value()))
	if query == "" {
		m.filtered = make([]int, len(m.models))
		for i := range m.models {
			m.filtered[i] = i
		}
	} else {
		isTagFilter := false
		var tagSearch string
		if strings.HasPrefix(query, "#") {
			isTagFilter = true
			tagSearch = strings.TrimPrefix(query, "#")
		} else if strings.HasPrefix(query, "tag:") {
			isTagFilter = true
			tagSearch = strings.TrimPrefix(query, "tag:")
		}

		m.filtered = []int{}
		for i, mod := range m.models {
			if isTagFilter {
				tags := m.config.ModelTags[mod.FilePath]
				match := false
				for _, t := range tags {
					if strings.Contains(strings.ToLower(t), tagSearch) {
						match = true
						break
					}
				}
				if match {
					m.filtered = append(m.filtered, i)
				}
			} else {
				if strings.Contains(strings.ToLower(mod.Name), query) ||
					strings.Contains(strings.ToLower(mod.Architecture), query) ||
					strings.Contains(strings.ToLower(mod.FilePath), query) {
					m.filtered = append(m.filtered, i)
				}
			}
		}
	}
	m.rebuildSidebar()
}

func (m *BrowserModel) saveLastSelected() {
	if m.selected >= 0 && m.selected < len(m.sidebarItems) {
		item := m.sidebarItems[m.selected]
		if item.Type == ItemModelEntry {
			m.config.LastSelectedModel = item.ModelPath
			_ = m.config.Save()
		}
	}
}

func (m *BrowserModel) rebuildSidebar() {
	m.sidebarItems = []SidebarItem{}

	modelPathMap := make(map[string]int)
	for idx, mod := range m.models {
		modelPathMap[mod.FilePath] = idx
	}

	query := strings.TrimSpace(m.searchInput.Value())
	if m.searchActive && query != "" {
		for _, idx := range m.filtered {
			mod := m.models[idx]
			m.sidebarItems = append(m.sidebarItems, SidebarItem{
				Type:      ItemModelEntry,
				Label:     mod.Name,
				ModelIdx:  idx,
				ModelPath: mod.FilePath,
			})
		}
		m.adjustSelection()
		return
	}

	// 1. Favorites
	var existingFavorites []SidebarItem
	for _, path := range m.config.Favorites {
		if idx, ok := modelPathMap[path]; ok {
			existingFavorites = append(existingFavorites, SidebarItem{
				Type:      ItemModelEntry,
				Label:     m.models[idx].Name,
				ModelIdx:  idx,
				ModelPath: path,
			})
		}
	}
	if len(existingFavorites) > 0 {
		m.sidebarItems = append(m.sidebarItems, SidebarItem{Type: ItemSectionHeader, Label: "★ FAVORITES"})
		m.sidebarItems = append(m.sidebarItems, existingFavorites...)
	}

	// 2. Recently Used
	var existingRecents []SidebarItem
	for _, path := range m.config.RecentLaunches {
		if idx, ok := modelPathMap[path]; ok {
			existingRecents = append(existingRecents, SidebarItem{
				Type:      ItemModelEntry,
				Label:     m.models[idx].Name,
				ModelIdx:  idx,
				ModelPath: path,
			})
		}
	}
	if len(existingRecents) > 0 {
		m.sidebarItems = append(m.sidebarItems, SidebarItem{Type: ItemSectionHeader, Label: "RECENTLY USED"})
		m.sidebarItems = append(m.sidebarItems, existingRecents...)
	}

	// 3. Collections
	var colNames []string
	for name := range m.config.Collections {
		colNames = append(colNames, name)
	}
	sort.Strings(colNames)

	if len(colNames) > 0 {
		m.sidebarItems = append(m.sidebarItems, SidebarItem{Type: ItemSectionHeader, Label: "COLLECTIONS"})
		for _, colName := range colNames {
			expanded := m.expandedCollections[colName]
			folderLabel := "▸ " + colName
			if expanded {
				folderLabel = "▾ " + colName
			}
			m.sidebarItems = append(m.sidebarItems, SidebarItem{
				Type:           ItemFolderHeader,
				Label:          folderLabel,
				CollectionName: colName,
				Expanded:       expanded,
			})
			if expanded {
				for _, path := range m.config.Collections[colName] {
					if idx, ok := modelPathMap[path]; ok {
						m.sidebarItems = append(m.sidebarItems, SidebarItem{
							Type:           ItemModelEntry,
							Label:          "  " + m.models[idx].Name,
							ModelIdx:       idx,
							ModelPath:      path,
							CollectionName: colName,
						})
					}
				}
			}
		}
	}

	// 4. All Models
	if len(m.models) > 0 {
		m.sidebarItems = append(m.sidebarItems, SidebarItem{Type: ItemSectionHeader, Label: "ALL MODELS"})
		for idx, mod := range m.models {
			m.sidebarItems = append(m.sidebarItems, SidebarItem{
				Type:      ItemModelEntry,
				Label:     mod.Name,
				ModelIdx:  idx,
				ModelPath: mod.FilePath,
			})
		}
	}

	m.adjustSelection()
}

func (m *BrowserModel) moveSelection(direction int) {
	if len(m.sidebarItems) == 0 {
		return
	}
	next := m.selected
	for {
		next += direction
		if next < 0 || next >= len(m.sidebarItems) {
			return
		}
		if m.sidebarItems[next].Type != ItemSectionHeader {
			m.selected = next
			m.saveLastSelected()
			return
		}
	}
}

func (m *BrowserModel) adjustSelection() {
	if len(m.sidebarItems) == 0 {
		m.selected = 0
		return
	}
	if m.selected >= len(m.sidebarItems) {
		m.selected = len(m.sidebarItems) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
	if m.sidebarItems[m.selected].Type == ItemSectionHeader {
		found := false
		for i := m.selected; i < len(m.sidebarItems); i++ {
			if m.sidebarItems[i].Type != ItemSectionHeader {
				m.selected = i
				found = true
				break
			}
		}
		if !found {
			for i := m.selected; i >= 0; i-- {
				if m.sidebarItems[i].Type != ItemSectionHeader {
					m.selected = i
					break
				}
			}
		}
	}
}

func (m *BrowserModel) rightPanelView(width int, height int) string {
	if len(m.sidebarItems) == 0 || m.selected < 0 || m.selected >= len(m.sidebarItems) {
		return "\n  No model selected."
	}
	item := m.sidebarItems[m.selected]
	if item.Type == ItemFolderHeader {
		var count int
		if list, ok := m.config.Collections[item.CollectionName]; ok {
			count = len(list)
		}
		return fmt.Sprintf("\n  %s\n\n  Collection: %s\n  Total Models: %d\n\n  Press [Enter/Space] to expand or collapse.\n  Select models under this folder to view details.",
			lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("FOLDER"),
			lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(item.CollectionName),
			count,
		)
	}
	if item.Type != ItemModelEntry {
		return "\n  No model selected."
	}
	selectedModel := m.models[item.ModelIdx]

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n  %s\n", lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(selectedModel.Name)))
	sb.WriteString(fmt.Sprintf("  %s\n\n", StyleHelp.Render(selectedModel.FilePath)))

	sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Architecture:", selectedModel.Architecture))
	sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Quantization:", selectedModel.Quantization))
	sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Context Length:", fmt.Sprintf("%d tokens", selectedModel.ContextLength)))
	sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Param Count:", formatParams(selectedModel.ParamCount)))
	sb.WriteString(fmt.Sprintf("  %-16s %s\n\n", "File Size:", formatSize(selectedModel.FileSize)))

	// Tags and Notes
	tags := m.config.ModelTags[selectedModel.FilePath]
	var tagsStr string
	if len(tags) > 0 {
		var formattedTags []string
		for _, tag := range tags {
			formattedTags = append(formattedTags, "["+tag+"]")
		}
		tagsStr = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(strings.Join(formattedTags, " "))
	} else {
		tagsStr = lipgloss.NewStyle().Foreground(ColorMuted).Render("(none) [press T to edit]")
	}
	sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Tags:", tagsStr))

	notes := m.config.GetNotes(selectedModel.FilePath)
	var notesStr string
	if notes != "" {
		lines := strings.Split(notes, "\n")
		var indentedLines []string
		for i, line := range lines {
			if i == 0 {
				indentedLines = append(indentedLines, line)
			} else {
				indentedLines = append(indentedLines, "                 "+line)
			}
		}
		notesStr = lipgloss.NewStyle().Foreground(ColorWhite).Render(strings.Join(indentedLines, "\n"))
	} else {
		notesStr = lipgloss.NewStyle().Foreground(ColorMuted).Render("(none) [press N to edit]")
	}
	sb.WriteString(fmt.Sprintf("  %-16s %s\n\n", "Notes:", notesStr))


	// Memory Estimates
	if m.hardwareSpecs != nil {
		est := hardware.EstimateMemory(selectedModel, m.hardwareSpecs, 0)
		var suitStr string
		switch est.Suitability {
		case hardware.SuitabilityFits:
			suitStr = StyleSuccess.Render("Fits Hardware")
		case hardware.SuitabilityPartial:
			suitStr = StyleWarning.Render("Partial Offloading Expected")
		case hardware.SuitabilityExceeds:
			suitStr = StyleDanger.Render("Exceeds Hardware limits")
		}

		sb.WriteString(fmt.Sprintf("  %s\n", lipgloss.NewStyle().Bold(true).Render("Memory Suitability:")))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Status:", suitStr))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "KV Cache:", formatSize(int64(est.KVCacheSize))))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Overhead:", formatSize(int64(est.Overhead))))
		sb.WriteString(fmt.Sprintf("  %-16s %s (GPU offload: %d%%)\n", "Total Memory:", formatSize(int64(est.TotalMemory)), est.GPUOffloadPct))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "Recommendation:", est.Reason))
		if est.Suitability == hardware.SuitabilityExceeds {
			sb.WriteString(fmt.Sprintf("                   %s %s\n",
				lipgloss.NewStyle().Foreground(ColorWarning).Bold(true).Render("Press [Enter]"),
				lipgloss.NewStyle().Foreground(ColorMuted).Render("to choose a profile with a smaller context length."),
			))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("  Detecting hardware requirements...\n\n")
	}

	sb.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Server Status:") + " ")
	statusText := ""
	switch m.serverUIStatus {
	case UIStatusStopped:
		statusText = lipgloss.NewStyle().Foreground(ColorMuted).Render("Stopped")
	case UIStatusStarting:
		statusText = lipgloss.NewStyle().Foreground(ColorAccent).Render("Starting...")
	case UIStatusRunning:
		statusText = lipgloss.NewStyle().Foreground(ColorSecondary).Render("Running on http://127.0.0.1:8080")
	case UIStatusFailed:
		statusText = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("Failed")
	}
	sb.WriteString(statusText + "\n")

	if m.serverUIStatus == UIStatusFailed && m.serverErr != nil {
		sb.WriteString(fmt.Sprintf("\n  %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(fmt.Sprintf("Error: %v", m.serverErr))))
	} else if m.serverUIStatus == UIStatusRunning && m.runningModelPath == selectedModel.FilePath {
		sb.WriteString("\n  Active model is currently serving requests.\n")
	}
	sb.WriteString("\n")

	// System specifications
	if m.hardwareSpecs != nil {
		gpuInfo := fmt.Sprintf("%s (%s VRAM)", m.hardwareSpecs.GPU.Name, formatSize(int64(m.hardwareSpecs.GPU.VRAM)))
		if m.hardwareSpecs.IsUnified {
			gpuInfo = fmt.Sprintf("%s (%s Unified)", m.hardwareSpecs.GPU.Name, formatSize(int64(m.hardwareSpecs.GPU.VRAM)))
		}
		sb.WriteString(fmt.Sprintf("  %s\n", lipgloss.NewStyle().Bold(true).Render("System Specifications:")))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "OS:", m.hardwareSpecs.OS))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "CPU Model:", m.hardwareSpecs.CPU.Model))
		sb.WriteString(fmt.Sprintf("  %-16s %d threads\n", "CPU Threads:", m.hardwareSpecs.CPU.Threads))
		sb.WriteString(fmt.Sprintf("  %-16s %s (%s available)\n", "RAM:", formatSize(int64(m.hardwareSpecs.RAM.Total)), formatSize(int64(m.hardwareSpecs.RAM.Available))))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", "GPU:", gpuInfo))
	}

	return sb.String()
}

func (m *BrowserModel) View() string {
	if m.loading {
		return "\n  Scanning models directory... Please wait."
	}

	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n  Press Q to quit.", m.err)
	}

	if m.screenMode == ScreenDashboard && m.dashboard != nil {
		dashView := m.dashboard.View(m.width, m.height)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dashView)
	}

	if m.screenMode == ScreenCollections && m.collectionsModel != nil {
		colView := m.collectionsModel.View(m.width, m.height)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, colView)
	}

	if (m.screenMode == ScreenTagsEditor || m.screenMode == ScreenNotesEditor) && m.editorModel != nil {
		editorView := m.editorModel.View(m.width, m.height)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, editorView)
	}

	if m.screenMode == ScreenBenchmarkProgress && m.benchmarkProgress != nil {
		progressView := m.benchmarkProgress.View(m.width, m.height)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, progressView)
	}

	if m.screenMode == ScreenPerformanceDashboard && m.perfDashboard != nil {
		perfView := m.perfDashboard.View(m.width, m.height)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, perfView)
	}

	if m.screenMode == ScreenServerMonitor && m.monitorModel != nil {
		monitorView := m.monitorModel.View(m.width, m.height)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, monitorView)
	}

	if m.screenMode == ScreenSettings && m.lifecycleModel != nil {
		settingsView := m.lifecycleModel.View(m.width, m.height)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, settingsView)
	}

	if m.screenMode == ScreenDownloader && m.downloaderModel != nil {
		downView := m.downloaderModel.View(m.width, m.height)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, downView)
	}

	totalWidth := m.width
	if totalWidth < 60 {
		totalWidth = 60
	}
	leftWidth := int(float64(totalWidth) * 0.40)
	if leftWidth < 25 {
		leftWidth = 25
	}
	rightWidth := totalWidth - leftWidth - 6

	panelHeight := m.height - 6
	if panelHeight < 10 {
		panelHeight = 10
	}

	var leftSb strings.Builder
	if len(m.sidebarItems) == 0 {
		leftSb.WriteString("\n  No models found.")
	} else {
		maxVisible := panelHeight - 2
		if maxVisible < 1 {
			maxVisible = 1
		}
		if m.selected < m.scrollOffset {
			m.scrollOffset = m.selected
		} else if m.selected >= m.scrollOffset+maxVisible {
			m.scrollOffset = m.selected - maxVisible + 1
		}

		end := m.scrollOffset + maxVisible
		if end > len(m.sidebarItems) {
			end = len(m.sidebarItems)
		}

		leftSb.WriteString("\n")
		for idx := m.scrollOffset; idx < end; idx++ {
			item := m.sidebarItems[idx]

			if item.Type == ItemSectionHeader {
				leftSb.WriteString(lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(fmt.Sprintf(" %s", item.Label)) + "\n")
				continue
			}

			if item.Type == ItemFolderHeader {
				folderLabel := item.Label
				if idx == m.selected {
					leftSb.WriteString(
						StyleSelectedListItem.Width(leftWidth - 2).Render(
							fmt.Sprintf("  %s", folderLabel),
						) + "\n",
					)
				} else {
					leftSb.WriteString(
						fmt.Sprintf("  %s\n", lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(folderLabel)),
					)
				}
				continue
			}

			// Model item entry
			mod := m.models[item.ModelIdx]

			bullet := "●"
			var bulletStyled string
			if m.hardwareSpecs != nil {
				est := hardware.EstimateMemory(mod, m.hardwareSpecs, 0)
				switch est.Suitability {
				case hardware.SuitabilityFits:
					bulletStyled = StyleSuccess.Render(bullet)
				case hardware.SuitabilityPartial:
					bulletStyled = StyleWarning.Render(bullet)
				case hardware.SuitabilityExceeds:
					bulletStyled = StyleDanger.Render(bullet)
				}
			} else {
				bulletStyled = StyleListItem.Render(bullet)
			}

			isRunningStr := ""
			if m.serverUIStatus == UIStatusRunning && m.runningModelPath == mod.FilePath {
				isRunningStr = "▶ "
			}

			displayName := item.Label
			if m.config.IsFavorite(mod.FilePath) {
				starSymbol := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB800")).Render("★ ")
				// Strip leading indent from nested elements to format properly
				if strings.HasPrefix(displayName, "  ") {
					displayName = "  " + starSymbol + strings.TrimPrefix(displayName, "  ")
				} else {
					displayName = starSymbol + displayName
				}
			}

			maxNameLen := leftWidth - 8
			if maxNameLen > 0 && len(displayName) > maxNameLen {
				displayName = displayName[:maxNameLen-3] + "..."
			}

			if idx == m.selected {
				leftSb.WriteString(
					StyleSelectedListItem.Width(leftWidth - 2).Render(
						fmt.Sprintf("%s%s %s", isRunningStr, bullet, displayName),
					) + "\n",
				)
			} else {
				leftSb.WriteString(
					fmt.Sprintf("  %s%s %s\n", isRunningStr, bulletStyled, StyleListItem.Render(displayName)),
				)
			}
		}
	}

	leftPanelContent := fmt.Sprintf("%s\n%s", StyleTitle.Render("Models"), leftSb.String())
	leftView := StyleLeftPanel.
		Width(leftWidth).
		Height(panelHeight).
		Render(leftPanelContent)

	rightPanelContent := fmt.Sprintf("%s\n%s", StyleTitle.Render("Details"), m.rightPanelView(rightWidth, panelHeight))
	rightView := StyleRightPanel.
		Width(rightWidth).
		Height(panelHeight).
		Render(rightPanelContent)

	mainView := lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView)

	header := StyleHeader.Render("LLAMA MANAGER — Local AI Control Center")

	var footer string
	if m.searchActive {
		footer = fmt.Sprintf("Search: %s  %s", m.searchInput.View(), StyleHelp.Render("[Esc] Clear/Exit  [Enter] Confirm"))
	} else {
		footer = fmt.Sprintf("%s Launch  %s Favorite  %s Collections  %s Tags  %s Notes  %s Benchmark  %s Dashboard  %s Monitor  %s Settings  %s Downloader  %s Search  %s Stop  %s Quit",
			StyleHelpKey.Render("[Enter]"),
			StyleHelpKey.Render("[F]"),
			StyleHelpKey.Render("[C]"),
			StyleHelpKey.Render("[T]"),
			StyleHelpKey.Render("[N]"),
			StyleHelpKey.Render("[B]"),
			StyleHelpKey.Render("[V]"),
			StyleHelpKey.Render("[M]"),
			StyleHelpKey.Render("[U]"),
			StyleHelpKey.Render("[D]"),
			StyleHelpKey.Render("[/]"),
			StyleHelpKey.Render("[S]"),
			StyleHelpKey.Render("[Q]"),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, mainView, StyleHelp.Render(footer))
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	suffix := ""
	switch exp {
	case 0:
		suffix = "KB"
	case 1:
		suffix = "MB"
	case 2:
		suffix = "GB"
	default:
		suffix = "TB"
	}
	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), suffix)
}

func formatParams(params uint64) string {
	if params == 0 {
		return "Unknown"
	}
	if params >= 1e9 {
		return fmt.Sprintf("%.2f B", float64(params)/1e9)
	}
	if params >= 1e6 {
		return fmt.Sprintf("%.2f M", float64(params)/1e6)
	}
	return fmt.Sprintf("%d", params)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
