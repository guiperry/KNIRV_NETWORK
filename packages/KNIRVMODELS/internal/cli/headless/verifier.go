package headless

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"knirvmodels/pkg/factslice"
	"knirvmodels/pkg/heartqwen"
	"knirvmodels/pkg/models"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// enableVerifier makes POST /models/hasher/verify run the model-backed pipeline
// (pkg/factslice) on the local llama. The daemon does not need llama to be up: until
// it is, every verification answers degraded (gate uncommitted, not charged), never a
// verdict.
//
//	KNIRV_VERIFY=0                      keep the interim exact-containment verifier
//	KNIRV_FACTSLICE_CALIBRATION=<path>  a calibration fitted on owner-labelled gold data
//	                                    (factslice.Calibrator JSON); without it no claim
//	                                    is taken past "seen"
//	KNIRV_LLAMA_MODEL_SHA256, KNIRV_LLAMA_ADAPTERS_HASH
//	                                    identify the judge in responses and in the cache key
func (s *Server) enableVerifier(hasher *models.Hasher) {
	if os.Getenv("KNIRV_VERIFY") == "0" {
		log.Printf("HASHER verify: interim verifier (KNIRV_VERIFY=0)")
		return
	}
	llm := &heartqwen.HTTPLLM{Client: sdk.NewUnixHTTPClient(llamaSocketPath(), 10*time.Minute), Base: "http://llama", Model: liveFacts().Model}
	cfg := models.VerifierConfig{
		LLM:   llm,
		Facts: liveFacts,
		Live:  llm.Ready,
		Judge: func(context.Context) sdk.VerifyJudge {
			return sdk.VerifyJudge{
				Model:    liveFacts().Model,
				SHA256:   strings.TrimSpace(os.Getenv("KNIRV_LLAMA_MODEL_SHA256")),
				Adapters: strings.TrimSpace(os.Getenv("KNIRV_LLAMA_ADAPTERS_HASH")),
			}
		},
	}
	if path := strings.TrimSpace(os.Getenv("KNIRV_FACTSLICE_CALIBRATION")); path != "" {
		if data, err := os.ReadFile(path); err != nil {
			log.Printf("HASHER verify: calibration %s unreadable (%v); running uncalibrated", path, err)
		} else if cal, err := factslice.LoadCalibrator(data); err != nil || !cal.IsFitted() {
			log.Printf("HASHER verify: calibration %s is not a fitted calibrator (%v); running uncalibrated", path, err)
		} else {
			cfg.Cal = cal
		}
	}
	hasher.UseVerifier(cfg)
	log.Printf("HASHER verify: model-backed (calibrated=%t)", cfg.Cal.IsFitted())
}
