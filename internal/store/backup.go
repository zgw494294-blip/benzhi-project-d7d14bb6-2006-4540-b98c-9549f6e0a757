package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func (s *Store) ExportDossier(id string) ([]byte, error) {
	d, ok := s.Get(id)
	if !ok {
		return nil, os.ErrNotExist
	}
	return json.MarshalIndent(d, "", "  ")
}
func (s *Store) DataPath(name string) string { return filepath.Join(s.dir, name) }
