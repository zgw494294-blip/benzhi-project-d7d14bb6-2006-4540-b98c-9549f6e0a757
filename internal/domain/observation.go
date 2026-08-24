package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

var MeasurementLimits = map[string]float64{
	"area": 100000000, "crackLength": 100000, "crackWidth": 1000,
	"decayDepth": 10000, "deflection": 10000, "depth": 10000,
	"length": 100000, "lossRatio": 100, "moistureContent": 100,
	"moisturePercent": 100, "width": 10000,
}

func (d *SurveyDossier) AddObservation(o ConditionObservation) error {
	if err := d.Mutable(); err != nil {
		return err
	}
	if d.State != StateSurveying && d.State != StateAssessed {
		return d.err("INVALID_STATE", "只有勘察中或已校核档案可提交观察修订")
	}
	c, ok := d.Components[o.ComponentID]
	if !ok {
		return d.err("UNKNOWN_COMPONENT", "构件不存在")
	}
	o.ConditionType = strings.TrimSpace(o.ConditionType)
	o.LocationDetail = strings.TrimSpace(o.LocationDetail)
	o.EvidenceRefs = normalizeStrings(o.EvidenceRefs)
	if o.ConditionType == "" || o.LocationDetail == "" || len(o.EvidenceRefs) == 0 {
		return d.err("INVALID_OBSERVATION", "病害类型、位置和证据引用不能为空")
	}
	if o.Severity != SeverityLow && o.Severity != SeverityMedium && o.Severity != SeverityHigh && o.Severity != SeverityCritical {
		return d.err("INVALID_SEVERITY", "severity只允许LOW、MEDIUM、HIGH或CRITICAL")
	}
	if o.ObservedAt.IsZero() || o.ObservedAt.After(time.Now().UTC()) {
		return d.err("INVALID_OBSERVED_AT", "observedAt必须有效且不能晚于当前时间")
	}
	measurements := make(map[string]float64, len(o.Measurements))
	for k, v := range o.Measurements {
		limit, ok := MeasurementLimits[k]
		if !ok {
			return d.err("INVALID_MEASUREMENT_KEY", fmt.Sprintf("测量键%s不受支持", k))
		}
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > limit {
			return d.err("MEASUREMENT_OUT_OF_RANGE", fmt.Sprintf("构件%s的测量值%s超出0到%g的允许范围", c.ComponentCode, k, limit))
		}
		measurements[k] = v
	}
	o.Measurements = measurements
	xs := d.Observations[o.ComponentID]
	o.Revision = uint32(len(xs) + 1)
	o.ID = Digest(struct {
		D, C uint32
		X    string
	}{o.Revision, uint32(len(xs)), c.ID + o.ConditionType})[:20]
	o.DossierID = d.ID
	if len(xs) > 0 {
		o.SupersedesID = xs[len(xs)-1].ID
	}
	d.Observations[o.ComponentID] = append(xs, o)
	d.bump()
	return nil
}

func normalizeStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
func (d *SurveyDossier) LatestObservation(componentID string) (ConditionObservation, bool) {
	xs := d.Observations[componentID]
	if len(xs) == 0 {
		return ConditionObservation{}, false
	}
	return xs[len(xs)-1], true
}
