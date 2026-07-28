package knirvbaseclient

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

type Client struct {
	Addr string
	HTTP *http.Client
}

func New(addr string) *Client { return &Client{Addr: addr, HTTP: &http.Client{}} }
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
	if len(data) < 8 || string(data[:4]) != "NRV2" {
		return 0, fmt.Errorf("invalid NRV file")
	}
	metaLen := binary.LittleEndian.Uint32(data[4:8])
	start := 8 + int(metaLen)
	if start > len(data) || (len(data)-start)%80 != 0 {
		return 0, fmt.Errorf("invalid NRV bracket section")
	}
	count := 0
	for start < len(data) {
		var raw [80]byte
		copy(raw[:], data[start:start+80])
		domain := domainName(binary.LittleEndian.Uint16(raw[41:43]))
		if err := c.Append(domain, raw); err != nil {
			return count, err
		}
		count++
		start += 80
	}
	return count, nil
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
	return filepath.Join(dataPath, "frames", "training_frames.nrv")
}
