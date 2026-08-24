package store

const SchemaVersion = 1

type Schema struct {
	Version     int    `json:"schemaVersion"`
	Description string `json:"description"`
}

func CurrentSchema() Schema {
	return Schema{Version: SchemaVersion, Description: "古建木构修缮放行事件账本"}
}
func SupportedSchema(v int) bool { return v == SchemaVersion }
func EventTypes() []string {
	return []string{"DOSSIER_CREATED", "COMPONENT_ADDED", "OBSERVATION_ADDED", "ASSESSED", "PLAN_SUBMITTED", "REVIEW_RECORDED", "DOSSIER_FROZEN", "RELEASE_ISSUED"}
}
