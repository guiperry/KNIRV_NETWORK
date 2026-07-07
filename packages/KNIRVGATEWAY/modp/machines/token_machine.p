// token_machine.p - P State Machine for NRN Token
// Models the NRN token operations including transfers, minting, and burning.
// Based on: KNIRVGATEWAY/internal/oracle/token/nrn.go

machine TokenMachine {
    // State variables
    var name: string;
    var symbol: string;
    var decimals: int;
    var totalSupply: BigInt;
    var maxSupply: BigInt;
    var owner: Address;
    var balances: map[Address, BigInt];
    var txCounter: int;

    // Start state
    start state Init {
        entry {
            name = "KNIRV Network Token";
            symbol = "NRN";
            decimals = 18;
            totalSupply = (value = 1000000000, isNegative = false);  // 1 billion initial
            maxSupply = (value = 10000000000, isNegative = false);   // 10 billion max
            owner = default(Address);
            balances = default(map[Address, BigInt]);
            txCounter = 0;

            announce eComponentStarted, (componentName = "TokenMachine");
        }

        // Initialize with owner
        on eOracleStart do {
            goto Active;
        }

        // Can respond to info queries even in init
        on eGetTokenInfo do {
            var info: TokenInfo;
            info = (
                name = name,
                symbol = symbol,
                decimals = decimals,
                totalSupply = totalSupply,
                maxSupply = maxSupply,
                owner = owner
            );
            announce eTokenInfoResponse, (info = info);
        }
    }

    state Active {
        // Handle transfer request
        on eTransferRequest do (payload: (from: Address, to: Address, amount: BigInt)) {
            // Validate amount
            if (payload.amount.isNegative || payload.amount.value <= 0) {
                announce eTransferFailed, (
                    from = payload.from,
                    to = payload.to,
                    amount = payload.amount,
                    error = InvalidAmount()
                );
                return;
            }

            // Get sender balance
            var senderBalance: BigInt;
            if (payload.from in balances) {
                senderBalance = balances[payload.from];
            } else {
                senderBalance = (value = 0, isNegative = false);
            }

            // Check sufficient balance
            if (senderBalance.value < payload.amount.value) {
                announce eTransferFailed, (
                    from = payload.from,
                    to = payload.to,
                    amount = payload.amount,
                    error = InsufficientBalance()
                );
                return;
            }

            // Execute transfer
            var newSenderBalance: BigInt;
            newSenderBalance = (value = senderBalance.value - payload.amount.value, isNegative = false);
            balances[payload.from] = newSenderBalance;

            // Update recipient balance
            var recipientBalance: BigInt;
            if (payload.to in balances) {
                recipientBalance = balances[payload.to];
            } else {
                recipientBalance = (value = 0, isNegative = false);
            }
            balances[payload.to] = (value = recipientBalance.value + payload.amount.value, isNegative = false);

            // Generate receipt
            txCounter = txCounter + 1;
            var receipt: TransferReceipt;
            receipt = (
                txHash = GenerateTxHash(txCounter),
                from = payload.from,
                to = payload.to,
                amount = payload.amount,
                timestamp = 0,  // Would be current timestamp
                success = true
            );

            announce eTransferResponse, (success = true, receipt = receipt);
        }

        // Handle mint request (only authorized mints)
        on eMintRequest do (payload: (to: Address, amount: BigInt, authorized: bool)) {
            // Check authorization
            if (!payload.authorized) {
                announce eMintFailed, (
                    to = payload.to,
                    amount = payload.amount,
                    error = UnauthorizedMint()
                );
                // Notify monitor of unauthorized mint attempt
                announce eTokenSupplyInvariantViolation, (
                    totalSupply = totalSupply,
                    maxSupply = maxSupply,
                    unauthorized = true
                );
                return;
            }

            // Validate amount
            if (payload.amount.isNegative || payload.amount.value <= 0) {
                announce eMintFailed, (
                    to = payload.to,
                    amount = payload.amount,
                    error = InvalidAmount()
                );
                return;
            }

            // Check max supply limit
            var newTotalSupply: int;
            newTotalSupply = totalSupply.value + payload.amount.value;
            if (newTotalSupply > maxSupply.value) {
                announce eMintFailed, (
                    to = payload.to,
                    amount = payload.amount,
                    error = ExceedsMaxSupply()
                );
                announce eTokenSupplyInvariantViolation, (
                    totalSupply = (value = newTotalSupply, isNegative = false),
                    maxSupply = maxSupply,
                    unauthorized = false
                );
                return;
            }

            // Execute mint
            totalSupply = (value = newTotalSupply, isNegative = false);

            // Update recipient balance
            var recipientBalance: BigInt;
            if (payload.to in balances) {
                recipientBalance = balances[payload.to];
            } else {
                recipientBalance = (value = 0, isNegative = false);
            }
            balances[payload.to] = (value = recipientBalance.value + payload.amount.value, isNegative = false);

            // Generate receipt
            txCounter = txCounter + 1;
            var receipt: MintReceipt;
            receipt = (
                txHash = GenerateTxHash(txCounter),
                to = payload.to,
                amount = payload.amount,
                newTotalSupply = totalSupply,
                timestamp = 0,
                success = true
            );

            announce eMintResponse, (success = true, receipt = receipt);
        }

        // Handle burn request
        on eBurnRequest do (payload: (from: Address, amount: BigInt, reason: string)) {
            // Validate amount
            if (payload.amount.isNegative || payload.amount.value <= 0) {
                announce eBurnFailed, (
                    from = payload.from,
                    amount = payload.amount,
                    error = InvalidAmount()
                );
                return;
            }

            // Get sender balance
            var senderBalance: BigInt;
            if (payload.from in balances) {
                senderBalance = balances[payload.from];
            } else {
                senderBalance = (value = 0, isNegative = false);
            }

            // Check sufficient balance
            if (senderBalance.value < payload.amount.value) {
                announce eBurnFailed, (
                    from = payload.from,
                    amount = payload.amount,
                    error = InsufficientBalance()
                );
                return;
            }

            // Execute burn
            balances[payload.from] = (value = senderBalance.value - payload.amount.value, isNegative = false);
            totalSupply = (value = totalSupply.value - payload.amount.value, isNegative = false);

            // Generate receipt
            txCounter = txCounter + 1;
            var receipt: BurnReceipt;
            receipt = (
                txHash = GenerateTxHash(txCounter),
                from = payload.from,
                amount = payload.amount,
                reason = payload.reason,
                newTotalSupply = totalSupply,
                timestamp = 0,
                success = true
            );

            announce eBurnResponse, (success = true, receipt = receipt);
        }

        // Handle balance query
        on eGetBalance do (payload: (address: Address)) {
            var balance: BigInt;
            if (payload.address in balances) {
                balance = balances[payload.address];
            } else {
                balance = (value = 0, isNegative = false);
            }

            announce eBalanceResponse, (address = payload.address, balance = balance);
        }

        // Handle token info query
        on eGetTokenInfo do {
            var info: TokenInfo;
            info = (
                name = name,
                symbol = symbol,
                decimals = decimals,
                totalSupply = totalSupply,
                maxSupply = maxSupply,
                owner = owner
            );
            announce eTokenInfoResponse, (info = info);
        }

        // Handle shutdown
        on eOracleStop do {
            goto Stopped;
        }
    }

    state Stopped {
        entry {
            announce eComponentStopped, (componentName = "TokenMachine");
        }

        // Allow restart
        on eOracleStart do {
            goto Active;
        }

        // Still respond to info queries in stopped state
        on eGetTokenInfo do {
            var info: TokenInfo;
            info = (
                name = name,
                symbol = symbol,
                decimals = decimals,
                totalSupply = totalSupply,
                maxSupply = maxSupply,
                owner = owner
            );
            announce eTokenInfoResponse, (info = info);
        }

        on eGetBalance do (payload: (address: Address)) {
            var balance: BigInt;
            if (payload.address in balances) {
                balance = balances[payload.address];
            } else {
                balance = (value = 0, isNegative = false);
            }
            announce eBalanceResponse, (address = payload.address, balance = balance);
        }

        ignore eTransferRequest, eMintRequest, eBurnRequest;
    }
}

// =====================================================================
// Helper Functions
// =====================================================================

// GenerateTxHash creates a unique transaction hash
fun GenerateTxHash(counter: int): string {
    // In real implementation, this would be a cryptographic hash
    return format("tx_{0}", counter);
}

// ValidateBigInt checks if a BigInt value is valid for token operations
fun ValidateBigInt(value: BigInt): bool {
    return !value.isNegative && value.value >= 0;
}

// AddBigInt adds two BigInt values
fun AddBigInt(a: BigInt, b: BigInt): BigInt {
    // Simplified: assumes non-negative values
    return (value = a.value + b.value, isNegative = false);
}

// SubtractBigInt subtracts two BigInt values
fun SubtractBigInt(a: BigInt, b: BigInt): BigInt {
    if (a.value < b.value) {
        return (value = b.value - a.value, isNegative = true);
    }
    return (value = a.value - b.value, isNegative = false);
}

// CompareBigInt compares two BigInt values (-1, 0, 1)
fun CompareBigInt(a: BigInt, b: BigInt): int {
    if (a.value < b.value) return -1;
    if (a.value > b.value) return 1;
    return 0;
}
