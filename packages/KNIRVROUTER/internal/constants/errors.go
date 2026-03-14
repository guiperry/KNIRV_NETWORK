package constants

import "errors"

// Error constants for the application
var (
	// P2P errors
	ErrP2PConsensusManagerNotInitialized = errors.New("p2p consensus manager not initialized")
	ErrP2PDiscoveryManagerNotInitialized = errors.New("p2p discovery manager not initialized")
	ErrP2PStreamFailed                   = errors.New("failed to open p2p stream")
	ErrP2PMessageEncoding                = errors.New("failed to encode p2p message")
	ErrP2PMessageDecoding                = errors.New("failed to decode p2p message")
	ErrP2PInvalidResponse                = errors.New("invalid p2p response")
	ErrP2PRequestTimeout                 = errors.New("p2p request timed out")
	
	// Blockchain errors
	ErrBlockchainNotInitialized          = errors.New("blockchain not initialized")
	ErrInvalidBlock                      = errors.New("invalid block")
	ErrInvalidTransaction                = errors.New("invalid transaction")
	ErrDuplicateTransaction              = errors.New("duplicate transaction")
	ErrInsufficientFunds                 = errors.New("insufficient funds")
	
	// Database errors
	ErrDatabaseNotInitialized            = errors.New("database not initialized")
	ErrDatabaseOperationFailed           = errors.New("database operation failed")
	
	// Network errors
	ErrNetworkNotInitialized             = errors.New("network not initialized")
	ErrNetworkConnectionFailed           = errors.New("network connection failed")
	
	// General errors
	ErrInvalidArgument                   = errors.New("invalid argument")
	ErrOperationFailed                   = errors.New("operation failed")
	ErrNotImplemented                    = errors.New("not implemented")
)