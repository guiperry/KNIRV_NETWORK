// skill_registry.p - Skill Registry State Machine for KNIRVCHAIN
// Models the registration and management of AI skills with LoRA adapters
// Based on: KNIRVCHAIN skill registry implementation

machine SkillRegistryMachine {
    // State variables
    var skills: map[UUID, Skill];
    var skillsByName: map[string, UUID];
    var skillsByOwner: map[Address, seq[UUID]];
    var skillCounter: int;

    // Configuration
    var registrationFee: BigInt;
    var updateFee: BigInt;
    var minStakeForRegistration: BigInt;

    // References
    var tokenMachine: machine;

    // Temp variables
    var tempSkillID: UUID;
    var tempNewSkill: Skill;
    var tempSkill: Skill;
    var tempOwnerSkills: seq[UUID];
    var tempTimestamp: Timestamp;
    var tempStatus: SkillStatus;

    start state Init {
        entry {
            skills = default(map[UUID, Skill]);
            skillsByName = default(map[string, UUID]);
            skillsByOwner = default(map[Address, seq[UUID]]);
            skillCounter = 0;

            registrationFee.value = 10000000;  // 10 NRN
            registrationFee.isNegative = false;
            updateFee.value = 1000000;  // 1 NRN
            updateFee.isNegative = false;
            minStakeForRegistration.value = 100000000;  // 100 NRN
            minStakeForRegistration.isNegative = false;
        }

        on eComponentStart do {
            goto Active;
        }

        on eNetworkStart do {
            goto Active;
        }
    }

    state Active {
        // Skill registration
        on eRegisterSkill do (payload: (
            name: string,
            version: string,
            owner: Address,
            loraPointer: string,
            metadata: map[string, any]
        )) {
            // Check if skill name already exists
            if (payload.name in skillsByName) {
                send this, eSkillRegistrationFailed, "Skill name already registered";
                return;
            }

            // Validate LoRA pointer
            if (payload.loraPointer == "") {
                send this, eSkillRegistrationFailed, "Invalid LoRA pointer";
                return;
            }

            // Create skill ID
            skillCounter = skillCounter + 1;
            tempSkillID.value = format("skill_{0}", skillCounter);

            // Create skill
            tempTimestamp.milliseconds = 0;
            tempStatus.status = "pending";

            tempNewSkill.id = tempSkillID;
            tempNewSkill.name = payload.name;
            tempNewSkill.version = payload.version;
            tempNewSkill.owner = payload.owner;
            tempNewSkill.loraPointer = payload.loraPointer;
            tempNewSkill.metadataPayload = payload.metadata;
            tempNewSkill.createdAt = tempTimestamp;
            tempNewSkill.status = tempStatus;

            // Store skill
            skills[tempSkillID] = tempNewSkill;
            skillsByName[payload.name] = tempSkillID;

            // Add to owner's skills
            if (payload.owner in skillsByOwner) {
                tempOwnerSkills = skillsByOwner[payload.owner];
            } else {
                tempOwnerSkills = default(seq[UUID]);
            }
            tempOwnerSkills += (sizeof(tempOwnerSkills), tempSkillID);
            skillsByOwner[payload.owner] = tempOwnerSkills;

            // Activate skill
            tempStatus.status = "active";
            tempNewSkill.status = tempStatus;
            skills[tempSkillID] = tempNewSkill;

            announce eSkillRegistered, tempNewSkill;
            send this, eSkillRegistered, tempNewSkill;
        }

        // Skill update
        on eUpdateSkill do (payload: (skillID: UUID, updates: map[string, any])) {
            if (!(payload.skillID in skills)) {
                return;
            }

            tempSkill = skills[payload.skillID];

            // Apply version update if provided
            if ("version" in payload.updates) {
                tempSkill.version = payload.updates["version"] as string;
            }

            skills[payload.skillID] = tempSkill;

            send this, eSkillUpdated, tempSkill;
        }

        // Skill deprecation
        on eDeprecateSkill do (payload: (skillID: UUID)) {
            if (!(payload.skillID in skills)) {
                return;
            }

            tempSkill = skills[payload.skillID];
            tempStatus.status = "deprecated";
            tempSkill.status = tempStatus;
            skills[payload.skillID] = tempSkill;

            send this, eSkillDeprecated, payload.skillID;
        }

        // Skill query
        on eQuerySkill do (payload: (skillID: UUID)) {
            if (payload.skillID in skills) {
                send this, eSkillQueryResult, skills[payload.skillID];
            }
        }

        // Handle skill node mined from error (integration with node transformation)
        on eSkillNodeMined do (payload: (skillNode: SkillNode, reward: BigInt)) {
            // Create a skill from the mined skill node
            skillCounter = skillCounter + 1;
            tempSkillID.value = format("skill_mined_{0}", skillCounter);

            tempTimestamp = payload.skillNode.timestamp;
            tempStatus.status = "active";

            tempNewSkill.id = tempSkillID;
            tempNewSkill.name = format("MinedSkill_{0}", payload.skillNode.id.value);
            tempNewSkill.version = "1.0.0";
            tempNewSkill.owner = payload.skillNode.trainer;
            tempNewSkill.loraPointer = payload.skillNode.loraAdapter;
            tempNewSkill.metadataPayload = default(map[string, any]);
            tempNewSkill.createdAt = tempTimestamp;
            tempNewSkill.status = tempStatus;

            skills[tempSkillID] = tempNewSkill;

            // Add to owner's skills
            if (payload.skillNode.trainer in skillsByOwner) {
                tempOwnerSkills = skillsByOwner[payload.skillNode.trainer];
            } else {
                tempOwnerSkills = default(seq[UUID]);
            }
            tempOwnerSkills += (sizeof(tempOwnerSkills), tempSkillID);
            skillsByOwner[payload.skillNode.trainer] = tempOwnerSkills;

            announce eSkillRegistered, tempNewSkill;
        }

        on eNetworkShutdown do {
            goto Halted;
        }

        on eEmergencyHalt do {
            goto Halted;
        }
    }

    state Halted {
        on eEmergencyResume do {
            goto Active;
        }
    }

    // Helper functions
    fun GetSkillCount(): int {
        return sizeof(skills);
    }

    fun GetSkillByName(name: string): Skill {
        if (name in skillsByName) {
            return skills[skillsByName[name]];
        }
        return default(Skill);
    }

    fun IsSkillActive(skillID: UUID): bool {
        if (skillID in skills) {
            return skills[skillID].status.status == "active";
        }
        return false;
    }
}
