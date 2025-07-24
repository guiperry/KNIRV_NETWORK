package consensus

import (
    "blockchain-app/internal/types"
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/tendermint/tendermint/abci/types"
    "github.com/tendermint/tendermint/libs/log"
    "github.com/tendermint/tendermint/node"
    "github.com/tendermint/tendermint/p2p"
    "github.com/tendermint/tendermint/privval"
    "github.com/tendermint/tendermint/proxy"
    tmtypes "github.com/tendermint/tendermint/types"
)

type TendermintConsensus struct {
    app    *BlockchainApp
    node   *node.Node
    logger log.Logger
}

type BlockchainApp struct {
    state  *types.State
    logger log.Logger
}

func NewTendermintConsensus(homeDir string, logger log.Logger) (*TendermintConsensus, error) {
    app := &BlockchainApp{
        state:  types.NewState(),
        logger: logger,
    }
    
    // Load node configuration
    config := node.DefaultConfig()
    config.SetRoot(homeDir)
    
    // Create private validator
    privValidator := privval.LoadOrGenFilePV(
        config.PrivValidatorKeyFile(),
        config.PrivValidatorStateFile(),
    )
    
    // Create node key
    nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
    if err != nil {
        return nil, fmt.Errorf("failed to load node key: %w", err)
    }
    
    // Create ABCI client
    proxyApp := proxy.NewLocalClientCreator(app)
    
    // Create node
    tmNode, err := node.NewNode(
        config,
        privValidator,
        nodeKey,
        proxyApp,
        node.DefaultGenesisDocProviderFunc(config),
        node.DefaultDBProvider,
        node.DefaultMetricsProvider(config.Instrumentation),
        logger,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create tendermint node: %w", err)
    }
    
    return &TendermintConsensus{
        app:    app,
        node:   tmNode,
        logger: logger,
    }, nil
}

func (tc *TendermintConsensus) Start(ctx context.Context) error {
    return tc.node.Start()
}

func (tc *TendermintConsensus) Stop() error {
    return tc.node.Stop()
}

// ABCI Application Methods
func (app *BlockchainApp) Info(req types.RequestInfo) types.ResponseInfo {
    return types.ResponseInfo{
        Data:             "Blockchain App",
        Version:          "1.0.0",
        AppVersion:       1,
        LastBlockHeight:  int64(app.state.Height),
        LastBlockAppHash: []byte{},
    }
}

func (app *BlockchainApp) CheckTx(req types.RequestCheckTx) types.ResponseCheckTx {
    var tx types.Transaction
    if err := json.Unmarshal(req.Tx, &tx); err != nil {
        return types.ResponseCheckTx{
            Code: 1,
            Log:  fmt.Sprintf("invalid transaction format: %v", err),
        }
    }
    
    if !tx.Verify() {
        return types.ResponseCheckTx{
            Code: 2,
            Log:  "invalid transaction signature",
        }
    }
    
    return types.ResponseCheckTx{
        Code: 0,
        Log:  "transaction valid",
    }
}

func (app *BlockchainApp) DeliverTx(req types.RequestDeliverTx) types.ResponseDeliverTx {
    var tx types.Transaction
    if err := json.Unmarshal(req.Tx, &tx); err != nil {
        return types.ResponseDeliverTx{
            Code: 1,
            Log:  fmt.Sprintf("invalid transaction format: %v", err),
        }
    }
    
    // Execute transaction
    if err := app.executeTransaction(&tx); err != nil {
        return types.ResponseDeliverTx{
            Code: 2,
            Log:  fmt.Sprintf("failed to execute transaction: %v", err),
        }
    }
    
    return types.ResponseDeliverTx{
        Code: 0,
        Log:  "transaction executed successfully",
    }
}

func (app *BlockchainApp) BeginBlock(req types.RequestBeginBlock) types.ResponseBeginBlock {
    app.logger.Info("Beginning block", "height", req.Header.Height)
    return types.ResponseBeginBlock{}
}

func (app *BlockchainApp) EndBlock(req types.RequestEndBlock) types.ResponseEndBlock {
    app.logger.Info("Ending block", "height", req.Height)
    return types.ResponseEndBlock{}
}

func (app *BlockchainApp) Commit() types.ResponseCommit {
    app.state.Height++
    stateData, _ := app.state.Serialize()
    
    // In production, compute proper app hash
    appHash := tmtypes.Tx(stateData).Hash()
    
    return types.ResponseCommit{
        Data: appHash,
    }
}

func (app *BlockchainApp) InitChain(req types.RequestInitChain) types.ResponseInitChain {
    app.logger.Info("Initializing chain")
    return types.ResponseInitChain{}
}

func (app *BlockchainApp) Query(req types.RequestQuery) types.ResponseQuery {
    switch req.Path {
    case "/account":
        account := app.state.GetAccount(string(req.Data))
        data, _ := json.Marshal(account)
        return types.ResponseQuery{
            Code:  0,
            Key:   req.Data,
            Value: data,
        }
    default:
        return types.ResponseQuery{
            Code: 1,
            Log:  "unknown query path",
        }
    }
}

func (app *BlockchainApp) executeTransaction(tx *types.Transaction) error {
    if tx.To != "" {
        return app.state.Transfer(tx.From, tx.To, tx.Amount)
    }
    return nil
}