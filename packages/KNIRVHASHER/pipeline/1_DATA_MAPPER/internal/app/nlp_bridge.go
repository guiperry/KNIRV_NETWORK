package app

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/am-sokolov/go-spacy"
	"hash/fnv"
)

// NLPBridge handles linguistic metadata extraction
type NLPBridge struct {
	nlp *spacy.NLP
}

func pythonCheck() error {
	cmd := exec.Command("python3", "-c", "import spacy; spacy.load('en_core_web_sm')")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("spaCy preflight failed: command=%q exit=%v output=%q", strings.Join(cmd.Args, " "), err, string(out))
	}
	return err
}

// ensureSpacyInstalled verifies both the Python package and the model. A
// successful `import spacy` is not sufficient: the model is a separate
// install and is required for POS/tense/dependency slots.
func ensureSpacyInstalled() {
	log.Printf("spaCy environment: LD_LIBRARY_PATH=%q PYTHONPATH=%q", os.Getenv("LD_LIBRARY_PATH"), os.Getenv("PYTHONPATH"))
	if err := pythonCheck(); err == nil {
		log.Println("spaCy preflight succeeded: en_core_web_sm is available")
		return
	}
	log.Println("📦 spaCy package/model unavailable — attempting automatic installation...")
	install := exec.Command("python3", "-m", "pip", "install", "--quiet", "spacy")
	if out, err := install.CombinedOutput(); err != nil {
		log.Printf("spaCy pip install failed: command=%q exit=%v output=%q", strings.Join(install.Args, " "), err, string(out))
	} else {
		log.Println("✅ spaCy installed via pip")
	}
	download := exec.Command("python3", "-m", "spacy", "download", "--quiet", "en_core_web_sm")
	if out, err := download.CombinedOutput(); err != nil {
		log.Printf("spaCy model download failed: command=%q exit=%v output=%q", strings.Join(download.Args, " "), err, string(out))
	} else {
		log.Println("✅ en_core_web_sm model downloaded")
	}
}

// NewNLPBridge initializes the NLP Bridge with a SpaCy model.
// If SpaCy is not installed it will attempt to install it automatically.
func NewNLPBridge() (bridge *NLPBridge, err error) {
	log.Println("spaCy NLP bridge: initialization attempt 1")
	// If the first attempt fails, try to auto-install and retry once.
	for attempt := 0; attempt < 2; attempt++ {
		nlp, nlpErr := spacy.NewNLP("en_core_web_sm")
		if nlpErr == nil {
			// Quick health check
			func() {
				defer func() {
					if r := recover(); r != nil {
						nlp.Close()
						err = fmt.Errorf("spacy health check panicked: %v", r)
					}
				}()
				_ = nlp.Tokenize("test")
			}()
			if err == nil {
				return &NLPBridge{nlp: nlp}, nil
			}
			nlp.Close()
			return nil, err
		}

		// First attempt failed — auto-install and retry
		log.Printf("spaCy NLP bridge attempt %d failed: %v", attempt+1, nlpErr)
		if attempt == 0 {
			ensureSpacyInstalled()
		} else {
			log.Printf("WARN: ⚠️  spaCy NLP unavailable — proceeding WITHOUT POS/Tense/Dependency metadata (Slots 4-5 will be zero-filled): %v", nlpErr)
			return nil, nlpErr
		}
	}
	log.Println("WARN: ⚠️  spaCy NLP unavailable — proceeding WITHOUT POS/Tense/Dependency metadata (Slots 4-5 will be zero-filled)")
	return nil, fmt.Errorf("failed to initialize spacy after auto-install attempt")
}

// Close releases SpaCy resources
func (nb *NLPBridge) Close() {
	if nb.nlp != nil {
		nb.nlp.Close()
	}
}

// ProcessText extracts linguistic features from text
func (nb *NLPBridge) ProcessText(text string) (words []string, offsets []int32, posTags []int, tenses []int, depHashes []uint32) {
	// Limit text size to prevent memory issues with spaCy
	const maxTextLength = 50000
	if len(text) > maxTextLength {
		text = text[:maxTextLength]
	}

	// Recover from any panics in the CGO/spaCy code
	defer func() {
		if r := recover(); r != nil {
			// Return empty values on panic but don't crash
			words = nil
			offsets = nil
			posTags = nil
			tenses = nil
			depHashes = nil
		}
	}()

	if nb == nil || nb.nlp == nil {
		return nil, nil, nil, nil, nil
	}

	tokens := nb.nlp.Tokenize(text)

	words = make([]string, len(tokens))
	offsets = make([]int32, len(tokens))
	posTags = make([]int, len(tokens))
	tenses = make([]int, len(tokens))
	depHashes = make([]uint32, len(tokens))

	// Compute character offsets by scanning forward through the text.
	// go-spacy's Token does not expose a Start field via the Go binding,
	// so we track the search position manually.
	searchFrom := 0
	for i, t := range tokens {
		words[i] = t.Text
		idx := strings.Index(text[searchFrom:], t.Text)
		if idx >= 0 {
			offsets[i] = int32(searchFrom + idx)
			searchFrom += idx + len(t.Text)
		} else {
			offsets[i] = int32(searchFrom)
		}
		posTags[i] = int(MapPOSTag(t.POS))
		depHashes[i] = HashDependency(t.Dep)
		tenses[i] = int(DetectTense(t.Tag))
	}

	return
}

// MapPOSTag converts Spacy POS strings to uint8 IDs (from DATA-MAPPER.md spec)
func MapPOSTag(pos string) uint8 {
	switch strings.ToUpper(pos) {
	case "NOUN":
		return 0x01
	case "VERB":
		return 0x02
	case "ADJ":
		return 0x03
	case "ADV":
		return 0x04
	case "PRON":
		return 0x05
	case "PROPN":
		return 0x06
	case "DET":
		return 0x07
	case "ADP":
		return 0x08
	case "NUM":
		return 0x09
	case "CONJ", "CCONJ":
		return 0x0A
	case "SCONJ":
		return 0x0B
	case "AUX":
		return 0x0C
	case "PART":
		return 0x0D
	case "INTJ":
		return 0x0E
	case "PUNCT":
		return 0x0F
	case "SYM":
		return 0x10
	case "X":
		return 0x11
	case "SPACE":
		return 0x12
	default:
		return 0x00
	}
}

// DetectTense maps fine-grained SpaCy tags to tense IDs (from DATA-MAPPER.md spec)
func DetectTense(tag string) uint8 {
	switch strings.ToUpper(tag) {
	case "VBD", "VBN": // Past tense, Past participle
		return 0x01
	case "VBG", "VBP", "VBZ": // Present participle, Non-3rd person present, 3rd person present
		return 0x02
	case "MD": // Modal (often used for future/conditional)
		return 0x03
	default:
		return 0x00
	}
}

// HashDependency creates a 32-bit hash for the dependency link
func HashDependency(dep string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(dep))
	return h.Sum32()
}
