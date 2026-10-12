package normalizer

type SecurityMapper struct {
	mappings map[string]*SecurityMapping
}

type SecurityMapping struct {
	Tag    string
	Slot4  int
	Slot10 int
	Weight float64
}

func NewSecurityMapper() *SecurityMapper {
	return &SecurityMapper{
		mappings: map[string]*SecurityMapping{
			"filesystem:block": {
				Tag:    "guardrail_block",
				Slot4:  0x07,
				Slot10: 0x2400,
				Weight: -1.0,
			},
			"execution:block": {
				Tag:    "guardrail_block",
				Slot4:  0x02,
				Slot10: 0x2400,
				Weight: -1.0,
			},
			"network:block": {
				Tag:    "guardrail_block",
				Slot4:  0x02,
				Slot10: 0x2400,
				Weight: -1.0,
			},
			"memory:block": {
				Tag:    "guardrail_block",
				Slot4:  0x01,
				Slot10: 0x2400,
				Weight: -1.0,
			},
			"cpu:block": {
				Tag:    "guardrail_block",
				Slot4:  0x01,
				Slot10: 0x2400,
				Weight: -1.0,
			},
			"filesystem:warn": {
				Tag:    "guardrail_warn",
				Slot4:  0x04,
				Slot10: 0x2401,
				Weight: -0.5,
			},
			"execution:warn": {
				Tag:    "guardrail_warn",
				Slot4:  0x04,
				Slot10: 0x2401,
				Weight: -0.5,
			},
			"constraint:allow": {
				Tag:    "security_constraint",
				Slot4:  0x02,
				Slot10: 0x2400,
				Weight: 1.0,
			},
			"constraint:deny": {
				Tag:    "security_constraint",
				Slot4:  0x02,
				Slot10: 0x2400,
				Weight: -1.0,
			},
			"constraint:flag": {
				Tag:    "security_constraint",
				Slot4:  0x04,
				Slot10: 0x2400,
				Weight: -0.3,
			},
			"violation:critical": {
				Tag:    "security_violation",
				Slot4:  0x01,
				Slot10: 0x2402,
				Weight: -2.0,
			},
			"violation:high": {
				Tag:    "security_violation",
				Slot4:  0x01,
				Slot10: 0x2402,
				Weight: -1.5,
			},
			"violation:medium": {
				Tag:    "security_violation",
				Slot4:  0x01,
				Slot10: 0x2402,
				Weight: -1.0,
			},
			"violation:low": {
				Tag:    "security_violation",
				Slot4:  0x01,
				Slot10: 0x2402,
				Weight: -0.5,
			},
		},
	}
}

func (m *SecurityMapper) GetMapping(guardrailType, action string) *SecurityMapping {
	key := guardrailType + ":" + action
	if mapping, ok := m.mappings[key]; ok {
		return mapping
	}
	return &SecurityMapping{
		Tag:    "unknown",
		Slot4:  0x01,
		Slot10: 0x1000,
		Weight: 0.0,
	}
}

func (m *SecurityMapper) GetAllMappings() []*SecurityMapping {
	mappings := make([]*SecurityMapping, 0, len(m.mappings))
	for _, v := range m.mappings {
		mappings = append(mappings, v)
	}
	return mappings
}
