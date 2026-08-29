export type ErrorNodeType = 'Memory Leak' | 'Logic Error' | 'Race Condition' | 'Buffer Overflow' | 'API Timeout';

/** Runtime challenge shape populated from backend-owned code_error postings. */
export interface Challenge {
  id: string;
  title: string;
  type: ErrorNodeType;
  difficulty: number;
  bounty: number;
  description: string;
  buggyCode: string;
  context: string;
  hints: string[];
}
