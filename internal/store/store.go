package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"timber-release-gate/internal/domain"
	"time"
)

type EventRecord struct {
	SchemaVersion  int                   `json:"schemaVersion"`
	Event          domain.AuditEvent     `json:"event"`
	Dossier        *domain.SurveyDossier `json:"dossier,omitempty"`
	IdempotencyKey string                `json:"idempotencyKey,omitempty"`
	RequestHash    string                `json:"requestHash,omitempty"`
}
type Store struct {
	mu       sync.Mutex
	dir      string
	dossiers map[string]*domain.SurveyDossier
	events   []domain.AuditEvent
	idem     map[string]EventRecord
}

func Open(dir string) (*Store, error) {
	if dir == "" {
		dir = "timber-data"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, dossiers: map[string]*domain.SurveyDossier{}, idem: map[string]EventRecord{}}
	if err := s.load(); err != nil {
		return nil, err
	}
	if err := s.VerifyChain(); err != nil {
		return nil, &domain.Error{Code: "AUDIT_INTEGRITY_ERROR", Message: err.Error()}
	}
	return s, nil
}
func (s *Store) load() error {
	p := filepath.Join(s.dir, "events.jsonl")
	f, e := os.Open(p)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for sc.Scan() {
		var r EventRecord
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			return errors.New("账本记录损坏或尾部残缺")
		}
		if !SupportedSchema(r.SchemaVersion) {
			return errors.New("账本schemaVersion不受支持")
		}
		s.events = append(s.events, r.Event)
		if r.Dossier != nil {
			s.dossiers[r.Dossier.ID] = r.Dossier
		}
		if r.IdempotencyKey != "" {
			s.idem[r.IdempotencyKey] = r
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return s.validateOrRecoverSnapshot()
}
func (s *Store) append(r EventRecord) (int64, error) {
	p := filepath.Join(s.dir, "events.jsonl")
	f, e := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if e != nil {
		return 0, e
	}
	defer f.Close()
	var before int64
	if fi, fe := f.Stat(); fe == nil {
		before = fi.Size()
	}
	b, _ := json.Marshal(r)
	if _, e = f.Write(append(b, '\n')); e != nil {
		return before, e
	}
	if e = f.Sync(); e != nil {
		return before, e
	}
	return before, nil
}
// truncateLedger drops any bytes appended beyond offset, used to roll back a
// ledger record when a later persistence step fails.
func (s *Store) truncateLedger(offset int64) {
	p := filepath.Join(s.dir, "events.jsonl")
	if e := os.Truncate(p, offset); e != nil {
		return
	}
	if f, e := os.OpenFile(p, os.O_WRONLY, 0644); e == nil {
		_ = f.Sync()
		_ = f.Close()
	}
}
func (s *Store) Commit(d *domain.SurveyDossier, expected uint64, key, reqHash, typ, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d == nil {
		return &domain.Error{Code: "INVALID_DOSSIER", Message: "档案不能为空"}
	}
	if err := d.Validate(); err != nil {
		return err
	}
	if old, ok := s.dossiers[d.ID]; ok && expected != 0 && old.Version != expected {
		return &domain.Error{Code: "VERSION_CONFLICT", Message: "档案版本已变化", State: old.State, Version: old.Version}
	}
	if key != "" {
		if old, ok := s.idem[key]; ok {
			if old.RequestHash != reqHash {
				return &domain.Error{Code: "IDEMPOTENCY_CONFLICT", Message: "幂等键对应请求不同"}
			}
			return nil
		}
	}
	prev := ""
	if len(s.events) > 0 {
		prev = s.events[len(s.events)-1].Hash
	}
	ev := domain.AuditEvent{Sequence: uint64(len(s.events) + 1), EventID: domain.Digest(struct {
		D string
		N int64
	}{d.ID, time.Now().UnixNano()})[:20], DossierID: d.ID, EventType: typ, Actor: actor, State: d.State, OccurredAt: time.Now().UTC(), PayloadHash: domain.Digest(d), PreviousHash: prev}
	ev.Hash = domain.AuditHash(ev)
	stored := domain.CloneDossier(d)
	r := EventRecord{SchemaVersion: SchemaVersion, Event: ev, Dossier: stored, IdempotencyKey: key, RequestHash: reqHash}
	before, e := s.append(r)
	if e != nil {
		return e
	}
	// Append the ledger record, then write the snapshot projection. Only when
	// both persistence steps succeed do we keep the in-memory state and the
	// appended record. If the snapshot fails we roll the ledger back and
	// restore the previous in-memory projection so queries and recovery never
	// observe a half-committed dossier, event or idempotency record.
	committed := false
	defer func() {
		if !committed {
			s.truncateLedger(before)
		}
	}()
	s.events = append(s.events, ev)
	prevDossier := s.dossiers[d.ID]
	s.dossiers[d.ID] = stored
	var prevIdem EventRecord
	hadIdem := false
	if key != "" {
		if rec, ok := s.idem[key]; ok {
			prevIdem = rec
			hadIdem = true
		}
		s.idem[key] = r
	}
	if e := s.snapshot(); e != nil {
		// Roll back the in-memory projection to its pre-commit state.
		if len(s.events) > 0 {
			s.events = s.events[:len(s.events)-1]
		}
		if prevDossier != nil {
			s.dossiers[d.ID] = prevDossier
		} else {
			delete(s.dossiers, d.ID)
		}
		if key != "" {
			if hadIdem {
				s.idem[key] = prevIdem
			} else {
				delete(s.idem, key)
			}
		}
		return e
	}
	committed = true
	return nil
}
func (s *Store) snapshot() error {
	p := filepath.Join(s.dir, "snapshot.json")
	tmp, e := os.CreateTemp(s.dir, "snapshot-*.tmp")
	if e != nil {
		return e
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	headHash := ""
	lastSequence := uint64(0)
	if len(s.events) > 0 {
		headHash = s.events[len(s.events)-1].Hash
		lastSequence = s.events[len(s.events)-1].Sequence
	}
	b, _ := json.Marshal(struct {
		SchemaVersion int                              `json:"schemaVersion"`
		LastSequence  uint64                           `json:"lastSequence"`
		HeadHash      string                           `json:"headHash"`
		Dossiers      map[string]*domain.SurveyDossier `json:"dossiers"`
	}{SchemaVersion, lastSequence, headHash, s.dossiers})
	if _, e = tmp.Write(b); e == nil {
		e = tmp.Sync()
	}
	closeErr := tmp.Close()
	if e != nil {
		return e
	}
	if closeErr != nil {
		return closeErr
	}
	if e = os.Rename(tmpName, p); e != nil {
		return e
	}
	dir, e := os.Open(s.dir)
	if e != nil {
		return e
	}
	e = dir.Sync()
	closeErr = dir.Close()
	if e != nil {
		return e
	}
	return closeErr
}

func (s *Store) validateOrRecoverSnapshot() error {
	path := filepath.Join(s.dir, "snapshot.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s.snapshot()
	}
	valid := err == nil
	var snapshot struct {
		SchemaVersion int                              `json:"schemaVersion"`
		LastSequence  uint64                           `json:"lastSequence"`
		HeadHash      string                           `json:"headHash"`
		Dossiers      map[string]*domain.SurveyDossier `json:"dossiers"`
	}
	if valid {
		valid = json.Unmarshal(data, &snapshot) == nil && SupportedSchema(snapshot.SchemaVersion)
	}
	if valid {
		expectedSequence := uint64(0)
		expectedHead := ""
		if len(s.events) > 0 {
			expectedSequence = s.events[len(s.events)-1].Sequence
			expectedHead = s.events[len(s.events)-1].Hash
		}
		valid = snapshot.LastSequence == expectedSequence && snapshot.HeadHash == expectedHead && len(snapshot.Dossiers) == len(s.dossiers)
	}
	if valid {
		for id, dossier := range s.dossiers {
			candidate, ok := snapshot.Dossiers[id]
			if !ok || candidate == nil || candidate.Validate() != nil || domain.Digest(candidate) != domain.Digest(dossier) {
				valid = false
				break
			}
		}
	}
	if valid {
		return nil
	}
	return s.snapshot()
}
func (s *Store) Get(id string) (*domain.SurveyDossier, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.dossiers[id]
	return domain.CloneDossier(d), ok
}
func (s *Store) Events(id string) []domain.AuditEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []domain.AuditEvent{}
	for _, e := range s.events {
		if e.DossierID == id {
			out = append(out, e)
		}
	}
	return out
}
func (s *Store) HeadHash() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.events) == 0 {
		return ""
	}
	return s.events[len(s.events)-1].Hash
}
