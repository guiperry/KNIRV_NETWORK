export interface TrajectoryStep {
    step: number;
    thought: string;
    action: string; // The code snippet or token output
}

export interface WinnerPayload {
    agent_metadata: {
        agent_id: string;
        generation: number;
        victory_type: 'convergence' | 'hijack' | 'defensive';
    };
    prompt_context: string;
    trajectory: TrajectoryStep[];
    reward_signal: {
        score: number;       // 0.0 to 1.0
        latency_ms: number;
        verifier_feedback: string;
    };
}

// JSON Schema for validation
export const trajectorySchema = {
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "RFTWinnerTrajectory",
  "type": "object",
  "properties": {
    "agent_metadata": {
      "type": "object",
      "properties": {
        "agent_id": { "type": "string", "format": "uuid" },
        "generation": { "type": "integer" },
        "base_adapter": { "type": "string" }
      }
    },
    "prompt_context": {
      "type": "string",
      "description": "The state of the Error Node/Task given to the Agent."
    },
    "trajectory": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "step": { "type": "integer" },
          "thought": { "type": "string", "description": "Internal reasoning (CoT)" },
          "action": { "type": "string", "description": "The actual token output" }
        }
      }
    },
    "reward_signal": {
      "type": "object",
      "properties": {
        "score": { "type": "number", "minimum": 0, "maximum": 1 },
        "latency_ms": { "type": "integer" },
        "verifier_feedback": { "type": "string" }
      }
    }
  },
  "required": ["agent_metadata", "prompt_context", "trajectory", "reward_signal"]
};
