package util_tab_table

import (
	"fmt"
	"github.com/samber/lo"
	"strings"
)

type TabTable struct {
	Headers []string
	Rows    [][]string
	RowMaps []map[string]string
}

func ParseTable(table string) (TabTable, error) {

	// Sanitize input into non-empty lines
	lines := strings.Split(strings.TrimSpace(table), "\n")
	lines = lo.Map(lines, func(item string, _ int) string {
		return strings.TrimSpace(item)
	})
	lines = lo.Filter(lines, func(item string, _ int) bool {
		return item != ""
	})

	if len(lines) == 0 {
		return TabTable{}, fmt.Errorf("table has no lines, not even headers")
	}

	parseColumns := func(line string) []string {
		items := strings.Split(line, "\t")
		items = lo.Map(items, func(item string, _ int) string {
			return strings.TrimSpace(item)
		})
		items = lo.Filter(items, func(item string, _ int) bool {
			return item != ""
		})
		return items
	}

	result := TabTable{
		Headers: parseColumns(lines[0]),
	}

	lines = lines[1:]
	result.Rows = make([][]string, len(lines))
	result.RowMaps = make([]map[string]string, len(lines))
	for i, line := range lines {
		result.Rows[i] = parseColumns(line)
		result.RowMaps[i] = make(map[string]string)
		for j, header := range result.Headers {
			if len(result.Rows[i]) > j {
				result.RowMaps[i][header] = result.Rows[i][j]
			}
		}
	}

	return result, nil
}
