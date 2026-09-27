package knirvbaseclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Client struct {
	Addr string
	HTTP *http.Client
}

type CollectionClient struct {
	client *Client
	name   string
}

func New(addr string) *Client { return &Client{Addr: addr, HTTP: &http.Client{}} }
func (c *Client) Collection(name string) *CollectionClient {
	return &CollectionClient{client: c, name: name}
}

func (c *CollectionClient) Insert(_ context.Context, doc map[string]interface{}) (map[string]interface{}, error) {
	body, err := json.Marshal(map[string]interface{}{"collection": c.name, "document": doc})
	if err != nil {
		return nil, err
	}
	resp, err := c.client.HTTP.Post("http://"+c.client.Addr+"/document", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("knirvbase document insert: HTTP %s", resp.Status)
	}
	return doc, nil
}
func (c *Client) Append(domain string, raw [80]byte) error {
	body, _ := json.Marshal(map[string]string{"domain": domain, "bracket": base64.StdEncoding.EncodeToString(raw[:])})
	resp, err := c.HTTP.Post("http://"+c.Addr+"/append", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("knirvbase append: HTTP %s", resp.Status)
	}
	return nil
}

// SubmitNRV reads the encoder-owned v2 container and submits every bracket.
// The container format is intentionally duplicated here to preserve the
// standalone-binary boundary; keep it synchronized with pipeline/2's nrvio.
func (c *Client) SubmitNRV(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if len(data) < 8 || (string(data[:4]) != "NRV2" && string(data[:4]) != "NRV3") {
		return 0, fmt.Errorf("invalid NRV file")
	}
	legacyV2 := string(data[:4]) == "NRV2"
	metaLen := binary.LittleEndian.Uint32(data[4:8])
	start := 8 + int(metaLen)
	if start > len(data) || (len(data)-start)%80 != 0 {
		return 0, fmt.Errorf("invalid NRV bracket section")
	}
	count := 0
	for start < len(data) {
		var raw [80]byte
		copy(raw[:], data[start:start+80])
		if legacyV2 {
			raw = migrateV2Bracket(raw)
		}
		domain := domainName(binary.LittleEndian.Uint16(raw[39:41]))
		if err := c.Append(domain, raw); err != nil {
			return count, err
		}
		count++
		start += 80
	}
	return count, nil
}

// SubmitNRVOnce records a successful immutable-artifact digest. Retrying a
// completed pipeline stage therefore does not append the same brackets again.
func (c *Client) SubmitNRVOnce(path string) (count int, alreadySubmitted bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false, err
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	receiptPath := filepath.Join(framesDirectoryForArtifact(path), "nrv_submissions.json")
	receipts, err := loadSubmissionReceipts(receiptPath)
	if err != nil {
		return 0, false, err
	}
	if _, ok := receipts.Submissions[digest]; ok {
		return 0, true, nil
	}
	count, err = c.SubmitNRV(path)
	if err != nil {
		return count, false, err
	}
	if receipts.Submissions == nil {
		receipts.Submissions = make(map[string]submissionReceipt)
	}
	receipts.Submissions[digest] = submissionReceipt{Artifact: filepath.Base(path), Count: count, SubmittedAt: time.Now().UTC()}
	if err := saveSubmissionReceipts(receiptPath, receipts); err != nil {
		return count, false, err
	}
	return count, false, nil
}

type submissionReceipt struct {
	Artifact    string    `json:"artifact"`
	Count       int       `json:"count"`
	SubmittedAt time.Time `json:"submitted_at"`
}
type submissionReceipts struct {
	Version     int                          `json:"version"`
	Submissions map[string]submissionReceipt `json:"submissions"`
}

func framesDirectoryForArtifact(path string) string {
	dir := filepath.Dir(path)
	if filepath.Base(filepath.Dir(dir)) == "batches" {
		return filepath.Dir(filepath.Dir(dir))
	}
	return dir
}
func loadSubmissionReceipts(path string) (submissionReceipts, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return submissionReceipts{Version: 1, Submissions: make(map[string]submissionReceipt)}, nil
	}
	if err != nil {
		return submissionReceipts{}, err
	}
	var receipts submissionReceipts
	if err := json.Unmarshal(data, &receipts); err != nil || receipts.Version != 1 {
		if err == nil {
			err = fmt.Errorf("unsupported submission receipt format")
		}
		return submissionReceipts{}, err
	}
	if receipts.Submissions == nil {
		receipts.Submissions = make(map[string]submissionReceipt)
	}
	return receipts, nil
}
func saveSubmissionReceipts(path string, receipts submissionReceipts) error {
	data, err := json.MarshalIndent(receipts, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".nrv-submissions-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// migrateV2Bracket translates the pre-canonical encoder layout to the public
// KNIRVBASE 80-byte layout before it crosses the network boundary.  Old
// artifacts remain readable, but all newly submitted records are canonical.
func migrateV2Bracket(old [80]byte) [80]byte {
	var canonical [80]byte
	copy(canonical[0:36], old[0:36])
	canonical[36] = (old[36] & 0x0f) | ((old[37] & 0x03) << 4) | ((old[38] & 0x03) << 6)
	canonical[37] = old[39]
	canonical[38] = old[40]
	copy(canonical[39:41], old[41:43])
	copy(canonical[41:45], old[43:47])
	copy(canonical[45:59], old[47:61])
	copy(canonical[59:63], old[61:65])
	copy(canonical[63:78], old[65:80])
	return canonical
}
func domainName(sig uint16) string {
	switch sig & 0xf000 {
	case 0x2000:
		return "math"
	case 0x3000:
		return "code"
	case 0x4000:
		return "academic"
	default:
		return "prose"
	}
}
func ResolveNRV(dataPath string) string {
	framesDir := filepath.Join(dataPath, "frames")
	if path := resolveBatchArtifact(framesDir, "nrv"); path != "" {
		return path
	}
	legacyPath := filepath.Join(framesDir, "training_frames.nrv")
	manifestPath := legacyPath + ".latest.json"
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return legacyPath
	}
	var manifest struct {
		Artifact string `json:"artifact"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return legacyPath
	}
	// A manifest contains a filename, never an arbitrary path. This prevents a
	// corrupt manifest from making the seeder submit a file outside framesDir.
	if manifest.Artifact == "" || filepath.Base(manifest.Artifact) != manifest.Artifact || filepath.Ext(manifest.Artifact) != ".nrv" {
		return legacyPath
	}
	artifactPath := filepath.Join(framesDir, manifest.Artifact)
	if _, err := os.Stat(artifactPath); err == nil {
		return artifactPath
	}
	return legacyPath
}

// ResolveTrainingFrames selects the JSON artifact in the current immutable
// batch. The bool is false when a legacy fallback must be considered.
func ResolveTrainingFrames(dataPath string) (string, bool) {
	path := resolveBatchArtifact(filepath.Join(dataPath, "frames"), "json")
	return path, path != ""
}

func resolveBatchArtifact(framesDir, kind string) string {
	data, err := os.ReadFile(filepath.Join(framesDir, "latest.json"))
	if err != nil {
		return ""
	}
	var manifest struct {
		Version   int               `json:"version"`
		BatchID   string            `json:"batch_id"`
		Artifacts map[string]string `json:"artifacts"`
	}
	if json.Unmarshal(data, &manifest) != nil || manifest.Version != 1 || manifest.BatchID == "" {
		return ""
	}
	name := manifest.Artifacts[kind]
	if name == "" || filepath.Base(name) != name {
		return ""
	}
	path := filepath.Join(framesDir, "batches", manifest.BatchID, name)
	if info, err := os.Stat(path); err == nil && !info.IsDir() && info.Size() > 0 {
		return path
	}
	return ""
}
