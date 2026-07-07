package did

import "time"

type DIDDocument struct {
	Context            []string             `json:"@context"`
	ID                 string               `json:"id"`
	VerificationMethod []VerificationMethod `json:"verificationMethod"`
	Authentication     []string             `json:"authentication"`
	AssertionMethod    []string             `json:"assertionMethod"`
	Created            time.Time            `json:"created"`
	Updated            time.Time            `json:"updated"`
}

type VerificationMethod struct {
	ID                 string `json:"id"`
	Type               string `json:"type"`
	Controller         string `json:"controller"`
	PublicKeyMultibase string `json:"publicKeyMultibase"`
}
