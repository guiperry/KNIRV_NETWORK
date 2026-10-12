// nrvconform is the Go half of the cross-implementation conformance suite
// (KNIRVBASE/conformance, ts/src/conformance). It writes and dumps .nrv files
// with the canonical Go implementation so the TypeScript port can be checked
// against it byte for byte.
//
//	nrvconform write <file>   write the fixed fixture (3 frames of brackets)
//	nrvconform dump  <file>   print registry + decoded brackets as JSON
//	nrvconform write-signed <file> <keyfile>   like write, signing each frame
//	nrvconform verify-frame <file> <keyfile> <frameId>
//	nrvconform parse <query...>                 print the parsed KNIRVQL query as JSON
//	nrvconform query                            stdin {docs,indexes,queries} -> per-query results
//	nrvconform ticker-write <file>              deterministic FrameTicker run (I/P deltas, drift spikes)
//	nrvconform write-nrv3 <file> [count...]     hasher-shaped NRV3 fixture, one frame per count (default 150 40)
//	nrvconform convert-nrv3 <in.nrv3> <out.nrv> [keyfile]   NRV3 -> KNIRVBASE; prints the files written
//	                                            (shards beyond the first are <out>.1.nrv, ...); keyfile signs frames (E18)
//	nrvconform wal-recover <walfile>            print WAL.Recover()
//	nrvconform p2p secret <net> <secret> <payload>      ServiceID/Topic/HMAC as JSON
//	nrvconform p2p post-socket <sock> <path> <net> <secret> <json>
//	nrvconform p2p socket-serve <sock> <net> <secret>   print received ops until stdin closes
//	nrvconform p2p node <port> <net> <secret> [peer..]  standalone node driven over stdin/stdout
//	nrvconform flight encode|decode|drift      Arrow IPC (BracketsToFlightData) interop
//	nrvconform pqc keygen
//	nrvconform pqc <encrypt|decrypt|sign|verify|envelope-encrypt|envelope-decrypt> <keyfile> <b64> [sig-b64]
//
// The pqc commands exchange base64 on stdout so TypeScript can check keys,
// KEM+AES-GCM ciphertexts, signatures and the signed envelope against Go.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/knirvcorp/knirvbase/internal/p2pconsensus"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/knirvcorp/knirvbase/internal/crypto/pqc"
	dbpkg "github.com/knirvcorp/knirvbase/internal/database"
	netpkg "github.com/knirvcorp/knirvbase/internal/network"
	"github.com/knirvcorp/knirvbase/internal/query"
	"github.com/knirvcorp/knirvbase/internal/storage"
	stor "github.com/knirvcorp/knirvbase/internal/storage"
	"github.com/knirvcorp/knirvbase/pkg/nrv"
)

func fixtureBracket(i int) nrv.Bracket {
	var b nrv.Bracket
	for j := range b.Projections {
		b.Projections[j] = byte(i*31 + j)
	}
	b.SubSecondUS = uint32(1000 + i)
	b.Syntactic = nrv.PackSyntacticByte(uint8(i+1), uint8(i%4), uint8((i+1)%4))
	b.DepHead = int8(-3 * (i + 1)) // negative on purpose: exercises sign handling
	b.IntentFlags = uint8(i + 7)
	b.DomainSig = uint16(0xA000 + i)
	b.GoldenSeed = uint32(0xDEADBE00 + i)
	for j := range b.Memory {
		b.Memory[j] = byte(100 + i + j)
	}
	b.LSHSalt = uint32(i<<16 | 0x1234)
	return b
}

func write(path string, kp *pqc.PQCKeyPair) error {
	w, err := storage.NewNRVWriter(path, kp)
	if err != nil {
		return err
	}
	defer w.Close()
	for f := 0; f < 3; f++ {
		var buf []byte
		var idx []nrv.BracketMeta
		for k := 0; k < 2; k++ {
			b := fixtureBracket(f*2 + k)
			enc := nrv.EncodeBracket(&b)
			buf = append(buf, enc[:]...)
			idx = append(idx, nrv.BracketMeta{
				ID: fmt.Sprintf("b-%d-%d", f, k), Type: nrv.DeltaType("I"),
				Offset: k * nrv.BracketSize, DriftScore: 0.25 * float64(k),
			})
		}
		err := w.AppendFrame(fmt.Sprintf("frame-%d", f), buf, idx,
			nrv.ThermoAtmosphere{AvgTempC: 60 + float32(f), PeakVoltV: 1.1, ClockMHz: 500},
			nrv.LinguisticMapping{Token: "tok", Unit: "u"})
		if err != nil {
			return err
		}
	}
	return nil
}

// nrv3Bracket is shared with the TS conformance test (see nrv3Bracket there):
// slowly drifting projections that hop hard every 90th bracket.
func nrv3Bracket(n int) nrv.Bracket {
	var b nrv.Bracket
	for j := range b.Projections {
		b.Projections[j] = byte(10+j) + byte(n%3)
		if n%90 == 89 {
			b.Projections[j] = byte(255 - 7*j)
		}
	}
	b.GoldenSeed = uint32(n * 2654435761)
	b.SubSecondUS = uint32(n)
	b.DepHead = int8(n%100 - 50)
	b.Memory[0] = byte(n)
	b.LSHSalt = uint32(n << 8)
	return b
}

// writeNrv3 writes a hasher-shaped NRV3 file: full brackets in the bracket
// section and, per bracket, an "xor" blob holding the bracket XOR the previous one.
func writeNrv3(path string, counts []int) error {
	fw := nrv.NewNRV3Writer()
	n, off := 0, int64(0)
	for fi, count := range counts {
		f := nrv.NRV3FrameEntry{
			ID: fmt.Sprintf("frame-%d", fi), TimestampUnix: 1700000000 + int64(fi), Count: count, Offset: off,
			Linguistic: nrv.NRV3LinguisticMetadata{Tokens: 3 + fi, Domain: "prose"},
			Thermo:     nrv.NRV3ThermoMetadata{Temperature: 65.25},
		}
		var buf []byte
		var prev [nrv.BracketSize]byte
		for k := 0; k < count; k++ {
			b := nrv3Bracket(n)
			n++
			enc := nrv.EncodeBracket(&b)
			xor, typ := make([]byte, nrv.BracketSize), "P"
			for j := range xor {
				xor[j] = enc[j] ^ prev[j]
			}
			if k == 0 {
				typ = "I"
				copy(xor, enc[:])
			}
			prev = enc
			f.BracketIndex = append(f.BracketIndex, nrv.NRV3BracketIndexEntry{Index: k, Type: typ, XOR: xor})
			buf = append(buf, enc[:]...)
		}
		fw.AddFrame(f, buf)
		off += int64(count) * nrv.BracketSize
	}
	return nrv.WriteNRV3(path, fw)
}

func dump(path string) error {
	r, err := storage.NewNRVReader(path)
	if err != nil {
		return err
	}
	defer r.Close()
	reg := r.GetRegistry()
	out := map[string]any{"registry": reg}
	brackets := map[string][]*nrv.Bracket{}
	for _, e := range reg.Frames {
		_, bs, err := r.GetFrame(e.ID)
		if err != nil {
			return err
		}
		brackets[e.ID] = bs
	}
	out["brackets"] = brackets
	return json.NewEncoder(os.Stdout).Encode(out)
}

func pqcCmd(args []string) error {
	if len(args) == 1 && args[0] == "keygen" {
		kp, err := pqc.GeneratePQCKeyPair("conform", "encryption")
		if err != nil {
			return err
		}
		out, err := kp.MarshalWithPrivateKeys()
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}
	if len(args) < 3 {
		return fmt.Errorf("usage: pqc <op> <keyfile> <b64> [sig-b64]")
	}
	raw, err := os.ReadFile(args[1])
	if err != nil {
		return err
	}
	kp, err := pqc.LoadPQCKeyPair(raw)
	if err != nil {
		return err
	}
	arg, err := base64.StdEncoding.DecodeString(args[2])
	if err != nil && args[0] != "envelope-decrypt" {
		return err
	}
	em := pqc.NewEncryptionManager()
	em.SetMasterKey(kp)
	var out []byte
	switch args[0] {
	case "encrypt":
		out, err = kp.Encrypt(arg)
	case "decrypt":
		out, err = kp.Decrypt(arg)
	case "sign":
		out, err = kp.Sign(arg)
	case "verify":
		sig, derr := base64.StdEncoding.DecodeString(args[3])
		if derr != nil {
			return derr
		}
		fmt.Println(kp.Verify(arg, sig))
		return nil
	case "envelope-encrypt":
		var enc string
		enc, err = em.EncryptData(arg, kp.ID)
		out = []byte(enc)
		fmt.Println(enc)
		return err
	case "envelope-decrypt":
		out, err = em.DecryptData(args[2])
	default:
		return fmt.Errorf("unknown pqc op %q", args[0])
	}
	if err != nil {
		return err
	}
	fmt.Println(base64.StdEncoding.EncodeToString(out))
	return nil
}

// tickerBracket is shared with the TS port: i drives every field.
func tickerBracket(i int) *nrv.Bracket {
	b := &nrv.Bracket{ID: fmt.Sprintf("t-%03d", i)}
	for j := range b.Projections {
		b.Projections[j] = byte(10 + (i % 3) + j)
		if i%40 == 39 { // drift spike
			b.Projections[j] = byte(255 - j*7)
		}
	}
	b.SubSecondUS = uint32(5000 + i)
	b.Syntactic = nrv.PackSyntacticByte(uint8(i%15), uint8(i%4), uint8(i%3))
	b.DepHead = int8(i%20 - 10)
	b.IntentFlags = uint8(i)
	b.DomainSig = uint16(0x1000 * (i%3 + 1))
	b.GoldenSeed = uint32(0x9E3779B1 * uint32(i+1))
	b.LSHSalt = uint32(i<<16 | 0xBEEF&0xFFFF)
	return b
}

func tickerWrite(path string) error {
	w, err := storage.NewNRVWriter(path, nil)
	if err != nil {
		return err
	}
	defer w.Close()
	ft := storage.NewFrameTicker(w, time.Hour)
	ft.SetLinguistic("tok", "unit")
	for i := 0; i < 120; i++ {
		th := nrv.ThermoAtmosphere{AvgTempC: 50 + float32(i)*0.37, PeakVoltV: 1.0 + float32(i)*0.013, ClockMHz: 400 + float32(i)}
		if err := ft.AppendBracket(context.Background(), tickerBracket(i), th); err != nil {
			return err
		}
		if i%7 == 6 {
			if err := ft.Flush(); err != nil {
				return err
			}
		}
	}
	if err := ft.Flush(); err != nil {
		return err
	}
	ft.Stop()
	return nil
}

func queryJSON(q *query.Query) map[string]any {
	filters := make([]map[string]any, 0, len(q.Filters))
	for _, f := range q.Filters {
		filters = append(filters, map[string]any{"key": f.Key, "operator": f.Operator, "value": f.Value})
	}
	sim := q.SimilarTo
	if sim == nil {
		sim = []float64{}
	}
	fields := q.Fields
	if fields == nil {
		fields = []string{}
	}
	return map[string]any{
		"type": int(q.Type), "entryType": string(q.EntryType), "collection": q.Collection,
		"id": q.ID, "key": q.Key, "value": q.Value, "filters": filters, "similarTo": sim,
		"limit": q.Limit, "indexName": q.IndexName, "fields": fields, "unique": q.Unique,
		"modalityType": q.ModalityType, "bracketField": q.BracketField,
	}
}

type queryInput struct {
	Docs    []map[string]any `json:"docs"`
	Indexes []struct {
		Name   string   `json:"name"`
		Fields []string `json:"fields"`
		Unique bool     `json:"unique"`
	} `json:"indexes"`
	Queries []string `json:"queries"`
}

// queryCmd executes each query against a fresh "memory" collection holding docs.
func queryCmd() error {
	var in queryInput
	if err := json.NewDecoder(bufio.NewReader(os.Stdin)).Decode(&in); err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "nrvconform-q")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	store := stor.NewFileStorage(dir)
	database, err := dbpkg.NewDistributedDatabase(context.Background(), dbpkg.DistributedDbOptions{}, store)
	if err != nil {
		return err
	}
	defer database.Shutdown()
	coll := database.Collection("memory", store)
	ctx := context.Background()
	for _, ix := range in.Indexes {
		if err := database.CreateIndex(ctx, "memory", ix.Name, stor.IndexTypeBTree, ix.Fields, ix.Unique, "", nil); err != nil {
			return err
		}
	}
	for _, d := range in.Docs {
		if _, err := coll.Insert(ctx, d); err != nil {
			return err
		}
	}
	parser := &query.KNIRVQLParser{}
	results := []map[string]any{}
	for _, text := range in.Queries {
		q, err := parser.Parse(text)
		if err != nil {
			results = append(results, map[string]any{"error": err.Error()})
			continue
		}
		res, err := q.Execute(database, coll)
		if err != nil {
			results = append(results, map[string]any{"error": err.Error()})
			continue
		}
		results = append(results, map[string]any{"result": res})
	}
	return json.NewEncoder(os.Stdout).Encode(results)
}

type printHandler struct{ mu sync.Mutex }

func (h *printHandler) emit(v map[string]any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	_ = json.NewEncoder(os.Stdout).Encode(v)
}
func (h *printHandler) OnOperationReceived(op p2pconsensus.OperationEnvelope) error {
	h.emit(map[string]any{"event": "op", "op": op})
	return nil
}
func (h *printHandler) OnSyncRequestReceived(req p2pconsensus.SyncRequest) (*p2pconsensus.SyncResponse, error) {
	h.emit(map[string]any{"event": "sync_request", "req": req})
	return &p2pconsensus.SyncResponse{NetworkID: req.NetworkID, Collection: req.Collection, Operations: []p2pconsensus.OperationEnvelope{
		{NetworkID: req.NetworkID, Collection: req.Collection, DocumentID: "from-go", Data: []byte(`{"hello":"go"}`), VectorClock: map[string]int64{"go": 1}, Timestamp: 1, PeerID: "go"},
	}}, nil
}
func (h *printHandler) OnSyncResponseReceived(resp p2pconsensus.SyncResponse) error {
	h.emit(map[string]any{"event": "sync_response", "resp": resp})
	return nil
}
func (h *printHandler) OnPeerDiscovered(p p2pconsensus.PeerInfo) error {
	h.emit(map[string]any{"event": "peer", "peer": p.ID})
	return nil
}
func (h *printHandler) HandleOperation(op p2pconsensus.OperationEnvelope) error {
	return h.OnOperationReceived(op)
}
func (h *printHandler) HandleSyncRequest(req p2pconsensus.SyncRequest) (*p2pconsensus.SyncResponse, error) {
	return h.OnSyncRequestReceived(req)
}

// flightBracket is shared with the TS test: odd brackets have no Meta and no
// FrameUnix, exercising the SubSecondUS and "I" fallbacks.
func flightBracket(i int) *nrv.Bracket {
	b := fixtureBracket(i)
	b.ID = fmt.Sprintf("fb-%d", i)
	b.FrameID = fmt.Sprintf("ff-%d", i)
	if i%2 == 0 {
		b.FrameUnix = int64(1700000000 + i)
		t := nrv.DeltaTypeI
		var anchor *string
		if i%4 == 2 {
			t = nrv.DeltaTypeP
			a := "fb-0"
			anchor = &a
		}
		b.Meta = &nrv.BracketMeta{ID: b.ID, Type: t, AnchorID: anchor, DriftScore: 0.125 * float64(i)}
	}
	return &b
}

func flightCmd(a []string) error {
	if len(a) == 0 {
		return fmt.Errorf("usage: flight encode <n> | decode <b64> | drift <i> <j>")
	}
	switch a[0] {
	case "encode":
		n, _ := strconv.Atoi(a[1])
		var bs []*nrv.Bracket
		for i := 0; i < n; i++ {
			bs = append(bs, flightBracket(i))
		}
		data, err := netpkg.BracketsToFlightData(bs)
		if err != nil {
			return err
		}
		fmt.Println(base64.StdEncoding.EncodeToString(data))
	case "decode":
		raw, err := base64.StdEncoding.DecodeString(a[1])
		if err != nil {
			return err
		}
		bs, err := netpkg.FlightDataToBrackets(raw)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(bs)
	case "drift":
		i, _ := strconv.Atoi(a[1])
		j, _ := strconv.Atoi(a[2])
		fmt.Println(strconv.FormatFloat(netpkg.CalcBracketDriftScore(flightBracket(i), flightBracket(j)), 'g', -1, 64))
	default:
		return fmt.Errorf("unknown flight op %q", a[0])
	}
	return nil
}

func p2pCmd(a []string) error {
	if len(a) == 0 {
		return fmt.Errorf("usage: p2p secret|post-socket|socket-serve|node ...")
	}
	switch a[0] {
	case "secret":
		if len(a) != 4 {
			return fmt.Errorf("usage: p2p secret <net> <secret> <payload>")
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]string{
			"service_id": p2pconsensus.ServiceID(a[1], a[2]),
			"topic":      p2pconsensus.Topic(a[1], a[2]),
			"sig":        p2pconsensus.SignMessage(a[1], a[2], []byte(a[3])),
		})
	case "post-socket":
		if len(a) != 6 {
			return fmt.Errorf("usage: p2p post-socket <sock> <path> <net> <secret> <json>")
		}
		body := []byte(a[5])
		req, _ := http.NewRequest(http.MethodPost, "http://unix"+a[2], bytes.NewReader(body))
		if sig := p2pconsensus.SignMessage(a[3], a[4], body); sig != "" {
			req.Header.Set("X-KNIRV-Signature", sig)
		}
		resp, err := p2pconsensus.UnixSocketDialer(a[1]).Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		out, _ := io.ReadAll(resp.Body)
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"status": resp.StatusCode, "body": string(bytes.TrimSpace(out))})
	case "socket-serve":
		if len(a) != 4 {
			return fmt.Errorf("usage: p2p socket-serve <sock> <net> <secret>")
		}
		h := &printHandler{}
		srv := p2pconsensus.NewSocketServer(a[1], h, a[2], a[3])
		go func() { _ = srv.Serve() }()
		for i := 0; i < 200; i++ {
			if _, err := os.Stat(a[1]); err == nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		h.emit(map[string]any{"event": "ready"})
		_, _ = io.Copy(io.Discard, os.Stdin)
		return srv.Stop(context.Background())
	case "node":
		if len(a) < 4 {
			return fmt.Errorf("usage: p2p node <port> <net> <secret> [peer..]")
		}
		port, _ := strconv.Atoi(a[1])
		cfg := p2pconsensus.DefaultConsensusConfig()
		cfg.NetworkID, cfg.NetworkSecret, cfg.Port, cfg.ListenAddr, cfg.Mode = a[2], a[3], port, "127.0.0.1", "standalone"
		cfg.BootstrapPeers = a[4:]
		h := &printHandler{}
		sc := p2pconsensus.NewStandaloneConsensus(cfg, h)
		if err := sc.Start(context.Background()); err != nil {
			return err
		}
		h.emit(map[string]any{"event": "ready"})
		sc2 := bufio.NewScanner(os.Stdin)
		sc2.Buffer(make([]byte, 0, 1<<20), 1<<20)
		for sc2.Scan() {
			line := sc2.Text()
			switch {
			case strings.HasPrefix(line, "PUB "):
				var op p2pconsensus.OperationEnvelope
				if err := json.Unmarshal([]byte(line[4:]), &op); err != nil {
					h.emit(map[string]any{"event": "error", "error": err.Error()})
					continue
				}
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				err := sc.BroadcastOperation(ctx, op)
				cancel()
				h.emit(map[string]any{"event": "published", "error": fmt.Sprint(err)})
			case strings.HasPrefix(line, "SYNC "):
				var req p2pconsensus.SyncRequest
				if err := json.Unmarshal([]byte(line[5:]), &req); err != nil {
					continue
				}
				err := sc.RequestSync(context.Background(), req)
				h.emit(map[string]any{"event": "synced", "error": fmt.Sprint(err)})
			case line == "PEERS":
				h.emit(map[string]any{"event": "peers", "count": len(sc.Peers())})
			}
		}
		return sc.Stop()
	}
	return fmt.Errorf("unknown p2p op %q", a[0])
}

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "flight" {
		if err := flightCmd(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "nrvconform:", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "p2p" {
		if err := p2pCmd(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "nrvconform:", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) == 3 && os.Args[1] == "wal-recover" {
		n, err := storage.NewWAL(os.Args[2]).Recover()
		if err != nil {
			fmt.Fprintln(os.Stderr, "nrvconform:", err)
			os.Exit(1)
		}
		fmt.Println(n)
		return
	}
	if len(os.Args) == 3 && os.Args[1] == "ticker-write" {
		if err := tickerWrite(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, "nrvconform:", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) >= 3 && os.Args[1] == "write-nrv3" {
		counts := []int{150, 40}
		if len(os.Args) > 3 {
			counts = counts[:0]
			for _, a := range os.Args[3:] {
				c, err := strconv.Atoi(a)
				if err != nil || c < 0 {
					fmt.Fprintln(os.Stderr, "nrvconform: bad frame count", a)
					os.Exit(2)
				}
				counts = append(counts, c)
			}
		}
		if err := writeNrv3(os.Args[2], counts); err != nil {
			fmt.Fprintln(os.Stderr, "nrvconform:", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) >= 4 && os.Args[1] == "convert-nrv3" {
		var kp *pqc.PQCKeyPair
		if len(os.Args) >= 5 {
			raw, err := os.ReadFile(os.Args[4])
			if err == nil {
				kp, err = pqc.LoadPQCKeyPair(raw)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, "nrvconform:", err)
				os.Exit(1)
			}
		}
		files, err := storage.ConvertNRV3File(os.Args[2], os.Args[3], kp)
		if err != nil {
			fmt.Fprintln(os.Stderr, "nrvconform:", err)
			os.Exit(1)
		}
		for _, f := range files {
			fmt.Println(f)
		}
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "query" {
		if err := queryCmd(); err != nil {
			fmt.Fprintln(os.Stderr, "nrvconform:", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "parse" {
		q, err := (&query.KNIRVQLParser{}).Parse(strings.Join(os.Args[2:], " "))
		if err != nil {
			fmt.Println(`{"error":` + strconv.Quote(err.Error()) + `}`)
			return
		}
		json.NewEncoder(os.Stdout).Encode(queryJSON(q))
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "pqc" {
		if err := pqcCmd(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "nrvconform:", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) >= 4 && (os.Args[1] == "write-signed" || os.Args[1] == "verify-frame") {
		raw, err := os.ReadFile(os.Args[3])
		if err == nil {
			var kp *pqc.PQCKeyPair
			if kp, err = pqc.LoadPQCKeyPair(raw); err == nil {
				if os.Args[1] == "write-signed" {
					err = write(os.Args[2], kp)
				} else if len(os.Args) != 5 {
					err = fmt.Errorf("usage: verify-frame <file> <keyfile> <frameId>")
				} else {
					var r *storage.NRVReader
					if r, err = storage.NewNRVReader(os.Args[2]); err == nil {
						defer r.Close()
						var ok bool
						if ok, err = r.VerifyFrame(os.Args[4], kp); err == nil {
							fmt.Println(ok)
						}
					}
				}
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "nrvconform:", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: nrvconform write|dump <file>")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "write":
		err = write(os.Args[2], nil)
	case "dump":
		err = dump(os.Args[2])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "nrvconform:", err)
		os.Exit(1)
	}
}
