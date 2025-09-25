"use strict";
var __spreadArray = (this && this.__spreadArray) || function (to, from, pack) {
    if (pack || arguments.length === 2) for (var i = 0, l = from.length, ar; i < l; i++) {
        if (ar || !(i in from)) {
            if (!ar) ar = Array.prototype.slice.call(from, 0, i);
            ar[i] = from[i];
        }
    }
    return to.concat(ar || Array.prototype.slice.call(from));
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.createFeasibilitySlice = createFeasibilitySlice;
// Simple feasibility slice stub: compare to a small local token set
function createFeasibilitySlice(title, description, existingItems) {
    if (existingItems === void 0) { existingItems = []; }
    // naive similarity: common word overlap
    var norm = function (s) { return s.toLowerCase().replace(/[^a-z0-9\s]/g, '').split(/\s+/).filter(Boolean); };
    var descWords = norm("".concat(title, " ").concat(description));
    var similar = existingItems.map(function (item) {
        var itemWords = norm(item.text || '');
        var intersection = descWords.filter(function (w) { return itemWords.includes(w); }).length;
        var union = new Set(__spreadArray(__spreadArray([], descWords, true), itemWords, true)).size || 1;
        var score = intersection / union;
        return { id: item.id, score: score, summary: item.text.slice(0, 140) };
    }).filter(function (s) { return s.score > 0; }).sort(function (a, b) { return b.score - a.score; }).slice(0, 5);
    var exists = similar.length > 0 && similar[0].score > 0.6;
    var feasibilityScore = Math.round((1 - (similar.length > 0 ? similar[0].score : 0)) * 100);
    return {
        exists: exists,
        similar: similar,
        feasibilityScore: feasibilityScore,
        provenance: { generatedBy: 'feasibility-slice-stub', timestamp: Date.now() }
    };
}
