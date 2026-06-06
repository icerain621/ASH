package alerts

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ash-repwiki/ash/internal/store"
	"gorm.io/gorm"
)

// PrometheusOptions controls tenant scope for DB-derived Prometheus text.
type PrometheusOptions struct {
	// SpaceID scopes metrics to one tenant. Empty means deployment-global aggregate.
	SpaceID string
}

func (o PrometheusOptions) normalized() PrometheusOptions {
	o.SpaceID = strings.TrimSpace(o.SpaceID)
	return o
}

func (o PrometheusOptions) scoped() bool {
	return o.SpaceID != ""
}

func (o PrometheusOptions) labelSuffix() string {
	if !o.scoped() {
		return ""
	}
	return fmt.Sprintf(`,space_id=%q`, label(o.SpaceID))
}

func (s *Service) scopedRunsQuery(opts PrometheusOptions) *gorm.DB {
	q := s.gdb().Model(&store.RunRecord{})
	if opts.scoped() {
		q = q.Where("space_id = ?", opts.SpaceID)
	}
	return q
}

func (s *Service) countBy(table, column string, opts PrometheusOptions) []countRow {
	q := s.gdb().Table(table).Select(column + " as key, COUNT(*) as count").Group(column)
	q = s.applyPrometheusScope(q, table, opts)
	var rows []countRow
	_ = q.Scan(&rows).Error
	sortSliceCountRows(rows)
	return rows
}

func sortSliceCountRows(rows []countRow) {
	sort.Slice(rows, func(i, j int) bool { return rows[i].Key < rows[j].Key })
}

func (s *Service) applyPrometheusScope(q *gorm.DB, table string, opts PrometheusOptions) *gorm.DB {
	if !opts.scoped() {
		return q
	}
	switch table {
	case "runs", "feedback", "ci_diagnoses", "alert_events":
		return q.Where(table+".space_id = ?", opts.SpaceID)
	case "tool_calls", "agent_tasks", "run_steps":
		return q.Where(table+".run_id IN (?)",
			s.gdb().Model(&store.RunRecord{}).Select("id").Where("space_id = ?", opts.SpaceID))
	default:
		return q
	}
}
