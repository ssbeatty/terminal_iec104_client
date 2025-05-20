package ui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"iec104/config"
	"iec104/iec_client"
	"math"
	"strconv"
	"sync/atomic"
)

// App represents the main application UI
type App struct {
	app           *tview.Application
	config        *config.Config
	iecClient     *iec_client.IEC104Client
	logger        *Logger
	pages         *tview.Pages
	operationForm *tview.Form
	dataTable     *tview.Table
	logView       *tview.TextView
	tabBar        *tview.TextView
	currentTab    iec_client.DataType
	statusBar     *tview.TextView

	started atomic.Bool
}

// NewApp creates a new application UI
func NewApp(cfg *config.Config) *App {
	app := &App{
		app:        tview.NewApplication(),
		config:     cfg,
		iecClient:  iec_client.NewIEC104Client(cfg),
		currentTab: iec_client.Telemetry,
	}

	// Initialize UI components
	app.setupUI()

	return app
}

// setupUI initializes all UI components
func (a *App) setupUI() {
	// Create pages for main view and dialogs
	a.pages = tview.NewPages()

	// Setup log view
	a.setupLogView()

	// Create logger
	a.logger = NewLogger(a.logView, LoggerLevelInfo)
	a.logger.Infof("Application started")

	// Setup config form
	a.setupConfigForm()

	// Setup data table
	a.setupDataTable()

	// Setup tab bar
	a.setupTabBar()

	// Setup status bar
	a.setupStatusBar()

	// Create main layout
	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow).
		AddItem(a.operationForm, 5, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(a.tabBar, 1, 1, false).
			AddItem(a.dataTable, 0, 10, true).
			AddItem(a.logView, 5, 1, false).
			AddItem(a.statusBar, 1, 1, false),
			0, 8, false)

	// Add main layout to pages
	a.pages.AddPage("main", flex, true, true)

	// Set up key bindings
	a.setupKeyBindings()

	a.iecClient.RegisterConnectionStateHandler(func(b bool) {
		a.app.QueueUpdateDraw(func() {
			a.updateStatusBar()
		})
	})

	a.iecClient.RegisterDataHandler(func(typ iec_client.DataType, iot int, data interface{}) {
		if typ != a.currentTab {
			return
		}

		var (
			rowMax  int
			address int
		)

		switch typ {
		case iec_client.Telemetry:
			rowMax = int(math.Ceil(float64(a.config.TelemetryCount) / 10))
			address = iot - 0x4000 - 1
			if address > a.config.TelemetryCount {
				return
			}
		case iec_client.Teleindication:
			rowMax = int(math.Ceil(float64(a.config.TeleindCount) / 10))
			address = iot - 1
			if address > a.config.TeleindCount {
				return
			}
		default:
			return
		}
		if address < 0 {
			a.logger.Errorf("Invalid address: %d", address)
			return
		}

		row := (address/10 + 1) * 2
		col := address%10 + 1

		if row > rowMax*2 {
			return
		}

		a.app.QueueUpdateDraw(func() {
			switch val := data.(type) {
			case float64:
				a.dataTable.SetCell(row, col, tview.NewTableCell(fmt.Sprintf("%.2f", val)))
			case bool:
				if val {
					a.dataTable.SetCell(row, col, tview.NewTableCell("ON"))
				} else {
					a.dataTable.SetCell(row, col, tview.NewTableCell("OFF"))
				}
			}
		})

	})
	a.iecClient.Logger = a.logger
}

// setupConfigForm creates the configuration form
func (a *App) setupConfigForm() {
	a.operationForm = tview.NewForm()
	a.operationForm.SetBorder(true).SetTitle("Options")

	// Add buttons
	a.operationForm.AddButton("Edit Config", func() {
		a.showConfigDialog()
	})

	a.operationForm.AddButton("Start", func() {
		a.toggleConnection()
	})

}

// setupDataTable creates the data table
func (a *App) setupDataTable() {
	a.dataTable = tview.NewTable().SetBorders(true)
	a.dataTable.SetBorder(true).SetTitle("Data")

	// Make headers fixed so they don't disappear when scrolling
	a.dataTable.SetFixed(1, 0)
	a.dataTable.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			a.dataTable.SetSelectable(true, true)
		}
	})

	// Initialize table headers
	a.updateTableHeaders()

	// Initialize table data
	a.updateTableData()

	// Also keep the selected func for double-clicks
	a.dataTable.SetSelectedFunc(func(row, column int) {
		if row == 0 {
			// Header row, do nothing
			return
		}

		// Handle cell selection based on current tab
		switch a.currentTab {
		case iec_client.Telecontrol:
			// Only respond to clicks on the action column (column 3)
			a.logger.Infof("Selected Telecontrol row %d, column %d", row-1, column-1)
			a.showTelecontrolDialog(row, column)
		case iec_client.Teleregulation:
			// Only respond to clicks on the action column (column 4)
			a.logger.Infof("Selected Telecontrol row %d, column %d", row-1, column-1)
			a.showTeleregulationDialog(row, column)
		case iec_client.Telemetry, iec_client.Teleindication:
			// Only respond to clicks on the action column (column 0)
			a.logger.Infof("Selected %s row %d, column %d", a.currentTab, row-1, column-1)
			a.showDescriptionDialog(row, column)

		}

		a.dataTable.SetSelectable(false, false)
	})
}

// setupTabBar creates the tab bar for switching between data types
func (a *App) setupTabBar() {
	a.tabBar = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWrap(false)

	a.updateTabBar()
}

// setupLogView creates the log view
func (a *App) setupLogView() {
	a.logView = tview.NewTextView().
		SetDynamicColors(true).
		SetChangedFunc(func() {
			a.app.Draw()
		})
	a.logView.SetBorder(true).SetTitle("Logs")
}

// setupStatusBar creates the status bar
func (a *App) setupStatusBar() {
	a.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	a.updateStatusBar()
}

// setupKeyBindings sets up global key bindings
func (a *App) setupKeyBindings() {
	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle tab switching
		if event.Key() == tcell.KeyF1 {
			a.switchTab(iec_client.Telemetry)
			return nil
		} else if event.Key() == tcell.KeyF2 {
			a.switchTab(iec_client.Teleindication)
			return nil
		} else if event.Key() == tcell.KeyF3 {
			a.switchTab(iec_client.Telecontrol)
			return nil
		} else if event.Key() == tcell.KeyF4 {
			a.switchTab(iec_client.Teleregulation)
			return nil
		} else if event.Key() == tcell.KeyEscape {
			a.iecClient.Close()
			a.app.Stop()
			return nil
		}
		return event
	})
}

// updateTableHeaders updates the table headers based on the current tab
func (a *App) updateTableHeaders() {
	a.dataTable.Clear()

	for col := 0; col < 10; col++ {
		a.dataTable.SetCell(0, col+1, tview.NewTableCell(fmt.Sprintf("%-10d", col)).SetAlign(tview.AlignCenter).SetSelectable(false).SetTextColor(tcell.ColorYellow))
	}
}

// updateTableData updates the table data based on the current tab
func (a *App) updateTableData() {
	a.logger.Debugf("Config: %+v", a.config)
	// Clear existing data rows but keep headers
	for row := 1; row < a.dataTable.GetRowCount(); row++ {
		for col := 0; col < a.dataTable.GetColumnCount(); col++ {
			a.dataTable.SetCell(row, col, tview.NewTableCell(""))
		}
	}

	// Populate data based on current tab
	switch a.currentTab {
	case iec_client.Telemetry:
		rowMax := int(math.Ceil(float64(a.config.TelemetryCount) / 10))
		for row := 0; row < rowMax; row++ {
			a.dataTable.SetCell((row)*2+1, 0, tview.NewTableCell(strconv.Itoa(row*10)).SetSelectable(false))
			a.dataTable.SetCell((row+1)*2, 0, tview.NewTableCell(strconv.Itoa(row*10)).SetSelectable(false))
		}
		for index, desc := range a.config.TelemetryDescriptions {
			row := (index/10+1)*2 - 1
			col := index%10 + 1
			if row > rowMax*2 {
				continue
			}
			if row < 1 {
				a.logger.Errorf("Invalid telemetry row: %d", row)
				continue
			}
			a.dataTable.SetCell(row, col, tview.NewTableCell(desc).SetTextColor(tcell.ColorGreen).SetSelectable(false))
		}
		for address, point := range a.iecClient.Telemetry {
			address = address - 0x4000 - 1
			if address < 0 {
				a.logger.Errorf("Invalid telemetry address: %d", address)
				continue
			}
			if address >= a.config.TelemetryCount {
				continue
			}
			row := (address/10 + 1) * 2
			col := address%10 + 1

			if row > rowMax*2 {
				continue
			}

			a.dataTable.SetCell(row, col, tview.NewTableCell(fmt.Sprintf("%.2f", point.Value)))
		}
	case iec_client.Teleindication:
		rowMax := int(math.Ceil(float64(a.config.TeleindCount) / 10))
		for row := 0; row < rowMax; row++ {
			a.dataTable.SetCell((row)*2+1, 0, tview.NewTableCell(strconv.Itoa(row*10)).SetSelectable(false))
			a.dataTable.SetCell((row+1)*2, 0, tview.NewTableCell(strconv.Itoa(row*10)).SetSelectable(false))
		}
		for index, desc := range a.config.TeleindDescriptions {
			row := (index/10+1)*2 - 1
			col := index%10 + 1
			if row > rowMax*2 {
				continue
			}
			if row < 1 {
				a.logger.Errorf("Invalid teleindication row: %d", row)
				continue
			}
			a.dataTable.SetCell(row, col, tview.NewTableCell(desc).SetTextColor(tcell.ColorGreen).SetSelectable(false))
		}
		for address, point := range a.iecClient.Teleindication {
			address = address - 1
			if address < 0 {
				a.logger.Errorf("Invalid teleindication address: %d", address)
				continue
			}
			if address >= a.config.TeleindCount {
				continue
			}

			row := (address/10 + 1) * 2
			col := address%10 + 1

			if row > rowMax*2 {
				continue
			}

			val := "OFF"
			if point.Value {
				val = "ON"
			}
			a.dataTable.SetCell(row, col, tview.NewTableCell(val))
		}
	case iec_client.Telecontrol:
		rowMax := 10
		// Add sample telecontrol points or actual ones
		for row := 0; row < rowMax+1; row++ {
			a.dataTable.SetCell(row+1, 0, tview.NewTableCell(strconv.Itoa(row*10)).SetSelectable(false))
			for col := 0; col < 11; col++ {
				if row == 0 || col == 0 {
					continue
				}
				index := (row-1)*10 + col - 1
				if v, ok := a.iecClient.Telecontrol[index]; ok {
					val := "OFF"
					if v.Value {
						val = "ON"
					}

					a.dataTable.SetCell(row, col, tview.NewTableCell(val))
				} else {
					a.dataTable.SetCell(row, col, tview.NewTableCell("OFF"))
				}
			}
		}
	case iec_client.Teleregulation:
		// Add sample teleregulation points or actual ones
		rowMax := 10
		for row := 0; row < rowMax+1; row++ {
			a.dataTable.SetCell(row+1, 0, tview.NewTableCell(strconv.Itoa(row*10)).SetSelectable(false))
			for col := 0; col < 11; col++ {
				if row == 0 || col == 0 {
					continue
				}
				index := (row-1)*10 + col - 1
				if v, ok := a.iecClient.Teleregulation[index]; ok {
					a.dataTable.SetCell(row, col, tview.NewTableCell(fmt.Sprintf("%.2f", v.Value)))
				} else {
					a.dataTable.SetCell(row, col, tview.NewTableCell("0.00"))
				}
			}
		}
	}
}

// updateTabBar updates the tab bar based on the current tab
func (a *App) updateTabBar() {
	a.tabBar.Clear()
	fmt.Fprintf(a.tabBar, "%s F1 Telemetry %s | %s F2 Teleindication %s | %s F3 Telecontrol %s | %s F4 Teleregulation %s",
		getTabHighlight(a.currentTab == iec_client.Telemetry),
		getTabHighlight(false),
		getTabHighlight(a.currentTab == iec_client.Teleindication),
		getTabHighlight(false),
		getTabHighlight(a.currentTab == iec_client.Telecontrol),
		getTabHighlight(false),
		getTabHighlight(a.currentTab == iec_client.Teleregulation),
		getTabHighlight(false))
}

// updateStatusBar updates the status bar
func (a *App) updateStatusBar() {
	status := "Disconnected"
	color := "red"
	if a.iecClient.Connected.Load() {
		status = "Connected"
		color = "green"
	}
	a.statusBar.Clear()
	fmt.Fprintf(a.statusBar, "Status: [%s]%s[white] | Server: %s:%d | Common Address: %d",
		color, status, a.config.IPAddress, a.config.Port, a.config.CommonAddress)
}

// switchTab switches to the specified data type tab
func (a *App) switchTab(tab iec_client.DataType) {
	a.currentTab = tab
	a.updateTabBar()
	a.updateTableHeaders()
	a.updateTableData()
	a.logger.Infof("Switched to %s tab", getTabName(tab))
}

// saveConfig saves the current configuration
func (a *App) saveConfig() {
	err := a.config.Save()
	if err != nil {
		a.logger.Infof("Error saving configuration: %v", err)
		return
	}

	a.iecClient.UpdateConfig(a.config)

	a.logger.Infof("Configuration saved successfully")
}

// toggleConnection toggles the connection state
func (a *App) toggleConnection() {
	if a.started.Load() {
		err := a.iecClient.Disconnect()
		if err != nil {
			a.logger.Infof("Error disconnecting: %v", err)
			return
		}
		a.logger.Infof("Disconnected from server")
		a.started.Store(false)
	} else {
		err := a.iecClient.Connect()
		if err != nil {
			a.logger.Infof("Error connecting: %v", err)
			return
		}
		a.logger.Infof("Connecting to server %s:%d", a.config.IPAddress, a.config.Port)
		a.started.Store(true)
	}

	a.updateConnectButton()
}

func (a *App) updateConnectButton() {
	// Update the connect button text
	buttonText := "Start"
	if a.started.Load() {
		buttonText = "Stop"
	}
	a.operationForm.GetButton(1).SetLabel(buttonText)
}

func (a *App) showConfigDialog() {
	// Create form for telecontrol
	form := tview.NewForm()
	form.SetBorder(true).SetTitle("Config Settings")

	// Add form fields
	form.AddInputField("IP Address", a.config.IPAddress, 20, nil, func(text string) {
		a.config.IPAddress = text
	})
	form.AddInputField("Port", fmt.Sprintf("%d", a.config.Port), 10, nil, func(text string) {
		var port int
		fmt.Sscanf(text, "%d", &port)
		a.config.Port = port
	})
	form.AddInputField("Common Address", fmt.Sprintf("%d", a.config.CommonAddress), 10, nil, func(text string) {
		var ca int
		fmt.Sscanf(text, "%d", &ca)
		a.config.CommonAddress = ca
	})
	form.AddInputField("Telemetry Count", fmt.Sprintf("%d", a.config.TelemetryCount), 10, nil, func(text string) {
		var tc int
		fmt.Sscanf(text, "%d", &tc)
		a.config.TelemetryCount = tc
	})
	form.AddInputField("Teleindication Count", fmt.Sprintf("%d", a.config.TeleindCount), 10, nil, func(text string) {
		var tic int
		fmt.Sscanf(text, "%d", &tic)
		a.config.TeleindCount = tic
	})
	form.AddInputField("Interrogation Interval (s)", fmt.Sprintf("%d", a.config.InterrogationInterval), 10, nil, func(text string) {
		var ii int
		fmt.Sscanf(text, "%d", &ii)
		a.config.InterrogationInterval = ii
	})

	// Add buttons
	form.AddButton("Save", func() {
		a.saveConfig()
		a.pages.RemovePage("dialog")
	})
	form.AddButton("Cancel", func() {
		a.pages.RemovePage("dialog")
	})

	// Create a modal for the form
	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 60, 1, true).
			AddItem(nil, 0, 1, false),
			20, 1, true).
		AddItem(nil, 0, 1, false)

	// Add the page and show it
	a.pages.AddPage("dialog", modal, true, true)
}

// showTelecontrolDialog shows a dialog for sending telecontrol commands
func (a *App) showTelecontrolDialog(row, col int) {
	index := (row-1)*10 + col - 1
	// Create form for telecontrol
	form := tview.NewForm()
	form.SetBorder(true).SetTitle("Send Telecontrol Command")

	// Add address field (read-only)
	form.AddInputField("Offset", fmt.Sprintf("%d", index), 10, nil, nil).
		SetFieldBackgroundColor(tcell.ColorDarkGray)

	// Add value field
	value := false
	form.AddCheckbox("Value", value, func(checked bool) {
		value = checked
	})

	// Add buttons
	form.AddButton("Send", func() {
		err := a.iecClient.SendTelecontrol(index, value)
		if err != nil {
			a.logger.Infof("Error sending telecontrol: %v", err)
		} else {
			a.logger.Infof("Telecontrol command sent to address %d, value: %v", index, value)

			v := "OFF"
			if value {
				v = "ON"
			}
			a.dataTable.SetCell(row, col, tview.NewTableCell(v))

			a.iecClient.Telecontrol[index] = iec_client.TelecontrolPoint{
				Value: value,
			}
		}
		a.pages.RemovePage("dialog")
	})
	form.AddButton("Cancel", func() {
		a.pages.RemovePage("dialog")
	})

	// Create a modal for the form
	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 40, 1, true).
			AddItem(nil, 0, 1, false),
			10, 1, true).
		AddItem(nil, 0, 1, false)

	// Add the page and show it
	a.pages.AddPage("dialog", modal, true, true)
}

// showTeleregulationDialog shows a dialog for sending teleregulation setpoints
func (a *App) showTeleregulationDialog(row, col int) {
	index := (row-1)*10 + col - 1
	// Create form for teleregulation
	form := tview.NewForm()
	form.SetBorder(true).SetTitle("Send Teleregulation Setpoint")

	// Add address field (read-only)
	form.AddInputField("Offset", fmt.Sprintf("%d", index), 10, nil, nil).
		SetFieldBackgroundColor(tcell.ColorDarkGray)

	// Add value field
	valueStr := "0.0"
	form.AddInputField("Value", valueStr, 10, nil, func(text string) {
		valueStr = text
	})

	// Add buttons
	form.AddButton("Send", func() {
		var value float64
		fmt.Sscanf(valueStr, "%f", &value)
		err := a.iecClient.SendTelemetry(index, value)
		if err != nil {
			a.logger.Infof("Error sending teleregulation: %v", err)
		} else {
			a.logger.Infof("Teleregulation setpoint sent to address %d, value: %v", index, value)
			a.dataTable.SetCell(row, col, tview.NewTableCell(fmt.Sprintf("%.2f", value)))

			a.iecClient.Teleregulation[index] = iec_client.TeleregulationPoint{
				Value: value,
			}
		}
		a.pages.RemovePage("dialog")
	})
	form.AddButton("Cancel", func() {
		a.pages.RemovePage("dialog")
	})

	// Create a modal for the form
	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 40, 1, true).
			AddItem(nil, 0, 1, false),
			10, 1, true).
		AddItem(nil, 0, 1, false)

	// Add the page and show it
	a.pages.AddPage("dialog", modal, true, true)
}

// showDescriptionDialog shows a dialog for editing point descriptions
func (a *App) showDescriptionDialog(row, col int) {
	index := (row-1)/2*10 + col - 1

	var currentDesc string
	switch a.currentTab {
	case iec_client.Telemetry:
		currentDesc = a.config.TelemetryDescriptions[index]
	case iec_client.Teleindication:
		currentDesc = a.config.TeleindDescriptions[index]
	}

	// Create form for description
	form := tview.NewForm()
	form.SetBorder(true).SetTitle("Edit Point Description")

	// Add address field (read-only)
	form.AddInputField("Offset", fmt.Sprintf("%d", index), 10, nil, nil).
		SetFieldBackgroundColor(tcell.ColorDarkGray)

	// Add description field
	form.AddInputField("Description", currentDesc, 40, nil, func(text string) {
		currentDesc = text
	})

	// Add buttons
	form.AddButton("Save", func() {
		// 保存描述
		switch a.currentTab {
		case iec_client.Telemetry:
			a.config.TelemetryDescriptions[index] = currentDesc
		case iec_client.Teleindication:
			a.config.TeleindDescriptions[index] = currentDesc
		}

		// 保存配置
		if err := a.config.Save(); err != nil {
			a.logger.Errorf("Error saving description: %v", err)
		} else {
			a.logger.Infof("Description saved for offset %d", index)
			if row-1 < 1 {
				a.logger.Errorf("Invalid offset %d", index)
				return
			}
			a.dataTable.SetCell(row-1, col, tview.NewTableCell(currentDesc).SetTextColor(tcell.ColorGreen).SetSelectable(false))
		}
		a.pages.RemovePage("dialog")
	})
	form.AddButton("Cancel", func() {
		a.pages.RemovePage("dialog")
	})

	// Create a modal for the form
	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 40, 1, true).
			AddItem(nil, 0, 1, false),
			10, 1, true).
		AddItem(nil, 0, 1, false)

	// Add the page and show it
	a.pages.AddPage("dialog", modal, true, true)
}

// Run starts the application
func (a *App) Run() error {
	return a.app.SetRoot(a.pages, true).EnableMouse(true).Run()
}

// Helper functions

// getTabHighlight returns the highlight formatting for a tab
func getTabHighlight(active bool) string {
	if active {
		return "[black:white]"
	}
	return "[white:black]"
}

// getTabName returns the name of a tab
func getTabName(tab iec_client.DataType) string {
	switch tab {
	case iec_client.Telemetry:
		return "Telemetry"
	case iec_client.Teleindication:
		return "Teleindication"
	case iec_client.Telecontrol:
		return "Telecontrol"
	case iec_client.Teleregulation:
		return "Teleregulation"
	default:
		return "Unknown"
	}
}
