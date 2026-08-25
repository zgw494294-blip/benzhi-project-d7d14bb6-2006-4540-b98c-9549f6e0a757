package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"timber-release-gate/internal/domain"
)

func (s *Store) VerifyChain() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := s.verificationLedger()
	if os.IsNotExist(err) && len(s.events) == 0 {
		return nil
	}
	if err != nil {
		return fmt.Errorf("账本读取失败: %w", err)
	}
	persisted := make([]EventRecord, 0, len(s.events))
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record EventRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return fmt.Errorf("账本记录损坏: %w", err)
		}
		persisted = append(persisted, record)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("账本读取失败: %w", err)
	}
	if len(persisted) != len(s.events) {
		return fmt.Errorf("账本记录数量不一致")
	}
	prev := ""
	for i, record := range persisted {
		e := record.Event
		if e.Sequence != uint64(i+1) {
			return fmt.Errorf("账本序号错误: %d", e.Sequence)
		}
		if e.PreviousHash != prev {
			return fmt.Errorf("账本前序哈希错误: %d", e.Sequence)
		}
		expected := domain.AuditHash(e)
		if e.Hash != expected {
			return fmt.Errorf("账本哈希错误: %d", e.Sequence)
		}
		if record.Dossier == nil || e.PayloadHash != domain.Digest(record.Dossier) {
			return fmt.Errorf("账本载荷摘要错误: %d", e.Sequence)
		}
		if s.events[i].Hash != e.Hash {
			return fmt.Errorf("账本内存投影不一致: %d", e.Sequence)
		}
		prev = e.Hash
	}
	return nil
}

func (s *Store) verificationLedger() (*os.File, error) {
	if s.ledgerFile == nil {
		file, err := os.Open(filepath.Join(s.dir, "events.jsonl"))
		if err != nil {
			return nil, err
		}
		s.ledgerFile = file
	}
	if _, err := s.ledgerFile.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return s.ledgerFile, nil
}

func (s *Store) SnapshotVersion(id string) (uint64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.dossiers[id]
	if !ok {
		return 0, false
	}
	return d.Version, true
}
