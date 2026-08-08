package types

import "time"

// SkillOwnershipRecord grants an agent a durable, perpetual (or bounded)
// invocation-fee entitlement on a canonically-minted skill, registered via
// POST /oracle/v3/skills/ownership. This is KNIRVORACLE's real ledger for
// what KNIRVGRAPH's DRQ skill-minting pipeline previously only modeled as an
// in-memory SkillOwnershipRights struct with no durable counterpart.
type SkillOwnershipRecord struct {
	SkillID       string    `json:"skill_id"`
	AgentID       string    `json:"agent_id"`
	InvocationFee float64   `json:"invocation_fee"`
	Perpetual     bool      `json:"perpetual"`
	RegisteredAt  time.Time `json:"registered_at"`
}
