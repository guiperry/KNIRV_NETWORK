package types

// Block defines the interface for block operations
type Block interface {
    // Block identification
    BlockNumber() uint64
    Hash() string
    HashString() string
    PrevHash() string
    PrevHashString() string
    Timestamp() int64
    
    // Block operations
    VerifyHash() bool
    VerifyBlock() bool
    
    // Transaction operations
    Transactions() []*Transaction
}