// token_machine.p - NRN Token State Machine for KNIRVORACLE
// Models the NRN token operations including transfers, minting, and burning
// Based on: KNIRVORACLE token module implementation

machine TokenMachine {
    // State variables
    var balances: map[Address, TokenBalance];
    var totalSupply: BigInt;
    var maxSupply: BigInt;
    var authorizedMinters: set[Address];
    var transfersEnabled: bool;

    // Token configuration
    var tokenName: string;
    var tokenSymbol: string;
    var decimals: int;

    // Temp variables for event construction
    var tmpReceipt: TransferReceipt;
    var tmpBigInt: BigInt;
    var tmpTimestamp: Timestamp;
    var tmpHash: Hash;
    var tmpResponse: TransferResponseData;

    start state Init {
        entry {
            balances = default(map[Address, TokenBalance]);
            totalSupply = (value = 0, isNegative = false);
            maxSupply = (value = 2000000000, isNegative = false); // Max supply (model uses smaller values)
            authorizedMinters = default(set[Address]);
            transfersEnabled = true;
            tokenName = "KNIRV Network Token";
            tokenSymbol = "NRN";
            decimals = 6;
        }

        on eComponentStart do {
            goto Active;
        }

        on eNetworkStart do {
            goto Active;
        }
    }

    state Active {
        // Transfer handling
        on eTransferRequest do (payload: (from: Address, recipient: Address, amount: BigInt, caller: machine)) {
            var fromBalance: TokenBalance;
            var toBalance: TokenBalance;

            // Get or initialize balances
            if (payload.from in balances) {
                fromBalance = balances[payload.from];
            } else {
                fromBalance = (owner = payload.from, available = (value = 0, isNegative = false),
                              staked = (value = 0, isNegative = false), locked = (value = 0, isNegative = false));
            }

            if (payload.recipient in balances) {
                toBalance = balances[payload.recipient];
            } else {
                toBalance = (owner = payload.recipient, available = (value = 0, isNegative = false),
                            staked = (value = 0, isNegative = false), locked = (value = 0, isNegative = false));
            }

            // Check sufficient balance
            if (!transfersEnabled) {
                send payload.caller, eTransferFailed, "Transfers are disabled";
                return;
            }

            if (fromBalance.available.value < payload.amount.value) {
                send payload.caller, eTransferFailed, "Insufficient balance";
                return;
            }

            // Execute transfer
            fromBalance.available.value = fromBalance.available.value - payload.amount.value;
            toBalance.available.value = toBalance.available.value + payload.amount.value;

            balances[payload.from] = fromBalance;
            balances[payload.recipient] = toBalance;

            // Build receipt
            tmpBigInt.value = 0;
            tmpBigInt.isNegative = false;
            tmpTimestamp.milliseconds = 0;
            tmpHash.bytes = default(seq[int]);
            tmpReceipt.from = payload.from;
            tmpReceipt.recipient = payload.recipient;
            tmpReceipt.amount = payload.amount;
            tmpReceipt.fee = tmpBigInt;
            tmpReceipt.timestamp = tmpTimestamp;
            tmpReceipt.txHash = tmpHash;

            tmpResponse.success = true;
            tmpResponse.receipt = tmpReceipt;

            // Announce for monitors
            announce eTransferResponse, tmpResponse;
            send payload.caller, eTransferResponse, tmpResponse;
        }

        // Minting handling
        on eMintRequest do (payload: (recipient: Address, amount: BigInt, authorized: bool, caller: machine)) {
            var newSupply: int;
            var toBalance: TokenBalance;

            // Check authorization
            if (!payload.authorized) {
                announce eMintFailed, "Unauthorized minting attempt";
                send payload.caller, eMintFailed, "Unauthorized minting attempt";
                return;
            }

            // Check max supply
            newSupply = totalSupply.value + payload.amount.value;

            if (newSupply > maxSupply.value) {
                announce eMintFailed, "Would exceed max supply";
                send payload.caller, eMintFailed, "Would exceed max supply";
                return;
            }

            // Execute mint
            if (payload.recipient in balances) {
                toBalance = balances[payload.recipient];
            } else {
                toBalance = (owner = payload.recipient, available = (value = 0, isNegative = false),
                            staked = (value = 0, isNegative = false), locked = (value = 0, isNegative = false));
            }

            toBalance.available.value = toBalance.available.value + payload.amount.value;
            balances[payload.recipient] = toBalance;
            totalSupply.value = newSupply;

            // Announce for monitors
            announce eMintResponse, (success = true, receipt = default(MintReceipt));
            send payload.caller, eMintResponse, (success = true, receipt = default(MintReceipt));
        }

        // Burning handling
        on eBurnRequest do (payload: (from: Address, amount: BigInt, caller: machine)) {
            var fromBalance: TokenBalance;

            if (payload.from in balances) {
                fromBalance = balances[payload.from];
            } else {
                send payload.caller, eBurnFailed, "No balance to burn";
                return;
            }

            if (fromBalance.available.value < payload.amount.value) {
                send payload.caller, eBurnFailed, "Insufficient balance to burn";
                return;
            }

            // Execute burn
            fromBalance.available.value = fromBalance.available.value - payload.amount.value;
            balances[payload.from] = fromBalance;
            totalSupply.value = totalSupply.value - payload.amount.value;

            send payload.caller, eBurnResponse, (success = true, amount = payload.amount);
        }

        // Balance query
        on eQueryBalance do (payload: (address: Address, caller: machine)) {
            var balance: TokenBalance;

            if (payload.address in balances) {
                balance = balances[payload.address];
            } else {
                balance = (owner = payload.address, available = (value = 0, isNegative = false),
                          staked = (value = 0, isNegative = false), locked = (value = 0, isNegative = false));
            }

            send payload.caller, eBalanceResponse, balance;
        }

        on eNetworkShutdown do {
            goto Halted;
        }

        on eEmergencyHalt do {
            transfersEnabled = false;
            goto Halted;
        }
    }

    state Halted {
        on eEmergencyResume do {
            transfersEnabled = true;
            goto Active;
        }
    }

    // Helper to get total supply for monitors
    fun GetTotalSupply(): BigInt {
        return totalSupply;
    }

    fun GetMaxSupply(): BigInt {
        return maxSupply;
    }
}