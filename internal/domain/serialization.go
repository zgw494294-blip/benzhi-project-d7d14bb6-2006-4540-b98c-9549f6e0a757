package domain

import (
	"encoding/json"
	"sort"
)

func CanonicalDossier(d *SurveyDossier) ([]byte, error) {
	type component struct{ ID, Code, Type, Location string }
	cs := make([]component, 0, len(d.Components))
	for _, c := range d.Components {
		cs = append(cs, component{c.ID, c.ComponentCode, c.ComponentType, c.Location})
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
	return json.Marshal(struct {
		ID         string
		State      DossierState
		Version    uint64
		Components []component
		Manifest   *FreezeManifest
	}{d.ID, d.State, d.Version, cs, d.Manifest})
}
func CanonicalPlan(p RepairPlanRevision) []byte {
	b, _ := json.Marshal(struct {
		ID       string
		Revision uint32
		Actions  []RepairAction
	}{p.ID, p.Revision, p.Actions})
	return b
}
