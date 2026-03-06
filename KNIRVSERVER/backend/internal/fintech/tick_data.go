package fintech

import (
	"fmt"
	"time"

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/array"
	"github.com/apache/arrow/go/v14/arrow/memory"
)

const (
	TickDataFlightPath  = "/fintech/tickdata"
	BarDataFlightPath   = "/fintech/bardata"
	QuoteDataFlightPath = "/fintech/quotes"
)

type TickDataType string

const (
	TickTypeTrade TickDataType = "TRADE"
	TickTypeQuote TickDataType = "QUOTE"
	TickTypeBid   TickDataType = "BID"
	TickTypeAsk   TickDataType = "ASK"
)

type TickData struct {
	Symbol     string
	Timestamp  time.Time
	Price      float64
	Volume     float64
	TickType   TickDataType
	Exchange   string
	Bid        float64
	Ask        float64
	BidSize    float64
	AskSize    float64
	Conditions string
	TradeID    string
	Sequence   uint64
}

type BarData struct {
	Symbol       string
	Timestamp    time.Time
	Interval     string
	Open         float64
	High         float64
	Low          float64
	Close        float64
	Volume       float64
	VWAP         float64
	Trades       int64
	OpenInterest float64
}

var TickDataSchema = arrow.NewSchema([]arrow.Field{
	{Name: "symbol", Type: &arrow.StringType{}, Nullable: false},
	{Name: "timestamp", Type: arrow.PrimitiveTypes.Int64, Nullable: false},
	{Name: "price", Type: arrow.PrimitiveTypes.Float64, Nullable: false},
	{Name: "volume", Type: arrow.PrimitiveTypes.Float64, Nullable: false},
	{Name: "tick_type", Type: &arrow.StringType{}, Nullable: false},
	{Name: "exchange", Type: &arrow.StringType{}, Nullable: true},
	{Name: "bid", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
	{Name: "ask", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
	{Name: "bid_size", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
	{Name: "ask_size", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
	{Name: "conditions", Type: &arrow.StringType{}, Nullable: true},
	{Name: "trade_id", Type: &arrow.StringType{}, Nullable: true},
	{Name: "sequence", Type: arrow.PrimitiveTypes.Uint64, Nullable: false},
}, nil)

var BarDataSchema = arrow.NewSchema([]arrow.Field{
	{Name: "symbol", Type: &arrow.StringType{}, Nullable: false},
	{Name: "timestamp", Type: arrow.PrimitiveTypes.Int64, Nullable: false},
	{Name: "interval", Type: &arrow.StringType{}, Nullable: false},
	{Name: "open", Type: arrow.PrimitiveTypes.Float64, Nullable: false},
	{Name: "high", Type: arrow.PrimitiveTypes.Float64, Nullable: false},
	{Name: "low", Type: arrow.PrimitiveTypes.Float64, Nullable: false},
	{Name: "close", Type: arrow.PrimitiveTypes.Float64, Nullable: false},
	{Name: "volume", Type: arrow.PrimitiveTypes.Float64, Nullable: false},
	{Name: "vwap", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
	{Name: "trades", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
	{Name: "open_interest", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
}, nil)

type TickDataWriter struct {
	pool memory.Allocator
}

func NewTickDataWriter() *TickDataWriter {
	return &TickDataWriter{
		pool: memory.DefaultAllocator,
	}
}

func (w *TickDataWriter) BuildRecord(ticks []TickData) (arrow.Record, error) {
	if len(ticks) == 0 {
		return nil, fmt.Errorf("no tick data to build")
	}

	symbols := array.NewStringBuilder(w.pool)
	prices := array.NewFloat64Builder(w.pool)
	volumes := array.NewFloat64Builder(w.pool)
	tickTypes := array.NewStringBuilder(w.pool)
	exchanges := array.NewStringBuilder(w.pool)
	bids := array.NewFloat64Builder(w.pool)
	asks := array.NewFloat64Builder(w.pool)
	bidSizes := array.NewFloat64Builder(w.pool)
	askSizes := array.NewFloat64Builder(w.pool)
	conditions := array.NewStringBuilder(w.pool)
	tradeIDs := array.NewStringBuilder(w.pool)
	sequences := array.NewUint64Builder(w.pool)
	timestamps := array.NewInt64Builder(w.pool)

	for _, tick := range ticks {
		symbols.Append(tick.Symbol)
		timestamps.Append(tick.Timestamp.UnixNano())
		prices.Append(tick.Price)
		volumes.Append(tick.Volume)
		tickTypes.Append(string(tick.TickType))
		if tick.Exchange != "" {
			exchanges.Append(tick.Exchange)
		} else {
			exchanges.AppendNull()
		}
		if tick.Bid > 0 {
			bids.Append(tick.Bid)
		} else {
			bids.AppendNull()
		}
		if tick.Ask > 0 {
			asks.Append(tick.Ask)
		} else {
			asks.AppendNull()
		}
		if tick.BidSize > 0 {
			bidSizes.Append(tick.BidSize)
		} else {
			bidSizes.AppendNull()
		}
		if tick.AskSize > 0 {
			askSizes.Append(tick.AskSize)
		} else {
			askSizes.AppendNull()
		}
		if tick.Conditions != "" {
			conditions.Append(tick.Conditions)
		} else {
			conditions.AppendNull()
		}
		if tick.TradeID != "" {
			tradeIDs.Append(tick.TradeID)
		} else {
			tradeIDs.AppendNull()
		}
		sequences.Append(tick.Sequence)
	}

	columns := []arrow.Array{
		symbols.NewArray(),
		timestamps.NewArray(),
		prices.NewArray(),
		volumes.NewArray(),
		tickTypes.NewArray(),
		exchanges.NewArray(),
		bids.NewArray(),
		asks.NewArray(),
		bidSizes.NewArray(),
		askSizes.NewArray(),
		conditions.NewArray(),
		tradeIDs.NewArray(),
		sequences.NewArray(),
	}

	record := array.NewRecord(TickDataSchema, columns, int64(len(ticks)))
	return record, nil
}

func (w *TickDataWriter) BuildBarRecord(bars []BarData) (arrow.Record, error) {
	if len(bars) == 0 {
		return nil, fmt.Errorf("no bar data to build")
	}

	symbols := array.NewStringBuilder(w.pool)
	intervals := array.NewStringBuilder(w.pool)
	opens := array.NewFloat64Builder(w.pool)
	highs := array.NewFloat64Builder(w.pool)
	lows := array.NewFloat64Builder(w.pool)
	closes := array.NewFloat64Builder(w.pool)
	volumes := array.NewFloat64Builder(w.pool)
	vwaps := array.NewFloat64Builder(w.pool)
	trades := array.NewInt64Builder(w.pool)
	openInterests := array.NewFloat64Builder(w.pool)
	timestamps := array.NewInt64Builder(w.pool)

	for _, bar := range bars {
		symbols.Append(bar.Symbol)
		timestamps.Append(bar.Timestamp.UnixNano())
		intervals.Append(bar.Interval)
		opens.Append(bar.Open)
		highs.Append(bar.High)
		lows.Append(bar.Low)
		closes.Append(bar.Close)
		volumes.Append(bar.Volume)
		if bar.VWAP > 0 {
			vwaps.Append(bar.VWAP)
		} else {
			vwaps.AppendNull()
		}
		if bar.Trades > 0 {
			trades.Append(bar.Trades)
		} else {
			trades.AppendNull()
		}
		if bar.OpenInterest > 0 {
			openInterests.Append(bar.OpenInterest)
		} else {
			openInterests.AppendNull()
		}
	}

	columns := []arrow.Array{
		symbols.NewArray(),
		timestamps.NewArray(),
		intervals.NewArray(),
		opens.NewArray(),
		highs.NewArray(),
		lows.NewArray(),
		closes.NewArray(),
		volumes.NewArray(),
		vwaps.NewArray(),
		trades.NewArray(),
		openInterests.NewArray(),
	}

	record := array.NewRecord(BarDataSchema, columns, int64(len(bars)))
	return record, nil
}

func (w *TickDataWriter) Release() {
}

type TickDataReader struct {
	pool memory.Allocator
}

func NewTickDataReader() *TickDataReader {
	return &TickDataReader{
		pool: memory.DefaultAllocator,
	}
}

func (r *TickDataReader) ReadTicks(record arrow.Record) ([]TickData, error) {
	ticks := make([]TickData, 0, record.NumRows())

	symbols := record.Column(0).(*array.String)
	timestamps := record.Column(1).(*array.Int64)
	prices := record.Column(2).(*array.Float64)
	volumes := record.Column(3).(*array.Float64)
	tickTypes := record.Column(4).(*array.String)
	exchanges := record.Column(5).(*array.String)
	bids := record.Column(6).(*array.Float64)
	asks := record.Column(7).(*array.Float64)
	bidSizes := record.Column(8).(*array.Float64)
	askSizes := record.Column(9).(*array.Float64)
	conditions := record.Column(10).(*array.String)
	tradeIDs := record.Column(11).(*array.String)
	sequences := record.Column(12).(*array.Uint64)

	for i := 0; i < int(record.NumRows()); i++ {
		tick := TickData{
			Symbol:    symbols.Value(i),
			Timestamp: time.Unix(0, timestamps.Value(i)),
			Price:     prices.Value(i),
			Volume:    volumes.Value(i),
			TickType:  TickDataType(tickTypes.Value(i)),
			Sequence:  sequences.Value(i),
		}

		if !exchanges.IsNull(i) {
			tick.Exchange = exchanges.Value(i)
		}
		if !bids.IsNull(i) {
			tick.Bid = bids.Value(i)
		}
		if !asks.IsNull(i) {
			tick.Ask = asks.Value(i)
		}
		if !bidSizes.IsNull(i) {
			tick.BidSize = bidSizes.Value(i)
		}
		if !askSizes.IsNull(i) {
			tick.AskSize = askSizes.Value(i)
		}
		if !conditions.IsNull(i) {
			tick.Conditions = conditions.Value(i)
		}
		if !tradeIDs.IsNull(i) {
			tick.TradeID = tradeIDs.Value(i)
		}

		ticks = append(ticks, tick)
	}

	return ticks, nil
}

func (r *TickDataReader) ReadBars(record arrow.Record) ([]BarData, error) {
	bars := make([]BarData, 0, record.NumRows())

	symbols := record.Column(0).(*array.String)
	timestamps := record.Column(1).(*array.Int64)
	intervals := record.Column(2).(*array.String)
	opens := record.Column(3).(*array.Float64)
	highs := record.Column(4).(*array.Float64)
	lows := record.Column(5).(*array.Float64)
	closes := record.Column(6).(*array.Float64)
	volumes := record.Column(7).(*array.Float64)
	vwaps := record.Column(8).(*array.Float64)
	trades := record.Column(9).(*array.Int64)
	openInterests := record.Column(10).(*array.Float64)

	for i := 0; i < int(record.NumRows()); i++ {
		bar := BarData{
			Symbol:    symbols.Value(i),
			Timestamp: time.Unix(0, timestamps.Value(i)),
			Interval:  intervals.Value(i),
			Open:      opens.Value(i),
			High:      highs.Value(i),
			Low:       lows.Value(i),
			Close:     closes.Value(i),
			Volume:    volumes.Value(i),
		}

		if !vwaps.IsNull(i) {
			bar.VWAP = vwaps.Value(i)
		}
		if !trades.IsNull(i) {
			bar.Trades = trades.Value(i)
		}
		if !openInterests.IsNull(i) {
			bar.OpenInterest = openInterests.Value(i)
		}

		bars = append(bars, bar)
	}

	return bars, nil
}

func (r *TickDataReader) Release() {
}

type TickStreamManager struct {
	streams  map[string]*TickDataStream
	writer   *TickDataWriter
	reader   *TickDataReader
	sequence uint64
}

type TickDataStream struct {
	Symbol   string
	TickType string
	Data     []TickData
	MaxSize  int
	OnFlush  func([]TickData) error
}

func NewTickStreamManager() *TickStreamManager {
	return &TickStreamManager{
		streams:  make(map[string]*TickDataStream),
		writer:   NewTickDataWriter(),
		reader:   NewTickDataReader(),
		sequence: 0,
	}
}

func (m *TickStreamManager) CreateStream(symbol string, tickType string, maxSize int, onFlush func([]TickData) error) *TickDataStream {
	stream := &TickDataStream{
		Symbol:   symbol,
		TickType: tickType,
		Data:     make([]TickData, 0, maxSize),
		MaxSize:  maxSize,
		OnFlush:  onFlush,
	}
	m.streams[symbol] = stream
	return stream
}

func (m *TickStreamManager) GetStream(symbol string) (*TickDataStream, bool) {
	stream, ok := m.streams[symbol]
	return stream, ok
}

func (m *TickStreamManager) AddTick(streamSymbol string, tick TickData) error {
	stream, ok := m.streams[streamSymbol]
	if !ok {
		return fmt.Errorf("stream not found: %s", streamSymbol)
	}

	m.sequence++
	tick.Sequence = m.sequence
	stream.Data = append(stream.Data, tick)

	if len(stream.Data) >= stream.MaxSize && stream.OnFlush != nil {
		if err := stream.OnFlush(stream.Data); err != nil {
			return err
		}
		stream.Data = stream.Data[:0]
	}

	return nil
}

func (m *TickStreamManager) FlushStream(symbol string) error {
	stream, ok := m.streams[symbol]
	if !ok {
		return fmt.Errorf("stream not found: %s", symbol)
	}

	if len(stream.Data) > 0 && stream.OnFlush != nil {
		if err := stream.OnFlush(stream.Data); err != nil {
			return err
		}
		stream.Data = stream.Data[:0]
	}

	return nil
}

func (m *TickStreamManager) BuildRecord(symbol string) (arrow.Record, error) {
	stream, ok := m.streams[symbol]
	if !ok {
		return nil, fmt.Errorf("stream not found: %s", symbol)
	}

	return m.writer.BuildRecord(stream.Data)
}

func (m *TickStreamManager) DeleteStream(symbol string) {
	delete(m.streams, symbol)
}

func (m *TickStreamManager) Release() {
	m.writer.Release()
	m.reader.Release()
}

func (m *TickStreamManager) GetAllStreams() map[string]*TickDataStream {
	return m.streams
}

type TickDataFilter struct {
	Symbol    string
	StartTime int64
	EndTime   int64
	TickTypes []TickDataType
	MinPrice  float64
	MaxPrice  float64
	MinVolume float64
	MaxVolume float64
	Limit     int
	Offset    int
}
