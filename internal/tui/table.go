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

	// Right-align numeric columns (PODS=3, CPU REQ=4, MEM REQ=5, $/HR=6, COST=7).
	numericStyle = lipgloss.NewStyle().Padding(0, 1).Align(lipgloss.Right)
)

// sortIndicator returns the header text with a sort direction arrow if this
// column is the active sort column.
func sortIndicator(header string, col SortColumn, cfg SortConfig) string {
	if col != cfg.Column {
		return header
	}
	if cfg.Asc {
		return header + " ^"
	}
	return header + " v"
}

// RenderTable renders the aggregated costs as a formatted table string.
// When showSubtype is true, a SUBTYPE column is included.
// The sortCfg controls which column header receives a sort indicator arrow.
func RenderTable(aggs []cost.AggregatedCost, showSubtype bool, sortCfg SortConfig) string {
	rows := make([][]string, 0, len(aggs)+1)

	var totalCostPerHour, totalCost float64
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
		row = append(row,
			fmt.Sprintf("%d", a.PodCount),
			fmt.Sprintf("%.2f", a.TotalCPUVCPU),
			fmt.Sprintf("%.1f GB", a.TotalMemGB),
			fmt.Sprintf("$%.4f", a.CostPerHour),
			fmt.Sprintf("$%.4f", a.TotalCost),
			spot,
		)
		rows = append(rows, row)
		totalCostPerHour += a.CostPerHour
		totalCost += a.TotalCost
	}

	// Total row
	totalRow := []string{"TOTAL", ""}
	if showSubtype {
		totalRow = append(totalRow, "")
	}
	totalRow = append(totalRow, "", "", "",
		fmt.Sprintf("$%.4f", totalCostPerHour),
		fmt.Sprintf("$%.4f", totalCost),
		"",
	)
	rows = append(rows, totalRow)

	headers := []string{
		sortIndicator("TEAM", SortByTeam, sortCfg),
		sortIndicator("WORKLOAD", SortByWorkload, sortCfg),
	}
	if showSubtype {
		headers = append(headers, sortIndicator("SUBTYPE", SortBySubtype, sortCfg))
	}
	headers = append(headers,
		sortIndicator("PODS", SortByPods, sortCfg),
		sortIndicator("CPU REQ", SortByCPU, sortCfg),
		sortIndicator("MEM REQ", SortByMem, sortCfg),
		sortIndicator("$/HR", SortByCostPerHour, sortCfg),
		sortIndicator("COST", SortByCost, sortCfg),
		"SPOT",
	)

	// First numeric column index depends on whether SUBTYPE is shown.
	numericStart := 2
	if showSubtype {
		numericStart = 3
	}
	numericEnd := numericStart + 4 // PODS, CPU REQ, MEM REQ, $/HR, COST

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderRow(false).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			if col >= numericStart && col <= numericEnd {
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
