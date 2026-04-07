// EntryType specifies the kind of data stored.
export var EntryType;
(function (EntryType) {
    EntryType["Memory"] = "MEMORY";
    EntryType["Auth"] = "AUTH";
})(EntryType || (EntryType = {}));
// MemoryCategory specifies the category of memory entries
export var MemoryCategory;
(function (MemoryCategory) {
    MemoryCategory["Error"] = "ERROR";
    MemoryCategory["Context"] = "CONTEXT";
    MemoryCategory["Idea"] = "IDEA";
    MemoryCategory["Solution"] = "SOLUTION";
    MemoryCategory["Skill"] = "SKILL";
    MemoryCategory["Generic"] = "GENERIC";
    MemoryCategory["Event"] = "EVENT";
    MemoryCategory["Preference"] = "PREFERENCE";
    MemoryCategory["Trait"] = "TRAIT";
})(MemoryCategory || (MemoryCategory = {}));
// OperationType enumerates CRDT operation kinds
export var OperationType;
(function (OperationType) {
    OperationType[OperationType["Insert"] = 0] = "Insert";
    OperationType[OperationType["Update"] = 1] = "Update";
    OperationType[OperationType["Delete"] = 2] = "Delete";
})(OperationType || (OperationType = {}));
// MessageType strings for protocol
export var MessageType;
(function (MessageType) {
    MessageType["SyncRequest"] = "sync_request";
    MessageType["SyncResponse"] = "sync_response";
    MessageType["Operation"] = "operation";
    MessageType["Heartbeat"] = "heartbeat";
    MessageType["CollectionAnnounce"] = "collection_announce";
    MessageType["CollectionRequest"] = "collection_request";
    MessageType["DhtPut"] = "dht_put";
    MessageType["DhtGet"] = "dht_get";
})(MessageType || (MessageType = {}));
//# sourceMappingURL=types.js.map