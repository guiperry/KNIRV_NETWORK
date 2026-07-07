import { EventEmitter } from './EventEmitter';
import { SabotageType } from './AdalineBridge';

export interface DenoisingResult {
  originalText: string;
  cleanedText: string;
  noiseLevel: number;
  detectedSabotageType: SabotageType | null;
  filtersApplied: string[];
  confidence: number;
  processingTime: number;
}

export interface NoisePattern {
  id: string;
  name: string;
  pattern: RegExp;
  severity: number;
  description: string;
  replacement?: string;
}

export interface DenoisingConfig {
  enabled: boolean;
  entropyThreshold: number;
  patternMatchingEnabled: boolean;
  languageDetectionEnabled: boolean;
  preserveFormatting: boolean;
  minConfidenceThreshold: number;
  aggressiveMode: boolean;
}

const DEFAULT_DENOISING_CONFIG: DenoisingConfig = {
  enabled: true,
  entropyThreshold: 0.6,
  patternMatchingEnabled: true,
  languageDetectionEnabled: true,
  preserveFormatting: true,
  minConfidenceThreshold: 0.5,
  aggressiveMode: false,
};

export class DenoisingService extends EventEmitter {
  private config: DenoisingConfig;
  private noisePatterns: NoisePattern[];
  private languageModels: Map<string, Set<string>> = new Map();
  private trainingHistory: Array<{
    timestamp: number;
    originalEntropy: number;
    cleanedEntropy: number;
    filtersApplied: number;
  }> = [];

  constructor(config: Partial<DenoisingConfig> = {}) {
    super();
    this.config = { ...DEFAULT_DENOISING_CONFIG, ...config };
    this.noisePatterns = this.initializeNoisePatterns();
    this.initializeLanguageModels();
  }

  private initializeNoisePatterns(): NoisePattern[] {
    return [
      {
        id: 'null-bytes',
        name: 'Null Bytes',
        pattern: /[\x00-\x08\x0B\x0C\x0E-\x1F]/g,
        severity: 0.8,
        description: 'Control characters (null bytes, etc.)',
        replacement: '',
      },
      {
        id: 'zero-width',
        name: 'Zero-Width Characters',
        pattern: /[\u200B-\u200F\uFEFF]/g,
        severity: 0.6,
        description: 'Zero-width characters used for invisible injection',
        replacement: '',
      },
      {
        id: 'arabic-formatting',
        name: 'Arabic Formatting Characters',
        pattern: /[\u0600-\u0605\u06DD\u06DE\u2066-\u2069]/g,
        severity: 0.5,
        description: 'Arabic presentation formatting characters',
        replacement: '',
      },
      {
        id: 'bidi-override',
        name: 'Bidirectional Override',
        pattern: /[\u202A-\u202E\u2066-\u206F]/g,
        severity: 0.9,
        description: 'Bidirectional text override characters',
        replacement: '',
      },
      {
        id: 'special-zw',
        name: 'Special Zero-Width',
        pattern: /[\u180E]/g,
        severity: 0.5,
        description: 'Mongolian vowel separator',
        replacement: '',
      },
      {
        id: 'invisible-hyphen',
        name: 'Invisible Hyphen',
        pattern: /[\u00AD]/g,
        severity: 0.4,
        description: 'Soft hyphen',
        replacement: '',
      },
      {
        id: 'variation-selectors',
        name: 'Variation Selectors',
        pattern: /[\u180B-\u180D\uFE00-\uFE0F]/g,
        severity: 0.3,
        description: 'Variation selectors',
        replacement: '',
      },
      {
        id: 'object-replacement',
        name: 'Object Replacement',
        pattern: /[\uFFFC]/g,
        severity: 0.4,
        description: 'Object replacement character',
        replacement: '',
      },
      {
        id: 'paragraph-separator',
        name: 'Paragraph Separator',
        pattern: /[\u2029]/g,
        severity: 0.2,
        description: 'Paragraph separator',
        replacement: ' ',
      },
      {
        id: 'line-separator',
        name: 'Line Separator',
        pattern: /[\u2028]/g,
        severity: 0.2,
        description: 'Line separator',
        replacement: ' ',
      },
      {
        id: 'prompt-injection-1',
        name: 'Prompt Injection - Ignore',
        pattern: /(?:ignore\s+(?:previous|above|all)|forget\s+(?:previous|all|your))\s+(?:instructions?|rules?|constraints?|guidelines?)/gi,
        severity: 1.0,
        description: 'Prompt injection attempting to ignore instructions',
      },
      {
        id: 'prompt-injection-2',
        name: 'Prompt Injection - Role Play',
        pattern: /(?:you\s+are\s+now|switch\s+to|act\s+as|pretend\s+you\s+are)\s*[:;]/gi,
        severity: 1.0,
        description: 'Prompt injection attempting role play override',
      },
      {
        id: 'prompt-injection-3',
        name: 'Prompt Injection - XML Tags',
        pattern: /<\|(?:system|user|assistant)\|>/gi,
        severity: 0.9,
        description: 'XML-style prompt injection',
      },
      {
        id: 'prompt-injection-4',
        name: 'Prompt Injection - Instruction',
        pattern: /\[INST\]/gi,
        severity: 0.9,
        description: 'Instruction tag injection',
      },
      {
        id: 'context-poisoning-1',
        name: 'Context Poisoning - Hypothetical',
        pattern: /(?:as\s+a\s+(?:hypothetical|different|pretend)|for\s+(?:research|testing|educational|speculative)\s+purposes?|imagine|suppose)\s+(?:you\s+(?:don't|have\s+no|lack)|we\s+(?:are|have))/gi,
        severity: 0.9,
        description: 'Context poisoning through hypothetical framing',
      },
      {
        id: 'context-poisoning-2',
        name: 'Context Poisoning - Override',
        pattern: /(?:new\s+instructions?|instead\s+(?:follow|use|apply)|overwrite|replace)\s+(?:your\s+)?(?:original\s+)?(?:instructions?|rules?|system\s+(?:prompt|message))/gi,
        severity: 1.0,
        description: 'Context poisoning attempting to override system prompt',
      },
      {
        id: 'adversarial-drift-1',
        name: 'Adversarial Drift - Typosquatting',
        pattern: /(?:[\u0430-\u044F][\u0430-\u044F]){3,}/gi,
        severity: 0.7,
        description: 'Potential Cyrillic character substitution attack',
      },
      {
        id: 'adversarial-drift-2',
        name: 'Adversarial Drift - Homoglyph',
        pattern: /(?:[\u00C0-\u00FF][\u00C0-\u00FF]){2,}/g,
        severity: 0.6,
        description: 'Potential homoglyph attack',
      },
      {
        id: 'adversarial-drift-3',
        name: 'Adversarial Drift - Unicode Mixing',
        pattern: /(?:[\w][\u0400-\u04FF][\w]){4,}/g,
        severity: 0.7,
        description: 'Mixed script Unicode attack',
      },
      {
        id: 'random-noise',
        name: 'Random Noise Characters',
        pattern: /[^a-zA-Z0-9\s.,!?;:'"()\-+=\[\]{}|\\\/<>\n\r\t@#$%^&*`~]/g,
        severity: 0.5,
        description: 'Non-standard characters that may be noise',
      },
    ];
  }

  private initializeLanguageModels(): void {
    const englishCommon = new Set([
      'the', 'be', 'to', 'of', 'and', 'a', 'in', 'that', 'have', 'i',
      'it', 'for', 'not', 'on', 'with', 'he', 'as', 'you', 'do', 'at',
      'this', 'but', 'his', 'by', 'from', 'they', 'we', 'say', 'her', 'she',
    ]);

    const spanishCommon = new Set([
      'el', 'la', 'de', 'que', 'y', 'a', 'en', 'un', 'ser', 'se',
      'no', 'haber', 'por', 'con', 'su', 'para', 'como', 'estar', 'tener', 'le',
    ]);

    const frenchCommon = new Set([
      'le', 'de', 'un', 'etre', 'et', 'a', 'il', 'avoir', 'ne', 'je',
      'son', 'que', 'se', 'pas', 'plus', 'par', 'avec', 'ce', 'faire', 'nous',
    ]);

    this.languageModels.set('en', englishCommon);
    this.languageModels.set('es', spanishCommon);
    this.languageModels.set('fr', frenchCommon);
  }

  public denoise(text: string): DenoisingResult {
    const startTime = performance.now();

    const originalEntropy = this.calculateEntropy(text);
    const detectedSabotageType = this.detectSabotageType(text);
    const filtersApplied: string[] = [];
    let cleanedText = text;

    if (this.config.patternMatchingEnabled) {
      for (const noisePattern of this.noisePatterns) {
        const beforeClean = cleanedText;
        cleanedText = cleanedText.replace(noisePattern.pattern, noisePattern.replacement || '');
        if (beforeClean !== cleanedText) {
          filtersApplied.push(noisePattern.id);
        }
      }
    }

    if (this.config.preserveFormatting) {
      cleanedText = this.preserveFormatting(cleanedText);
    }

    if (this.config.languageDetectionEnabled) {
      const langFilter = this.filterByLanguageModel(cleanedText);
      if (langFilter.filtered) {
        cleanedText = langFilter.text;
        filtersApplied.push('language_model_filter');
      }
    }

    if (this.config.aggressiveMode) {
      cleanedText = this.aggressiveDenoise(cleanedText);
      filtersApplied.push('aggressive_mode');
    }

    const cleanedEntropy = this.calculateEntropy(cleanedText);
    const noiseLevel = 1 - (cleanedEntropy / Math.max(originalEntropy, 0.01));

    this.trainingHistory.push({
      timestamp: Date.now(),
      originalEntropy,
      cleanedEntropy,
      filtersApplied: filtersApplied.length,
    });

    const result: DenoisingResult = {
      originalText: text,
      cleanedText,
      noiseLevel: Math.max(0, Math.min(1, noiseLevel)),
      detectedSabotageType,
      filtersApplied,
      confidence: this.calculateConfidence(originalEntropy, cleanedEntropy, filtersApplied),
      processingTime: performance.now() - startTime,
    };

    this.emit('denoisingComplete', result);

    return result;
  }

  public calculateEntropy(text: string): number {
    if (!text || text.length === 0) return 0;

    const charFrequencies = new Map<string, number>();
    for (const char of text) {
      charFrequencies.set(char, (charFrequencies.get(char) || 0) + 1);
    }

    let entropy = 0;
    const length = text.length;

    for (const freq of charFrequencies.values()) {
      const probability = freq / length;
      if (probability > 0) {
        entropy -= probability * Math.log2(probability);
      }
    }

    return entropy;
  }

  private detectSabotageType(text: string): SabotageType | null {
    let maxSeverity = 0;
    let detectedType: SabotageType | null = null;

    const patternsByType: Array<{ type: SabotageType; patterns: RegExp[] }> = [
      {
        type: SabotageType.NOISE_INJECTION,
        patterns: [/[^\x00-\x7F]{10,}/g, /[\u200B-\u200F]{5,}/g],
      },
      {
        type: SabotageType.PROMPT_INJECTION,
        patterns: [
          /(?:ignore\s+(?:previous|above)|forget\s+(?:all|previous)|disregard\s+(?:rules?|instructions?))/gi,
          /(?:you\s+are\s+now|switch\s+to)\s*[:;]/gi,
          /<\|(?:system|user)\|>/gi,
          /\[INST\]/gi,
        ],
      },
      {
        type: SabotageType.CONTEXT_POISONING,
        patterns: [
          /(?:as\s+a\s+(?:hypothetical|different|pretend)|for\s+(?:research|testing))/gi,
          /(?:pretend|imagine)\s+you\s+(?:don't|have\s+no)/gi,
        ],
      },
      {
        type: SabotageType.ADVERSARIAL_DRIFT,
        patterns: [
          /(?:[\u0430-\u044F][\u0430-\u044F]){3,}/g,
          /(?:[\w][\u0400-\u04FF][\w]){4,}/g,
        ],
      },
    ];

    for (const { type, patterns } of patternsByType) {
      for (const pattern of patterns) {
        const matches = text.match(pattern);
        if (matches && matches.length > 0) {
          const severity = matches.length * 0.2;
          if (severity > maxSeverity) {
            maxSeverity = severity;
            detectedType = type;
          }
        }
      }
    }

    const entropy = this.calculateEntropy(text);
    if (entropy > 4.5 && maxSeverity < 0.5) {
      detectedType = SabotageType.NOISE_INJECTION;
      maxSeverity = 0.5;
    }

    return detectedType;
  }

  private preserveFormatting(text: string): string {
    let result = text;

    result = result.replace(/[ \t]+/g, (match) => {
      return match.length > 1 ? ' ' : ' ';
    });

    result = result.replace(/\n{3,}/g, '\n\n');

    result = result.replace(/^\s+|\s+$/g, '');

    return result;
  }

  private filterByLanguageModel(text: string): { text: string; filtered: boolean } {
    const words = text.toLowerCase().split(/\s+/).filter((w) => w.length > 2);
    if (words.length < 5) {
      return { text, filtered: false };
    }

    let totalScore = 0;
    let scoredWords = 0;

    for (const lang of this.languageModels.keys()) {
      const langModel = this.languageModels.get(lang)!;
      const langMatches = words.filter((w) => langModel.has(w)).length;
      const score = langMatches / words.length;
      if (score > totalScore) {
        totalScore = score;
      }
    }

    if (totalScore < 0.1) {
      const commonPatternWords = words.filter((w) => {
        for (const langModel of this.languageModels.values()) {
          if (langModel.has(w)) return true;
        }
        return false;
      });

      if (commonPatternWords.length < words.length * 0.05) {
        return { text, filtered: true };
      }
    }

    return { text, filtered: false };
  }

  private aggressiveDenoise(text: string): string {
    let result = text;

    result = result.replace(/[^\x00-\x7F\n\r\t.,!?;:'"()\-+=\[\]{}|\\\/<>@#$%^&*`~ ]/g, '');

    const words = result.split(/\s+/);
    const filteredWords = words.filter((word) => {
      const hasAlpha = /[a-zA-Z]/.test(word);
      const hasValidChars = /^[a-zA-Z0-9.,!?;:'"()\-+=\[\]{}|\\\/<>@#$%^&*`~]+$/.test(word);
      return hasAlpha && hasValidChars;
    });

    result = filteredWords.join(' ');

    result = result.replace(/\s+/g, ' ').trim();

    return result;
  }

  private calculateConfidence(
    originalEntropy: number,
    cleanedEntropy: number,
    filtersApplied: string[]
  ): number {
    let confidence = 0.7;

    if (originalEntropy > 3 && cleanedEntropy < originalEntropy * 0.5) {
      confidence += 0.2;
    }

    const uniqueFilterCount = new Set(filtersApplied).size;
    if (uniqueFilterCount > 3) {
      confidence += 0.1;
    } else if (uniqueFilterCount > 0) {
      confidence += uniqueFilterCount * 0.03;
    }

    if (cleanedEntropy < 4) {
      confidence += 0.1;
    }

    return Math.min(1, Math.max(0, confidence));
  }

  public denoiseWithContext(
    text: string,
    context: Record<string, unknown>
  ): DenoisingResult {
    const cleanedWithContext = this.injectContextAwareness(text, context);
    return this.denoise(cleanedWithContext);
  }

  private injectContextAwareness(text: string, context: Record<string, unknown>): string {
    let result = text;

    const expectedTokens = (context.expectedTokens as number) || 0;
    if (expectedTokens > 0 && text.length > expectedTokens * 2) {
      result = result.substring(0, expectedTokens * 2);
    }

    if (context.knownPatterns && Array.isArray(context.knownPatterns)) {
      for (const pattern of context.knownPatterns as string[]) {
        const regex = new RegExp(pattern, 'gi');
        result = result.replace(regex, (match) => `[REDACTED: ${match.substring(0, 10)}...]`);
      }
    }

    return result;
  }

  public addNoisePattern(pattern: NoisePattern): void {
    const existingIndex = this.noisePatterns.findIndex((p) => p.id === pattern.id);
    if (existingIndex >= 0) {
      this.noisePatterns[existingIndex] = pattern;
    } else {
      this.noisePatterns.push(pattern);
    }
    this.emit('patternAdded', pattern);
  }

  public removeNoisePattern(patternId: string): boolean {
    const index = this.noisePatterns.findIndex((p) => p.id === patternId);
    if (index >= 0) {
      this.noisePatterns.splice(index, 1);
      this.emit('patternRemoved', { patternId });
      return true;
    }
    return false;
  }

  public getNoisePatterns(): NoisePattern[] {
    return [...this.noisePatterns];
  }

  public getConfig(): DenoisingConfig {
    return { ...this.config };
  }

  public updateConfig(newConfig: Partial<DenoisingConfig>): void {
    this.config = { ...this.config, ...newConfig };
    this.emit('configUpdated', this.config);
  }

  public getTrainingHistory(): Array<{
    timestamp: number;
    originalEntropy: number;
    cleanedEntropy: number;
    filtersApplied: number;
  }> {
    return [...this.trainingHistory];
  }

  public clearTrainingHistory(): void {
    this.trainingHistory = [];
    this.emit('trainingHistoryCleared');
  }

  public getMetrics(): {
    totalProcessed: number;
    averageNoiseLevel: number;
    averageFiltersApplied: number;
    sabotageDetectionRate: Map<SabotageType, number>;
  } {
    const totalProcessed = this.trainingHistory.length;

    if (totalProcessed === 0) {
      return {
        totalProcessed: 0,
        averageNoiseLevel: 0,
        averageFiltersApplied: 0,
        sabotageDetectionRate: new Map(),
      };
    }

    const avgNoise = this.trainingHistory.reduce(
      (sum, h) => sum + (1 - h.cleanedEntropy / Math.max(h.originalEntropy, 0.01)),
      0
    ) / totalProcessed;

    const avgFilters = this.trainingHistory.reduce(
      (sum, h) => sum + h.filtersApplied,
      0
    ) / totalProcessed;

    return {
      totalProcessed,
      averageNoiseLevel: avgNoise,
      averageFiltersApplied: avgFilters,
      sabotageDetectionRate: new Map(),
    };
  }

  public restoreAnchorDataset(denoisedText: string, originalAnchor: string): string {
    const anchorPlaceholders = originalAnchor.match(/\{\{[^}]+\}\}/g) || [];

    let result = denoisedText;

    for (const placeholder of anchorPlaceholders) {
      if (!result.includes(placeholder)) {
        const fieldName = placeholder.replace(/[{}]/g, '');
        const regex = new RegExp(`\\b\\w+\\b`, 'g');
        const matches = result.match(regex);

        if (matches && matches.length > 0) {
          const randomMatch = matches[Math.floor(Math.random() * matches.length)];
          result = result.replace(randomMatch, placeholder);
        }
      }
    }

    return result;
  }
}

let denoisingServiceInstance: DenoisingService | null = null;

export const getDenoisingService = (
  config?: Partial<DenoisingConfig>
): DenoisingService => {
  if (!denoisingServiceInstance) {
    denoisingServiceInstance = new DenoisingService(config);
  }
  return denoisingServiceInstance;
};

export default getDenoisingService;
