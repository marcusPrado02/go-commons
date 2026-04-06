package observability_test

import (
	"context"
	"testing"
	"time"

	"github.com/marcusPrado02/go-commons/app/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type alwaysUpCheck struct {
	name      string
	checkType observability.HealthCheckType
}

func (c alwaysUpCheck) Name() string                                    { return c.name }
func (c alwaysUpCheck) Type() observability.HealthCheckType             { return c.checkType }
func (c alwaysUpCheck) Check(_ context.Context) observability.HealthCheckResult {
	return observability.HealthCheckResult{Status: observability.StatusUp}
}

type alwaysDownCheck struct {
	name      string
	checkType observability.HealthCheckType
}

func (c alwaysDownCheck) Name() string                                    { return c.name }
func (c alwaysDownCheck) Type() observability.HealthCheckType             { return c.checkType }
func (c alwaysDownCheck) Check(_ context.Context) observability.HealthCheckResult {
	return observability.HealthCheckResult{Status: observability.StatusDown}
}

func TestHealthChecks_Liveness_AllUp(t *testing.T) {
	hc := observability.NewHealthChecks(
		alwaysUpCheck{name: "db", checkType: observability.Liveness},
		alwaysUpCheck{name: "cache", checkType: observability.Liveness},
	)

	report := hc.Liveness(context.Background())
	assert.Equal(t, observability.StatusUp, report.Status)
	assert.Len(t, report.Checks, 2)
}

func TestHealthChecks_Liveness_OneDown(t *testing.T) {
	hc := observability.NewHealthChecks(
		alwaysUpCheck{name: "db", checkType: observability.Liveness},
		alwaysDownCheck{name: "cache", checkType: observability.Liveness},
	)

	report := hc.Liveness(context.Background())
	assert.Equal(t, observability.StatusDown, report.Status)
}

func TestHealthChecks_Readiness_FiltersType(t *testing.T) {
	hc := observability.NewHealthChecks(
		alwaysDownCheck{name: "db", checkType: observability.Liveness},  // DOWN, but Liveness
		alwaysUpCheck{name: "queue", checkType: observability.Readiness}, // UP, Readiness
	)

	// Readiness should only evaluate Readiness checks
	report := hc.Readiness(context.Background())
	assert.Equal(t, observability.StatusUp, report.Status)
}

func TestHealthChecks_Report_IncludesTimestamp(t *testing.T) {
	hc := observability.NewHealthChecks(alwaysUpCheck{name: "x", checkType: observability.Liveness})
	before := time.Now()
	report := hc.Liveness(context.Background())
	after := time.Now()

	require.NotZero(t, report.CheckedAt)
	assert.True(t, report.CheckedAt.After(before) || report.CheckedAt.Equal(before))
	assert.True(t, report.CheckedAt.Before(after) || report.CheckedAt.Equal(after))
}
