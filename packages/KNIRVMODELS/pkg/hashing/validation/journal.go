package validation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// journalEntry is one durable mutation of the knowledge base. The base is the
// built-in default rules; the journal is replayed over it, so deletes of
// default rules persist too.
type journalEntry struct {
	Op     string       `json:"op"` // add | delete
	Domain string       `json:"domain"`
	Rule   *LogicalRule `json:"rule,omitempty"`
	Index  int          `json:"index,omitempty"`
}

// RuleJournal is an append-only JSONL file of knowledge-base mutations.
type RuleJournal struct{ Path string }

// Open returns a validator whose knowledge base is the defaults plus every
// journalled mutation. A missing file is an empty journal.
func (j *RuleJournal) Open() (*LogicalValidator, error) {
	v, err := NewLogicalValidator()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(j.Path)
	if os.IsNotExist(err) {
		return v, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for n := 1; sc.Scan(); n++ {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var e journalEntry
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", j.Path, n, err)
		}
		switch e.Op {
		case "add":
			if e.Rule == nil {
				return nil, fmt.Errorf("%s line %d: add without rule", j.Path, n)
			}
			err = v.KnowledgeBase.AddRule(e.Domain, e.Rule)
		case "delete":
			err = v.KnowledgeBase.RemoveRule(e.Domain, e.Index)
		default:
			err = fmt.Errorf("unknown op %q", e.Op)
		}
		if err != nil {
			return nil, fmt.Errorf("%s line %d: %w", j.Path, n, err)
		}
	}
	return v, sc.Err()
}

func (j *RuleJournal) append(e journalEntry) error {
	if err := os.MkdirAll(filepath.Dir(j.Path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(j.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

// Add applies the rule to v and journals it. The journal is written only after
// the in-memory change succeeds, and the change is rolled back if the write fails.
func (j *RuleJournal) Add(v *LogicalValidator, domain string, rule *LogicalRule) error {
	if err := v.KnowledgeBase.AddRule(domain, rule); err != nil {
		return err
	}
	if err := j.append(journalEntry{Op: "add", Domain: domain, Rule: rule}); err != nil {
		rules := v.KnowledgeBase.Domains[domain]
		v.KnowledgeBase.Domains[domain] = rules[:len(rules)-1]
		return err
	}
	return nil
}

// Delete removes rule index from domain and journals it.
func (j *RuleJournal) Delete(v *LogicalValidator, domain string, index int) error {
	if err := v.KnowledgeBase.RemoveRule(domain, index); err != nil {
		return err
	}
	return j.append(journalEntry{Op: "delete", Domain: domain, Index: index})
}
