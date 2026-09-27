// supervision-advisor serves a pinned local retrieval index over loopback.
package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"knirvhasher/pkg/embeddings"
	"knirvhasher/pkg/hashing/supervision"
)

func main() {
	indexPath := flag.String("index", "", "verified supervision index")
	listen := flag.String("listen", "127.0.0.1:8787", "loopback listener")
	model := flag.String("model-version", "supervision-retrieval-v1", "model version")
	flag.Parse()
	if *indexPath == "" {
		fail("--index is required")
	}
	host, _, err := net.SplitHostPort(*listen)
	if err != nil || (host != "127.0.0.1" && host != "::1" && host != "localhost") {
		fail("--listen must bind loopback")
	}
	idx, err := supervision.LoadIndex(*indexPath)
	if err != nil {
		fail(err.Error())
	}
	r := supervision.NewRetriever(idx, func(text string) []float32 { return embeddings.NewDeterministicService().GetEmbedding(text) }, *model)
	fmt.Fprintf(os.Stderr, "supervision adviser: manifest=%s policy=%s listen=%s\n", idx.DatasetManifestID, idx.PolicyBundleHash, *listen)
	if err := http.ListenAndServe(*listen, supervision.NewHTTPHandler(r, *model, idx.DatasetManifestID)); err != nil {
		fail(err.Error())
	}
}
func fail(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(2) }
