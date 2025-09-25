"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createFactualitySlice = createFactualitySlice;
// Simple factuality slice stub: produce the standardized schema with mock citations
function createFactualitySlice(questionOrText, context) {
    if (context === void 0) { context = {}; }
    var answerText = questionOrText.length > 0 ? "Stub answer for: ".concat(questionOrText) : 'No answer';
    var citations = [
        { id: 'source:user:1', source: 'user', snippet: questionOrText.slice(0, 140), score: 0.6 },
        { id: 'source:context:1', source: 'context', snippet: JSON.stringify(context).slice(0, 140), score: 0.4 }
    ];
    var response = {
        answer: answerText,
        citations: citations.map(function (c) { return c.id; }),
        confidence: 0.85,
        refused: false,
        hallucination_risk: 0.05,
        evidence_quality_score: 0.6
    };
    return {
        response: response,
        citations: citations,
        provenance: { generatedBy: 'factuality-slice-stub', model: 'stub-v0', timestamp: Date.now() }
    };
}
