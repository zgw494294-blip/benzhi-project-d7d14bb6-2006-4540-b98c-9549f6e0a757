package domain

import (
	"fmt"
	"strings"
	"time"
)

func NewDossier(building, title, boundary string) (*SurveyDossier, error) {
	if strings.TrimSpace(building) == "" || strings.TrimSpace(title) == "" || strings.TrimSpace(boundary) == "" {
		return nil, Invalid("INVALID_INPUT", "建筑标识、标题和勘察边界不能为空")
	}
	now := time.Now().UTC()
	d := &SurveyDossier{ID: Digest(struct {
		B, T string
		N    int64
	}{building, title, now.UnixNano()})[:20], BuildingCode: building, Title: title, SurveyBoundary: boundary, State: StateDraft, Version: 1, CreatedAt: now, UpdatedAt: now, Components: map[string]TimberComponent{}, Observations: map[string][]ConditionObservation{}, Findings: map[string]ReviewFinding{}}
	return d, nil
}
func (d *SurveyDossier) AddComponent(c TimberComponent) error {
	return d.AddComponents([]TimberComponent{c})
}

func (d *SurveyDossier) AddComponents(components []TimberComponent) error {
	if err := d.Mutable(); err != nil {
		return err
	}
	if d.State != StateDraft && d.State != StateSurveying {
		return d.err("INVALID_STATE", "只有草稿或勘察中的档案可登记构件")
	}
	if len(components) == 0 {
		return d.err("INVALID_INPUT", "构件列表不能为空")
	}
	existing := make(map[string]bool, len(d.Components))
	for _, x := range d.Components {
		existing[x.ComponentCode] = true
	}
	batch := make(map[string]bool, len(components))
	for i := range components {
		c := &components[i]
		c.ComponentCode = strings.TrimSpace(c.ComponentCode)
		c.ComponentType = strings.TrimSpace(c.ComponentType)
		c.LoadPathParentCode = strings.TrimSpace(c.LoadPathParentCode)
		if c.ComponentCode == "" || c.ComponentType == "" {
			return d.err("INVALID_INPUT", fmt.Sprintf("构件[%d]的编号和类型不能为空", i))
		}
		if existing[c.ComponentCode] || batch[c.ComponentCode] {
			return d.err("DUPLICATE_COMPONENT", fmt.Sprintf("构件%s的componentCode重复", c.ComponentCode))
		}
		batch[c.ComponentCode] = true
	}
	for _, c := range components {
		if c.LoadPathParentCode == c.ComponentCode {
			return d.err("INVALID_LOAD_PATH", fmt.Sprintf("构件%s不能引用自身作为承重上级", c.ComponentCode))
		}
		if c.LoadPathParentCode != "" && !existing[c.LoadPathParentCode] && !batch[c.LoadPathParentCode] {
			return d.err("LOAD_PATH_PARENT_NOT_FOUND", fmt.Sprintf("构件%s的承重上级%s不存在", c.ComponentCode, c.LoadPathParentCode))
		}
	}
	parents := make(map[string]string, len(d.Components)+len(components))
	for _, c := range d.Components {
		parents[c.ComponentCode] = c.LoadPathParentCode
	}
	for _, c := range components {
		parents[c.ComponentCode] = c.LoadPathParentCode
	}
	for _, c := range components {
		seen := map[string]bool{c.ComponentCode: true}
		for parent := c.LoadPathParentCode; parent != ""; parent = parents[parent] {
			if seen[parent] {
				return d.err("INVALID_LOAD_PATH", fmt.Sprintf("构件%s形成承重传递环", c.ComponentCode))
			}
			seen[parent] = true
		}
	}
	for _, input := range components {
		c := input
		c.ID = Digest(struct{ D, C string }{d.ID, c.ComponentCode})[:20]
		c.DossierID = d.ID
		c.RequiredChecks = append([]string(nil), c.RequiredChecks...)
		d.Components[c.ID] = c
	}
	if d.State == StateDraft {
		if err := d.Transition(StateSurveying); err != nil {
			return err
		}
	}
	d.bump()
	return nil
}
func (d *SurveyDossier) bump() { d.Version++; d.UpdatedAt = time.Now().UTC() }
