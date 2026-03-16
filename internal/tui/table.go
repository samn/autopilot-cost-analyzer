package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/samn/autopilot-cost-analyzer/internal/cost"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	cellStyle   = lipgloss.NewStyle().Padding(0, 1)

	// Right-align numeric columns.
	numericStyle = lipgloss.NewStyle().Padding(0, 1).Align(lipgloss.Right)
)

// sortIndicator returns the header text with a sort direction arrow if this
// column is the active sort column.
func sortIndicator(header string, col SortColumn, sortable bool, cfg SortConfig) string {
	if !sortable || col != cfg.Column {
		return header
	}
	if cfg.Asc {
		return header + " ^"
	}
	return header + " v"
}

// costModeShort returns a short display string for the cost mode.
func costModeShort(mode string) string {
	switch mode {
	case "autopilot":
		return "AP"
	case "standard":
		return "STD"
	default:
		return mode
	}
}

// RenderTable renders the aggregated costs as a formatted table string.
// When showSubtype is true, a SUBTYPE column is included.
// When showUtilization is true, CPU%, MEM%, and WASTE columns are included.
// When showMode is true, a MODE column is included.
// The sortCfg controls which column header receives a sort indicator arrow.
func RenderTable(aggs []cost.AggregatedCost, showSubtype, showUtilization, showMode bool, sortCfg SortConfig) string {
	vis := ColumnVisibility{Subtype: showSubtype, Mode: showMode, Utilization: showUtilization}
	defs := visibleColumns(vis)

	rows := make([][]string, 0, len(aggs)+1)

	var totalCostPerHour, totalCost, totalWaste float64
	for _, a := range aggs {
		spot := ""
		if a.Key.IsSpot {
			spot = "yes"
		}
		row := []string{
			orDefault(a.Key.Team, "-"),
			orDefault(a.Key.Workload, "-"),
		}
		if showSubtype {
			row = append(row, orDefault(a.Key.Subtype, "-"))
		}
		if showMode {
			row = append(row, costModeShort(a.CostMode))
		}
		row = append(row,
			fmt.Sprintf("%d", a.PodCount),
			fmt.Sprintf("%.2f", a.TotalCPUVCPU),
			fmt.Sprintf("%.1f GB", a.TotalMemGB),
			fmt.Sprintf("$%.4f", a.CostPerHour),
			fmt.Sprintf("$%.4f", a.TotalCost),
			spot,
		)
		if showUtilization {
			if a.HasUtilization {
				row = append(row,
					fmt.Sprintf("%.0f%%", a.CPUUtilization*100),
					fmt.Sprintf("%.0f%%", a.MemUtilization*100),
					fmt.Sprintf("$%.4f", a.WastedCostPerHour),
				)
			} else {
				row = append(row, "-", "-", "-")
			}
		}
		rows = append(rows, row)
		totalCostPerHour += a.CostPerHour
		totalCost += a.TotalCost
		totalWaste += a.WastedCostPerHour
	}

	// Total row
	totalRow := []string{"TOTAL", ""}
	if showSubtype {
		totalRow = append(totalRow, "")
	}
	if showMode {
		totalRow = append(totalRow, "")
	}
	totalRow = append(totalRow, "", "", "",
		fmt.Sprintf("$%.4f", totalCostPerHour),
		fmt.Sprintf("$%.4f", totalCost),
		"",
	)
	if showUtilization {
		totalRow = append(totalRow, "", "",
			fmt.Sprintf("$%.4f", totalWaste),
		)
	}
	rows = append(rows, totalRow)

	// Build headers from column definitions.
	headers := make([]string, len(defs))
	for i, d := range defs {
		headers[i] = sortIndicator(d.header, d.sortCol, d.sortable, sortCfg)
	}

	// Build a set of numeric column indices from definitions.
	numericCols := make(map[int]bool)
	for i, d := range defs {
		if d.numeric {
			numericCols[i] = true
		}
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderRow(false).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			if numericCols[col] {
				return numericStyle
			}
			return cellStyle
		})

	return t.String()
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
