package api

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
)

// SAMLMessage represents a parsed and potentially mutated SAML assertion/request.
type SAMLMessage struct {
	XMLName      xml.Name      `json:"-"`
	Type         string        `json:"type"` // "AuthnRequest", "Response", "Assertion"
	ID           string        `json:"id"`
	Issuer       string        `json:"issuer,omitempty"`
	Subject      string        `json:"subject,omitempty"`
	NotOnOrAfter string        `json:"notOnOrAfter,omitempty"`
	Assertions   []SAMLMessage `json:"assertions,omitempty"`
	RawXML       string        `json:"rawXML"`
	Mutations    []string      `json:"mutations,omitempty"`
}

// samlRaiderArgs carries the input for the native SAML Raider tool.
type samlRaiderArgs struct {
	FlowID      int    `json:"flowId,omitempty"`      // ID of a Proxy flow containing SAML
	SAMLContent string `json:"samlContent,omitempty"` // raw SAML XML string
	Action      string `json:"action,omitempty"`      // "parse", "strip-signatures", "forge-assertion"
}

// samlMutationResult is the output of a SAML mutation operation.
type samlMutationResult struct {
	Original SAMLMessage `json:"original"`
	Mutated  SAMLMessage `json:"mutated"`
	Action   string      `json:"action"`
	Success  bool        `json:"success"`
	Error    string      `json:"error,omitempty"`
}

func init() {
	registerLane6Tool("saml-raider", nativeToolFunc(func(session *SandboxSession, args json.RawMessage) (json.RawMessage, error) {
		var req samlRaiderArgs
		if err := json.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("invalid SAML Raider args: %v", err)
		}

		samlContent := req.SAMLContent
		if samlContent == "" {
			// Try to find SAML in captured proxy flows.
			samlContent = findSAMLInFlows(session)
		}
		if samlContent == "" {
			return nil, fmt.Errorf("no SAML content provided and no SAML traffic detected in proxy flows")
		}

		parsed, err := parseSAML(samlContent)
		if err != nil {
			return nil, fmt.Errorf("failed to parse SAML: %v", err)
		}

		result := samlMutationResult{
			Original: parsed,
			Action:   req.Action,
		}

		switch req.Action {
		case "strip-signatures":
			mutated := stripSAMLSignatures(parsed)
			result.Mutated = mutated
			result.Success = true
		case "forge-assertion":
			mutated := forgeSAMLAssertion(parsed)
			result.Mutated = mutated
			result.Success = true
		case "parse", "":
			// Just return the parsed result.
			result.Mutated = parsed
			result.Success = true
		default:
			result.Error = fmt.Sprintf("unknown action: %s", req.Action)
		}

		return json.Marshal(result)
	}))
}

// findSAMLInFlows searches captured proxy flows for SAML content.
func findSAMLInFlows(session *SandboxSession) string {
	session.mutex.RLock()
	defer session.mutex.RUnlock()
	for _, flow := range session.proxyFlows {
		// SAML messages are typically in POST bodies; we can't access bodies
		// directly from the flow struct, but we can check the path/host.
		if strings.Contains(strings.ToLower(flow.Path), "saml") ||
			strings.Contains(strings.ToLower(flow.Host), "saml") {
			// In a real implementation, we'd retrieve the body from a buffer.
			return ""
		}
	}
	return ""
}

// parseSAML parses a SAML XML message into a SAMLMessage struct.
func parseSAML(content string) (SAMLMessage, error) {
	var msg SAMLMessage
	msg.RawXML = content

	// Try parsing as AuthnResponse/Assertion.
	var response struct {
		XMLName   xml.Name `xml:"Response"`
		ID        string   `xml:"ID,attr"`
		Issuer    string   `xml:"Issuer"`
		Assertion struct {
			ID         string `xml:"ID,attr"`
			Issuer     string `xml:"Issuer"`
			Subject    string `xml:"Subject>NameID"`
			Conditions struct {
				NotOnOrAfter string `xml:"NotOnOrAfter,attr"`
			} `xml:"Conditions"`
		} `xml:"Assertion"`
	}
	if err := xml.Unmarshal([]byte(content), &response); err == nil {
		msg.Type = "Response"
		msg.ID = response.ID
		msg.Issuer = response.Issuer
		if response.Assertion.ID != "" {
			msg.Subject = response.Assertion.Subject
			msg.NotOnOrAfter = response.Assertion.Conditions.NotOnOrAfter
		}
		return msg, nil
	}

	// Try parsing as AuthnRequest.
	var request struct {
		XMLName xml.Name `xml:"AuthnRequest"`
		ID      string   `xml:"ID,attr"`
		Issuer  string   `xml:"Issuer"`
	}
	if err := xml.Unmarshal([]byte(content), &request); err == nil {
		msg.Type = "AuthnRequest"
		msg.ID = request.ID
		msg.Issuer = request.Issuer
		return msg, nil
	}

	return msg, fmt.Errorf("unrecognized SAML message format")
}

// stripSAMLSignatures removes XML digital signatures from a SAML message.
func stripSAMLSignatures(msg SAMLMessage) SAMLMessage {
	mutated := msg
	mutated.Mutations = append(mutated.Mutations, "stripsig")
	// Remove <ds:Signature> and <Signature> elements from the XML.
	raw := msg.RawXML
	raw = removeXMLTag(raw, "ds:Signature")
	raw = removeXMLTag(raw, "Signature")
	mutated.RawXML = raw
	return mutated
}

// forgeSAMLAssertion creates a modified assertion with the same issuer.
func forgeSAMLAssertion(msg SAMLMessage) SAMLMessage {
	mutated := msg
	mutated.Mutations = append(mutated.Mutations, "forge")
	mutated.Subject = "forgeed@knirv.local"
	return mutated
}

// removeXMLTag removes all occurrences of <tag>...</tag> from XML content.
func removeXMLTag(content, tag string) string {
	openTag := "<" + tag
	closeTag := "</" + tag + ">"
	for {
		start := strings.Index(content, openTag)
		if start == -1 {
			break
		}
		end := strings.Index(content[start:], closeTag)
		if end == -1 {
			break
		}
		end += start + len(closeTag)
		content = content[:start] + content[end:]
	}
	return content
}
