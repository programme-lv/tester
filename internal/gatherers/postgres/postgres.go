package postgres

import "github.com/programme-lv/tester/internal/testing"

type Gatherer struct {
}

var _ testing.EvalResGatherer = (*Gatherer)(nil)
