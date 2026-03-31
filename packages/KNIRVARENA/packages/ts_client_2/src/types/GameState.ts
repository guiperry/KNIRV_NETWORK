import { ErrorNode, SkillNode, IdeaNode, PropertyNode, Agent } from '../components/game/stores/useKnirvana';

export interface GameState {
    // Game state
    gamePhase: "menu" | "playing" | "paused" | "training";
    gameTime: number;
    
    // Resources
    nrnBalance: number;
    skillsLearned: number;
    errorsResolved: number;
    ideasDeveloped: number;
    propertiesCreated: number;

    // Game objects
    errorNodes: ErrorNode[];
    skillNodes: SkillNode[];
    ideaNodes: IdeaNode[];
    propertyNodes: PropertyNode[];
    agents: Agent[];

    // Selection
    selectedErrorNode: string | null;
    selectedIdeaNode: string | null;
    selectedPropertyNode: string | null;
    selectedAgent: string | null;

    // Tournament specific
    skillSlotOwner: string | null;
    incumbentScore: number;
}
