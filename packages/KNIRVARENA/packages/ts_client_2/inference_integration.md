# Cognitive Engine Inference Integration Plan: Adaline Routing & In-Game Behaviors

## 1. Introduction
This document outlines the architectural plan to route the **Cognitive Engine** (`src/sensory-shell/CognitiveEngine.ts`) through the **Adaline** implementation (`src/services/llmProviderService.ts`) for all in-game inference. This transition enables a multi-model LLM gateway and incorporates advanced agent behaviors for gameplay.

## 2. Architectural Overview
The `CognitiveEngine` will act as the central nervous system for in-game agents, utilizing an **AdalineBridge** to delegate high-level reasoning while maintaining local WASM-based validation and noise management.

### Data Flow
`Game Logic` → `CognitiveEngine` → **`AdalineBridge`** → `LLMProviderService` → `Adaline Gateway` → `Target LLM`

---

## 3. Advanced Agent Behaviors

### 3.1 Anchor Dataset Orchestration
Agents will utilize "Anchor Datasets" to maintain behavioral consistency and provide few-shot context for LLM inference.
*   **Template Filling**: The engine will dynamically populate dataset templates using **Error Node context** (e.g., historical failure patterns, adversarial drift data).
*   **Contextual Injection**: These populated templates are injected into the Adaline prompt to guide the agent's decision-making process in complex world states.

### 3.2 Solution Validation (DVE/CDE)
Before an agent commits to a high-impact action, the solution must be validated in a controlled environment.
*   **DVE (Distributed Virtual Environment)**: The agent runs a simulation of the proposed solution across the peer network to score its effectiveness.
*   **CDE (Cognitive Development Environment)**: A local WASM-isolated sandbox where solutions are tested against known game constraints.
*   **Validation Threshold**: Only solutions with a `dveValidationScore > 0.7` are cleared for execution in the live arena.

### 3.3 Field Noise Remediation
Opponents may introduce "Field Noise" (adversarial context injection) to sabotage an agent's reasoning.
*   **Noise Detection**: The engine monitors the entropy of the input context to identify `SabotageType.NOISE_INJECTION`.
*   **Denoising Mechanic**: Utilizing `TrainingManager.denoise()` logic, the agent will filter out random/adversarial characters and restore the integrity of the "Anchor Dataset" before inference.
*   **Robustness Scoring**: Adaline results are cross-referenced with local `adversarialRobustness` metrics to ensure the agent hasn't been "hijacked."

---

## 4. Implementation Phases

### Phase 1: Adaline Gateway & Provider Alignment
*   **LLMProviderService**: Replace placeholders in `adalineChat` with a full `@adaline/gateway` implementation.
*   **Provider Logic**: Ensure the `Gateway` constructor handles Gemini, OpenAI, and Anthropic with robust error handling and fallbacks.
*   **Service Routing**: Update `ChatBrainService` to ensure the `CognitiveEngine` is the primary gatekeeper for all outgoing inference.

### Phase 2: AdalineBridge & Context Mapping
*   **Bridge Development**: Create `src/sensory-shell/AdalineBridge.ts` mirroring the `HRMBridge` interface.
*   **Context Translation**: Develop logic to transform `HRMCognitiveInput` (sensory data + context) into high-fidelity prompts.
*   **Output Parsing**: Translate LLM responses into structured `HRMCognitiveOutput` (confidence, reasoning, etc.).

### Phase 3: CognitiveEngine Core Integration
*   **Configuration**: Extend `CognitiveConfig` with `adalineEnabled` and `adalineConfig`.
*   **Bridge Initialization**: Setup `AdalineBridge` in `initializeComponents`.
*   **Priority Routing**: Update `processTextInput/Voice/Visual` to prioritize Adaline for complex reasoning.
*   **Skill Invocation**: Modify `invokeSkill` to allow Adaline to dynamically resolve and execute game-state skills.

### Phase 4: Anchor, Noise, & DVE Systems
*   **Anchor Orchestration**: Implement `AnchorDatasetManager` to populate templates with Error Node context.
*   **Noise Management**: Integrate `Denoising` filters into the input pipeline.
*   **Validation Loop**: Connect the engine to the DVE scoring system and implement "Pre-flight Validation" checks for high-severity tasks.

### Phase 5: UI & Feedback Systems
*   **Real-time Status**: Update `VerifierOverlay` and `AnalyticsDashboard` to display DVE scores, noise levels, and anchor status.
*   **Visual Feedback**: Reflect "Adaline-Powered" reasoning in the UI during agent thinking cycles.

## 5. Configuration Requirements
*   `VITE_ADALINE_API_KEY`: Required for gateway authentication.
*   `VITE_DVE_VALIDATION_THRESHOLD`: Default `0.7`.
