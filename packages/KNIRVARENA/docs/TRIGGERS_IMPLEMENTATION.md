# TRIGGERS IMPLEMENTATION GUIDE

## Overview

This document provides the technical architecture and implementation strategy for building the trigger system described in USER_TRIGGERS.md. The goal is to transform the KNIRVARENA from a destination chatbot into a proactive "shadow" system that follows users across their digital workspace.

## Architecture Principles

### Core Philosophy
- **Event-Driven Architecture**: All triggers are event-based, not polling-based
- **Context-First Design**: Every interaction enriches the user's context graph
- **Cross-Platform Synthesis**: Data flows from multiple sources into a unified memory system
- **Proactive Intelligence**: The system initiates interactions based on detected patterns
- **Privacy by Design**: User controls what data is indexed and surfaced

---

## System Architecture

### High-Level Components

```
┌─────────────────────────────────────────────────────────────────┐
│                      TRIGGER ORCHESTRATOR                       │
│  (Centralized event processing and trigger decision engine)     │
└────────────┬────────────────────────────────────┬───────────────┘
             │                                    │
     ┌───────▼────────┐                  ┌────────▼──────────┐
     │  EVENT STREAMS │                  │  TRIGGER ENGINES  │
     │                │                  │                   │
     │ • User Actions │                  │ • Habit Engine    │
     │ • Context      │                  │ • Resurrection    │
     │   Changes      │                  │ • Synthesis       │
     │ • External     │                  │ • Proactive       │
     │   Integrations │                  │   Surface         │
     └───────┬────────┘                  └────────┬──────────┘
             │                                    │
     ┌───────▼────────────────────────────────────▼──────────┐
     │              CONTEXT GRAPH STORE                      │
     │  (NebulaDB + Vector Store for semantic memory)        │
     └───────────────────────────────────────────────────────┘
             │
     ┌───────▼───────────────────────────────────────────────┐
     │              DELIVERY MECHANISMS                      │
     │  • Push Notifications • Email • In-App • Extensions   │
     └───────────────────────────────────────────────────────┘
```

---

## Phase 1: Foundation - Context Ingestion & Graph Building

### 1.1 Data Ingestion Pipeline

**Purpose**: Implement the "Memory Ingest" system that captures user context from multiple sources.

#### Data Sources to Integrate

| Source | Priority | Implementation | Privacy Level |
|--------|----------|----------------|---------------|
| **Chat History Import** | P0 | Direct upload of ChatGPT/Claude/Gemini exports | User-controlled |
| **Browser Extension** | P0 | Chrome/Firefox extension capturing tab context | Opt-in per domain |
| **Slack Integration** | P1 | OAuth + webhook for DMs and @mentions | Workspace-level consent |
| **Email (Gmail/Outlook)** | P1 | OAuth + selective folder access | User-controlled labels |
| **Notion/Obsidian** | P1 | API integration for personal workspaces | Page-level permissions |
| **Calendar Events** | P2 | Google/Outlook Calendar API | Event metadata only |
| **Code Repositories** | P2 | GitHub/GitLab webhooks for commits/PRs | Repo-level opt-in |

#### Implementation: Browser Extension

**File Structure**:
```
src/extensions/
├── browser-extension/
│   ├── manifest.json
│   ├── background/
│   │   ├── event-listener.ts      # Captures tab changes, copy events
│   │   ├── context-extractor.ts   # Extracts semantic entities from pages
│   │   └── sync-manager.ts        # Batches and syncs to backend
│   ├── content-scripts/
│   │   ├── page-monitor.ts        # Watches for specific triggers (Google Docs, etc.)
│   │   └── whisper-injector.ts    # Injects the "Option+Space" hotkey UI
│   └── popup/
│       └── quick-capture.tsx      # Mini-UI for manual capture
```

**Key Features**:
- **Passive Monitoring**: Track active tab URL, title, and domain (with user whitelist)
- **Smart Extraction**: Use regex + NLP to extract entities (project names, people, technical terms)
- **Whisper Mode**: Global hotkey (`Option+Space`) to open quick-capture modal
- **Privacy Controls**: Domain-level opt-in/opt-out with visual indicator

**API Contract**:
```typescript
// Event sent from extension to backend
interface ContextEvent {
  userId: string;
  timestamp: number;
  eventType: 'page_visit' | 'copy_text' | 'whisper_capture' | 'tab_switch';
  context: {
    url?: string;
    domain?: string;
    pageTitle?: string;
    selectedText?: string;
    voiceTranscript?: string;
    extractedEntities: Entity[];
  };
  metadata: {
    browserType: string;
    sessionId: string;
  };
}

interface Entity {
  type: 'person' | 'project' | 'concept' | 'task' | 'date';
  value: string;
  confidence: number;
  source: 'url' | 'title' | 'selection' | 'whisper';
}
```

### 1.2 Context Graph Store

**Database Schema** (NebulaDB + Vector Store hybrid):

```typescript
// Primary graph structure
interface MemoryNode {
  id: string;                    // UUID
  userId: string;
  nodeType: 'conversation' | 'entity' | 'project' | 'insight' | 'task';
  content: string;               // Raw text
  embedding: number[];           // Vector embedding for semantic search
  metadata: {
    source: string;              // 'slack', 'notion', 'browser', 'chat'
    platform: string;            // 'chatgpt', 'claude', 'gemini', 'web'
    timestamp: number;
    entities: Entity[];
    sentiment?: number;
    importance?: number;         // 0-1 score
  };
  edges: Edge[];
}

interface Edge {
  targetNodeId: string;
  edgeType: 'references' | 'relates_to' | 'continues' | 'solves' | 'contradicts';
  weight: number;                // Strength of connection
  createdAt: number;
}

// For tracking unresolved loops (Zeigarnik Effect)
interface UnresolvedLoop {
  id: string;
  userId: string;
  projectName: string;
  firstMentionedAt: number;
  lastMentionedAt: number;
  sourceNodes: string[];         // IDs of nodes that mention this loop
  status: 'open' | 'stale' | 'resolved';
  estimatedImportance: number;   // ML-based score
}

// User personality profile (for tone mirroring)
interface UserProfile {
  userId: string;
  communicationStyle: {
    formality: number;           // 0 = casual, 1 = formal
    verbosity: number;           // 0 = brief, 1 = verbose
    preferredFormat: 'bullets' | 'paragraphs' | 'mixed';
    commonPhrases: string[];
    jargonTerms: string[];
  };
  workPatterns: {
    peakHours: number[];         // Hours of day (0-23)
    projectFocus: Map<string, number>; // Project name → % attention
    topEntities: Entity[];
  };
  correctionHistory: {
    timestamp: number;
    field: string;
    oldValue: string;
    newValue: string;
  }[];
}
```

**Vector Store Integration**:
- Use **Qdrant** or **Pinecone** for semantic search
- Embed all memory nodes using a lightweight model (e.g., `all-MiniLM-L6-v2`)
- Enable hybrid search: vector similarity + graph traversal

### 1.3 Entity Extraction & Linking

**Implementation**: `src/core/entity-extraction/EntityLinker.ts`

```typescript
class EntityLinker {
  // Extract high-signal entities from raw text
  async extractEntities(text: string, context: ContextEvent): Promise<Entity[]> {
    const entities: Entity[] = [];

    // 1. Named Entity Recognition (use spaCy via Python bridge or compromise.js)
    const namedEntities = await this.ner.extract(text);

    // 2. Pattern matching for "high-signal" phrases
    const intensityPatterns = [
      /I need to (.+)/gi,
      /I'm worried about (.+)/gi,
      /What if we (.+)/gi,
      /TODO: (.+)/gi,
      /FIXME: (.+)/gi,
    ];

    for (const pattern of intensityPatterns) {
      const matches = text.matchAll(pattern);
      for (const match of matches) {
        entities.push({
          type: 'task',
          value: match[1],
          confidence: 0.9,
          source: context.eventType,
        });
      }
    }

    // 3. Cross-reference with existing graph to link to projects
    const linkedEntities = await this.linkToGraph(entities);

    return linkedEntities;
  }

  // Link extracted entities to existing nodes in the graph
  async linkToGraph(entities: Entity[]): Promise<Entity[]> {
    // Use vector similarity to find related existing nodes
    for (const entity of entities) {
      const embedding = await this.embed(entity.value);
      const similar = await this.vectorStore.search(embedding, { limit: 3, threshold: 0.7 });

      if (similar.length > 0) {
        // Create edge in graph
        await this.createEdge({
          sourceNodeId: entity.id,
          targetNodeId: similar[0].id,
          edgeType: 'relates_to',
          weight: similar[0].score,
        });
      }
    }

    return entities;
  }
}
```

---

## Phase 2: Trigger Engines

### 2.1 The Habit Formation Engine

**Purpose**: Implement the 7-day habit formation map with automated triggers.

**File**: `src/triggers/HabitEngine.ts`

```typescript
class HabitEngine {
  // Day 1: The Ingest
  async handleDay1Onboarding(userId: string): Promise<void> {
    // Trigger: User has connected first data source
    const dataSources = await this.getConnectedSources(userId);

    if (dataSources.length >= 1) {
      // Process the data and generate first "Aha!" moment
      const summary = await this.generateFirstSummary(userId);

      await this.sendNotification({
        userId,
        type: 'onboarding_insight',
        title: 'Your Digital Brain is Ready',
        message: summary.preview,
        action: {
          type: 'view_identity_brief',
          data: summary,
        },
      });

      // Mark milestone
      await this.updateUserJourney(userId, 'day1_complete');
    }
  }

  // Day 2: First Proactive Surface
  async scheduleProactiveSurface(userId: string): Promise<void> {
    // Background job: Watch for calendar events
    const upcomingMeetings = await this.getCalendarEvents(userId, {
      startTime: Date.now(),
      endTime: Date.now() + 24 * 60 * 60 * 1000, // Next 24 hours
    });

    for (const meeting of upcomingMeetings) {
      // Search memory graph for context about meeting attendees or topics
      const context = await this.findMeetingContext(userId, meeting);

      if (context.nodes.length > 0) {
        // Schedule notification 10 minutes before meeting
        await this.scheduleNotification({
          userId,
          sendAt: meeting.startTime - 10 * 60 * 1000,
          type: 'proactive_context',
          title: `Context for "${meeting.title}"`,
          message: `I found ${context.nodes.length} relevant notes from your past conversations.`,
          action: {
            type: 'show_context',
            data: context,
          },
        });
      }
    }
  }

  // Day 3: Cross-Model Bridge
  async detectCrossModelOpportunity(userId: string): Promise<void> {
    // Watch for new chat messages
    const recentChats = await this.getRecentChats(userId, { hours: 24 });

    for (const chat of recentChats) {
      // Check if current platform (e.g., Claude) can answer using data from different platform (e.g., Gemini)
      const crossModelMatch = await this.searchCrossModel(userId, chat.lastMessage);

      if (crossModelMatch.found && crossModelMatch.confidence > 0.8) {
        await this.injectCrossModelContext({
          userId,
          currentChat: chat,
          sourceData: crossModelMatch.nodes,
          message: `I remember you discussed this with ${crossModelMatch.originalPlatform} on ${crossModelMatch.date}.`,
        });

        // CRITICAL: This is the "Breaking the Siloed AI" moment
        await this.updateUserJourney(userId, 'day3_cross_model_bridge');
      }
    }
  }

  // Day 5: Friction Test (passive detection)
  async detectFrictionMoment(userId: string): Promise<void> {
    // Monitor browser extension events
    const recentActivity = await this.getBrowserActivity(userId, { hours: 1 });

    // Detect patterns: user searching for something they already discussed
    const searchQueries = recentActivity.filter(e => e.eventType === 'search');

    for (const query of searchQueries) {
      const existingMemory = await this.searchMemoryGraph(userId, query.context.searchTerm);

      if (existingMemory.nodes.length > 0) {
        // User is searching for something we already have
        await this.sendNotification({
          userId,
          type: 'friction_prevention',
          title: 'I have that answer',
          message: `You searched for "${query.context.searchTerm}" - I found ${existingMemory.nodes.length} notes you already created about this.`,
          action: {
            type: 'show_memory',
            data: existingMemory,
          },
        });

        // Track as a "save" event (quantify value)
        await this.trackFrictionSave(userId, query, existingMemory);
      }
    }
  }
}
```

### 2.2 The Proactive Surface Engine

**Purpose**: Surface old memories at contextually relevant moments.

**File**: `src/triggers/ProactiveSurfaceEngine.ts`

```typescript
class ProactiveSurfaceEngine {
  // Background worker: continuously monitors for surface opportunities
  async run(): Promise<void> {
    const activeUsers = await this.getActiveUsers();

    for (const user of activeUsers) {
      await this.checkSurfaceOpportunities(user.id);
    }
  }

  async checkSurfaceOpportunities(userId: string): Promise<void> {
    // Get current user context
    const currentContext = await this.getCurrentContext(userId);

    if (!currentContext) return;

    // Find related memories
    const relevantMemories = await this.findRelevantMemories(userId, currentContext);

    for (const memory of relevantMemories) {
      // Check if we should surface this memory
      const shouldSurface = await this.evaluateSurfaceDecision(memory, currentContext);

      if (shouldSurface.decision) {
        await this.surfaceMemory(userId, memory, shouldSurface.reason);
      }
    }
  }

  async findRelevantMemories(userId: string, context: UserContext): Promise<MemoryNode[]> {
    // Multi-strategy search:
    // 1. Vector similarity
    const embedding = await this.embed(context.summary);
    const vectorMatches = await this.vectorStore.search(embedding, {
      userId,
      limit: 10,
      threshold: 0.7,
    });

    // 2. Graph traversal (find connected nodes)
    const graphMatches = await this.traverseGraph(userId, context.entities);

    // 3. Temporal patterns (similar time of day, day of week)
    const temporalMatches = await this.findTemporalPatterns(userId, context.timestamp);

    // Merge and rank
    return this.rankMemories([...vectorMatches, ...graphMatches, ...temporalMatches]);
  }

  async evaluateSurfaceDecision(
    memory: MemoryNode,
    context: UserContext
  ): Promise<{ decision: boolean; reason: string; score: number }> {
    // Decision factors:
    const factors = {
      relevance: await this.computeRelevance(memory, context),      // 0-1
      recency: this.computeRecency(memory.metadata.timestamp),      // 0-1 (older = lower)
      novelty: await this.computeNovelty(memory, context),          // 0-1 (already shown = 0)
      importance: memory.metadata.importance || 0.5,                // 0-1
      userEngagement: await this.predictEngagement(memory, context), // 0-1
    };

    // Weighted scoring
    const score =
      factors.relevance * 0.4 +
      factors.novelty * 0.25 +
      factors.importance * 0.2 +
      factors.userEngagement * 0.15;

    // Threshold: only surface if score > 0.7
    if (score > 0.7) {
      return {
        decision: true,
        reason: this.explainDecision(factors),
        score,
      };
    }

    return { decision: false, reason: 'Score too low', score };
  }

  async surfaceMemory(userId: string, memory: MemoryNode, reason: string): Promise<void> {
    // Choose delivery mechanism based on context
    const deliveryMethod = await this.chooseDeliveryMethod(userId);

    const notification = {
      userId,
      type: 'contextual_echo',
      title: 'Relevant Memory',
      message: this.formatMemoryPreview(memory),
      metadata: {
        memoryNodeId: memory.id,
        surfaceReason: reason,
        timestamp: Date.now(),
      },
    };

    switch (deliveryMethod) {
      case 'push':
        await this.sendPushNotification(notification);
        break;
      case 'in_app':
        await this.queueInAppNotification(notification);
        break;
      case 'extension':
        await this.sendToExtension(notification);
        break;
    }

    // Track surfacing event
    await this.trackSurfaceEvent(userId, memory.id, deliveryMethod);
  }
}
```

### 2.3 The Synthesis Engine

**Purpose**: Generate "Connection of the Day" and weekly insights.

**File**: `src/triggers/SynthesisEngine.ts`

```typescript
class SynthesisEngine {
  // Runs daily: generates cross-platform, cross-time insights
  async generateDailySynthesis(userId: string): Promise<void> {
    // 1. Identify high-value node clusters from the past 7 days
    const recentNodes = await this.getRecentNodes(userId, { days: 7 });

    // 2. Find unexpected connections
    const connections = await this.findUnexpectedConnections(recentNodes);

    if (connections.length === 0) return;

    // 3. Rank by "Aha! potential"
    const topConnection = connections
      .sort((a, b) => b.ahaScore - a.ahaScore)[0];

    // 4. Generate human-readable synthesis
    const synthesis = await this.generateSynthesisText(topConnection);

    // 5. Deliver as "Connection of the Day"
    await this.sendNotification({
      userId,
      type: 'daily_synthesis',
      title: 'Connection of the Day',
      message: synthesis.preview,
      action: {
        type: 'view_synthesis',
        data: synthesis,
      },
    });
  }

  async findUnexpectedConnections(nodes: MemoryNode[]): Promise<Connection[]> {
    const connections: Connection[] = [];

    // Use embedding similarity to find cross-platform connections
    for (let i = 0; i < nodes.length; i++) {
      for (let j = i + 1; j < nodes.length; j++) {
        const nodeA = nodes[i];
        const nodeB = nodes[j];

        // Skip if from same conversation
        if (nodeA.metadata.source === nodeB.metadata.source &&
            nodeA.metadata.timestamp - nodeB.metadata.timestamp < 3600000) {
          continue;
        }

        // Compute semantic similarity
        const similarity = await this.computeSimilarity(nodeA.embedding, nodeB.embedding);

        if (similarity > 0.75) {
          // This is an unexpected connection
          connections.push({
            nodeA,
            nodeB,
            similarity,
            ahaScore: this.computeAhaScore(nodeA, nodeB, similarity),
            reasoning: await this.explainConnection(nodeA, nodeB),
          });
        }
      }
    }

    return connections;
  }

  async generateSynthesisText(connection: Connection): Promise<Synthesis> {
    // Use lightweight LLM to generate natural language
    const prompt = `
You are synthesizing insights from a user's memory graph.

Node A (from ${connection.nodeA.metadata.platform} on ${new Date(connection.nodeA.metadata.timestamp).toDateString()}):
"${connection.nodeA.content}"

Node B (from ${connection.nodeB.metadata.platform} on ${new Date(connection.nodeB.metadata.timestamp).toDateString()}):
"${connection.nodeB.content}"

Generate a concise, insightful connection between these two memories. Use second-person ("you"). Be specific.

Format:
"Interesting connection: In your chat with [Platform A], you mentioned [X]. This seems related to [Y] from your [Platform B] conversation. [Specific insight]."
`;

    const synthesisText = await this.llm.generate(prompt, { maxTokens: 150 });

    return {
      preview: synthesisText.slice(0, 100) + '...',
      fullText: synthesisText,
      nodeIds: [connection.nodeA.id, connection.nodeB.id],
      ahaScore: connection.ahaScore,
    };
  }
}
```

### 2.4 The Resurrection Engine (Day 14+ Recovery)

**Purpose**: Re-engage churned users with "Loss of Asset" warnings.

**File**: `src/triggers/ResurrectionEngine.ts`

```typescript
class ResurrectionEngine {
  // Runs hourly: identifies users at risk of churn
  async identifyChurnRisk(): Promise<void> {
    const usersAtRisk = await this.getUsersWithInactivity({ days: 3 });

    for (const user of usersAtRisk) {
      await this.attemptResurrection(user.id);
    }
  }

  async attemptResurrection(userId: string): Promise<void> {
    // 1. Detect "Knowledge Collision"
    const collision = await this.detectKnowledgeCollision(userId);

    if (collision.found) {
      await this.sendResurrectionTrigger(userId, 'knowledge_collision', collision);
      return;
    }

    // 2. Calculate "Amnesia Tax"
    const amnesiaMetrics = await this.calculateAmnesiaTax(userId);

    if (amnesiaMetrics.hoursInvested > 2) {
      await this.sendResurrectionTrigger(userId, 'amnesia_tax', amnesiaMetrics);
      return;
    }

    // 3. Generate "Serendipity Synthesis"
    const serendipity = await this.generateSerendipitySynthesis(userId);

    if (serendipity) {
      await this.sendResurrectionTrigger(userId, 'serendipity', serendipity);
      return;
    }
  }

  async detectKnowledgeCollision(userId: string): Promise<KnowledgeCollision> {
    // Monitor browser activity (via extension)
    const recentActivity = await this.getBrowserActivity(userId, { hours: 72 });

    // Find projects/topics user is currently working on
    const currentTopics = this.extractTopics(recentActivity);

    // Check if we have indexed memory about these topics
    for (const topic of currentTopics) {
      const existingMemory = await this.searchMemoryGraph(userId, topic);

      if (existingMemory.nodes.length > 3) {
        // User has significant past context on this topic
        const lastQueried = await this.getLastQueryTime(userId, topic);

        if (Date.now() - lastQueried > 72 * 60 * 60 * 1000) {
          // They haven't queried this topic in 72+ hours despite working on it
          return {
            found: true,
            topic: topic,
            memoryCount: existingMemory.nodes.length,
            lastQueried,
            currentActivity: recentActivity.filter(a => a.context.extractedEntities.some(e => e.value === topic)),
          };
        }
      }
    }

    return { found: false };
  }

  async calculateAmnesiaTax(userId: string): Promise<AmnesiaTax> {
    // Calculate how much time user invested in building their memory
    const allInteractions = await this.getUserInteractions(userId);

    const hoursInvested = allInteractions
      .filter(i => i.type === 'correction' || i.type === 'data_ingest')
      .reduce((sum, i) => sum + (i.duration || 0), 0) / 3600000;

    const nodeCount = await this.getMemoryNodeCount(userId);

    return {
      hoursInvested,
      nodeCount,
      estimatedValue: this.estimateMemoryValue(hoursInvested, nodeCount),
    };
  }

  async sendResurrectionTrigger(
    userId: string,
    triggerType: 'knowledge_collision' | 'amnesia_tax' | 'serendipity',
    data: any
  ): Promise<void> {
    let notification: Notification;

    switch (triggerType) {
      case 'knowledge_collision':
        notification = {
          userId,
          type: 'resurrection_collision',
          title: `Your ${data.topic} context is drifting`,
          message: `You built deep knowledge about ${data.topic} ${this.formatTimeSince(data.lastQueried)} ago. Today, I noticed you're working on it again. You're about to recreate context I already have.`,
          action: {
            type: 'inject_context',
            data: {
              topic: data.topic,
              memoryNodes: data.memoryCount,
            },
          },
        };
        break;

      case 'amnesia_tax':
        notification = {
          userId,
          type: 'resurrection_tax',
          title: `You've invested ${Math.round(data.hoursInvested)} hours building this brain`,
          message: `Your ${data.nodeCount} memory nodes are gathering dust. Every day you don't use them is wasted labor. Let me save you time today.`,
          action: {
            type: 'view_memory_asset',
            data: data,
          },
        };
        break;

      case 'serendipity':
        notification = {
          userId,
          type: 'resurrection_serendipity',
          title: 'I found a connection while you were away',
          message: data.preview,
          action: {
            type: 'view_synthesis',
            data: data,
          },
        };
        break;
    }

    await this.sendNotification(notification);
    await this.trackResurrectionAttempt(userId, triggerType);
  }
}
```

---

## Phase 3: Onboarding Flow Implementation

### 3.1 The "Digital Graft" Onboarding

**File**: `src/onboarding/OnboardingFlow.ts`

```typescript
class OnboardingFlow {
  // Minute 1: The Identity Mirror
  async startOnboarding(userId: string): Promise<void> {
    // Skip email/password - use passwordless magic link
    await this.sendMagicLink(userId);

    // Immediately show the Soul Selection
    await this.showSoulSelection(userId);
  }

  async showSoulSelection(userId: string): Promise<SoulSelection> {
    const questions = [
      {
        id: 'project_focus',
        question: 'What is the one project you can't stop thinking about right now?',
        type: 'text',
        placeholder: 'e.g., Building a habit-tracking app',
      },
      {
        id: 'communication_models',
        question: 'Who are the 3 people whose communication style you admire most?',
        type: 'text',
        placeholder: 'e.g., Paul Graham, my mentor Sarah, Derek Sivers',
      },
      {
        id: 'niche_knowledge',
        question: 'What is a piece of niche knowledge you possess that generic AI always gets wrong?',
        type: 'text',
        placeholder: 'e.g., Obscure programming language quirks, domain expertise',
      },
    ];

    const answers = await this.collectUserInput(userId, questions);

    // Store as high-value seeds for the memory graph
    await this.seedMemoryGraph(userId, answers);

    return answers;
  }

  // Minutes 2-3: The Memory Ingest
  async startMemoryIngest(userId: string): Promise<void> {
    // Show OAuth connection options
    const availableSources = [
      { id: 'slack', name: 'Slack (DMs to self)', icon: 'slack.svg' },
      { id: 'notion', name: 'Notion (Personal Workspace)', icon: 'notion.svg' },
      { id: 'browser', name: 'Browser History (Last 24 hours)', icon: 'chrome.svg' },
      { id: 'gmail', name: 'Gmail (Sent folder)', icon: 'gmail.svg' },
    ];

    const selectedSource = await this.showSourceSelection(userId, availableSources);

    // Start OAuth flow
    const authResult = await this.initiateOAuth(userId, selectedSource.id);

    if (authResult.success) {
      // Start background data pull with visual feedback
      await this.startSynapseMappingVisual(userId, selectedSource.id);

      // Process data in background
      this.processDataIngest(userId, selectedSource.id, authResult.accessToken);
    }
  }

  async startSynapseMappingVisual(userId: string, sourceId: string): Promise<void> {
    // WebSocket-based real-time visual
    const ws = await this.getWebSocket(userId);

    // Send periodic updates as data is processed
    const stream = this.ingestDataStream(userId, sourceId);

    for await (const batch of stream) {
      // Extract keywords from this batch
      const keywords = batch.flatMap(node => node.metadata.entities.map(e => e.value));

      // Send to frontend for visualization
      ws.send({
        type: 'synapse_update',
        keywords: keywords.slice(0, 10), // Top 10 keywords
        progress: batch.processedCount / batch.totalCount,
      });
    }
  }

  // Minute 4: The Mirror Test (Identity Brief)
  async generateIdentityBrief(userId: string): Promise<IdentityBrief> {
    // Wait for data ingest to have at least 50 nodes
    await this.waitForMinimumNodes(userId, 50);

    // Run the Mirror Algorithm (see section 3.2)
    const brief = await this.mirrorAlgorithm.generateBrief(userId);

    // Display to user
    await this.showIdentityBrief(userId, brief);

    return brief;
  }

  // Minute 5: The Partial Completion (The Hook)
  async createZeigarnikHook(userId: string): Promise<void> {
    // Find unresolved loops in the ingested data
    const unresolvedLoops = await this.findUnresolvedLoops(userId);

    if (unresolvedLoops.length > 0) {
      const topLoop = unresolvedLoops[0];

      // Ask a completion question
      await this.askUserQuestion(userId, {
        question: `I see you were researching "${topLoop.projectName}" ${this.formatTimeSince(topLoop.lastMentionedAt)} but didn't finish. Should I keep a 'watch' on new developments, or did you complete that thought?`,
        options: [
          { id: 'watch', label: 'Yes, keep watching' },
          { id: 'complete', label: 'I finished that' },
          { id: 'custom', label: 'Other' },
        ],
      });

      // Create open loop in user's mind
      await this.createTask(userId, {
        type: 'unresolved_thought',
        projectName: topLoop.projectName,
        status: 'pending_user_input',
      });
    }
  }
}
```

### 3.2 The Mirror Algorithm (Identity Brief)

**File**: `src/onboarding/MirrorAlgorithm.ts`

```typescript
class MirrorAlgorithm {
  async generateBrief(userId: string): Promise<IdentityBrief> {
    // Step 1: High-Signal Entity Extraction
    const highSignalEntities = await this.extractHighSignalEntities(userId);

    // Step 2: The "Blind Spot" Search (unresolved loops)
    const blindSpots = await this.findBlindSpots(userId);

    // Step 3: Personality Mirroring
    const userProfile = await this.analyzePersonality(userId);

    // Generate the brief sections
    const section1 = await this.generateStateOfMind(highSignalEntities, userProfile);
    const section2 = await this.generateSyntacticLeap(blindSpots, userProfile);
    const section3 = await this.generateProactiveProvision(section2.connection, userProfile);

    return {
      stateOfMind: section1,
      syntacticLeap: section2,
      proactiveProvision: section3,
      timestamp: Date.now(),
    };
  }

  async extractHighSignalEntities(userId: string): Promise<HighSignalEntity[]> {
    const nodes = await this.getRecentNodes(userId, { days: 7 });

    const entities: HighSignalEntity[] = [];

    for (const node of nodes) {
      // Intensity filter: look for specific phrases
      const intensityPhrases = [
        'I need to', 'I\'m worried about', 'What if we', 'sent to self',
      ];

      for (const phrase of intensityPhrases) {
        if (node.content.toLowerCase().includes(phrase.toLowerCase())) {
          entities.push({
            nodeId: node.id,
            content: node.content,
            intensityScore: 0.9,
            source: node.metadata.source,
            timestamp: node.metadata.timestamp,
          });
        }
      }

      // Novelty filter: flag first-time technical terms
      for (const entity of node.metadata.entities) {
        const isFirstMention = await this.isFirstMention(userId, entity.value);

        if (isFirstMention && entity.type === 'concept') {
          entities.push({
            nodeId: node.id,
            content: entity.value,
            intensityScore: 0.7,
            novelty: true,
            source: node.metadata.source,
            timestamp: node.metadata.timestamp,
          });
        }
      }
    }

    return entities.sort((a, b) => b.intensityScore - a.intensityScore);
  }

  async findBlindSpots(userId: string): Promise<BlindSpot[]> {
    // Logic: Find mentioned projects without associated documents
    const allNodes = await this.getUserNodes(userId);

    const projectMentions = allNodes.filter(n =>
      n.nodeType === 'entity' && n.metadata.entities.some(e => e.type === 'project')
    );

    const blindSpots: BlindSpot[] = [];

    for (const mention of projectMentions) {
      const projectName = mention.metadata.entities.find(e => e.type === 'project')?.value;

      if (!projectName) continue;

      // Check if there are documents/notes about this project
      const relatedDocs = await this.searchNodes(userId, {
        query: projectName,
        nodeTypes: ['document', 'note'],
      });

      if (relatedDocs.length === 0) {
        // Unresolved loop detected
        blindSpots.push({
          projectName,
          firstMentionedAt: mention.metadata.timestamp,
          mentionCount: await this.countMentions(userId, projectName),
          sources: [mention.metadata.source],
        });
      }
    }

    return blindSpots;
  }

  async analyzePersonality(userId: string): Promise<UserProfile> {
    const nodes = await this.getUserNodes(userId);

    // Analyze tone and style
    const allText = nodes.map(n => n.content).join(' ');

    // Simple heuristics (can be replaced with ML model)
    const formality = this.detectFormality(allText);
    const verbosity = this.detectVerbosity(nodes);
    const preferredFormat = this.detectFormat(allText);
    const commonPhrases = this.extractCommonPhrases(allText);

    return {
      userId,
      communicationStyle: {
        formality,
        verbosity,
        preferredFormat,
        commonPhrases,
        jargonTerms: this.extractJargon(allText),
      },
      workPatterns: await this.analyzeWorkPatterns(userId, nodes),
      correctionHistory: [],
    };
  }

  async generateStateOfMind(
    entities: HighSignalEntity[],
    profile: UserProfile
  ): Promise<string> {
    // Find dominant focus
    const entityFrequency = new Map<string, number>();

    for (const entity of entities) {
      const key = entity.content;
      entityFrequency.set(key, (entityFrequency.get(key) || 0) + entity.intensityScore);
    }

    const topFocus = Array.from(entityFrequency.entries())
      .sort((a, b) => b[1] - a[1])[0];

    const focusPercentage = Math.round((topFocus[1] / entities.length) * 100);

    // Find unresolved constraint
    const constraint = entities.find(e =>
      e.content.includes('worried') || e.content.includes('problem')
    );

    // Mirror the user's tone
    const brief = profile.communicationStyle.formality > 0.5
      ? `Currently, your cognitive load is focused ${focusPercentage}% on ${topFocus[0]}. You have mentioned the ${constraint?.content} three times across your notes, but it appears you have not yet defined a solution for it.`
      : `Right now, you're about ${focusPercentage}% focused on ${topFocus[0]}. You've mentioned ${constraint?.content} three times, but I don't see a clear solution yet.`;

    return brief;
  }

  async generateSyntacticLeap(
    blindSpots: BlindSpot[],
    profile: UserProfile
  ): Promise<{ connection: string; nodeIds: string[] }> {
    // Find cross-platform connections
    if (blindSpots.length === 0) {
      return { connection: '', nodeIds: [] };
    }

    const topBlindSpot = blindSpots[0];

    // Search for potential solutions in other conversations
    const potentialSolutions = await this.searchCrossPlatform(
      profile.userId,
      topBlindSpot.projectName
    );

    if (potentialSolutions.length > 0) {
      const solution = potentialSolutions[0];

      const connection = `Interesting connection: On ${new Date(topBlindSpot.firstMentionedAt).toLocaleDateString()}, you mentioned "${topBlindSpot.projectName}" in ${topBlindSpot.sources[0]}. This looks like it could relate to ${solution.content} from your ${solution.source} conversation on ${new Date(solution.timestamp).toLocaleDateString()}.`;

      return {
        connection,
        nodeIds: [solution.nodeId],
      };
    }

    return { connection: '', nodeIds: [] };
  }

  async generateProactiveProvision(
    connection: string,
    profile: UserProfile
  ): Promise<string> {
    if (!connection) {
      return '';
    }

    return `I've pre-indexed a context window for Claude 3.5 that combines your notes on both. Ready to solve this now?`;
  }
}
```

---

## Phase 4: Delivery Mechanisms

### 4.1 Notification System

**File**: `src/notifications/NotificationService.ts`

```typescript
class NotificationService {
  async sendNotification(notification: Notification): Promise<void> {
    // Choose delivery channel
    const channel = await this.selectChannel(notification.userId, notification.type);

    switch (channel) {
      case 'push':
        await this.sendPush(notification);
        break;
      case 'email':
        await this.sendEmail(notification);
        break;
      case 'in_app':
        await this.sendInApp(notification);
        break;
      case 'extension':
        await this.sendToExtension(notification);
        break;
    }

    // Track delivery
    await this.trackNotification(notification, channel);
  }

  async selectChannel(
    userId: string,
    notificationType: string
  ): Promise<DeliveryChannel> {
    // Get user preferences
    const prefs = await this.getUserPreferences(userId);

    // Get user's current context
    const context = await this.getUserContext(userId);

    // Decision logic
    if (context.isActive && context.hasExtension) {
      return 'extension'; // Most immediate
    } else if (notificationType.includes('resurrection')) {
      return 'email'; // More persistent
    } else if (prefs.allowPush) {
      return 'push';
    } else {
      return 'in_app';
    }
  }

  async sendToExtension(notification: Notification): Promise<void> {
    // Send via WebSocket to browser extension
    const ws = await this.getExtensionWebSocket(notification.userId);

    ws.send({
      type: 'notification',
      payload: {
        title: notification.title,
        message: notification.message,
        action: notification.action,
        timestamp: Date.now(),
      },
    });
  }
}
```

### 4.2 Browser Extension Integration

**Extension Manifest** (`src/extensions/browser-extension/manifest.json`):

```json
{
  "manifest_version": 3,
  "name": "KNIRV Controller - Digital Brain",
  "version": "1.0.0",
  "permissions": [
    "activeTab",
    "storage",
    "notifications",
    "tabs",
    "webNavigation"
  ],
  "background": {
    "service_worker": "background/index.js"
  },
  "content_scripts": [
    {
      "matches": ["<all_urls>"],
      "js": ["content-scripts/page-monitor.js"]
    }
  ],
  "commands": {
    "whisper-capture": {
      "suggested_key": {
        "default": "Alt+Space",
        "mac": "Alt+Space"
      },
      "description": "Quick capture thoughts"
    }
  },
  "action": {
    "default_popup": "popup/index.html"
  }
}
```

---

## Phase 5: Metrics & Monitoring

### 5.1 Tracking Schema

```typescript
interface MetricEvent {
  userId: string;
  eventType:
    | 'trigger_fired'
    | 'notification_sent'
    | 'notification_clicked'
    | 'memory_surfaced'
    | 'contextual_recall'
    | 'correction_made'
    | 'resurrection_attempt'
    | 'resurrection_success';
  timestamp: number;
  metadata: Record<string, any>;
}

interface UserJourney {
  userId: string;
  signupDate: number;
  currentDay: number;
  milestones: {
    day1_ingest_complete: boolean;
    day2_first_proactive: boolean;
    day3_cross_model: boolean;
    day4_customization: boolean;
    day5_friction_test: boolean;
    day6_synthesis: boolean;
    day7_habit_loop: boolean;
  };
  retention: {
    l7: boolean;  // Active in last 7 days
    dau: boolean; // Active today
    mau: boolean; // Active this month
  };
  engagement: {
    totalContextualRecalls: number;
    correctionsCount: number;
    memoryNodeCount: number;
    lastActiveAt: number;
  };
}
```

### 5.2 Dashboard Metrics

**File**: `src/analytics/MetricsDashboard.ts`

```typescript
class MetricsDashboard {
  async getProductHealth(): Promise<ProductHealthMetrics> {
    return {
      l7Retention: await this.calculateL7Retention(),
      contextualRecallRate: await this.calculateRecallRate(),
      correctionRatio: await this.calculateCorrectionRatio(),
      dauMau: await this.calculateDAUMAU(),
      resurrectionRate: await this.calculateResurrectionRate(),
    };
  }

  async calculateL7Retention(): Promise<number> {
    const totalUsers = await this.getUserCount({ signupBefore: Date.now() - 7 * 24 * 60 * 60 * 1000 });
    const activeUsers = await this.getActiveUserCount({ days: 7 });

    return activeUsers / totalUsers;
  }

  async calculateRecallRate(): Promise<number> {
    // Average contextual recalls per user per day
    const events = await this.getMetricEvents({
      eventType: 'contextual_recall',
      days: 7,
    });

    const uniqueUsers = new Set(events.map(e => e.userId)).size;

    return events.length / uniqueUsers / 7; // Per day
  }

  async calculateResurrectionRate(): Promise<number> {
    const attempts = await this.getMetricEvents({
      eventType: 'resurrection_attempt',
      days: 30,
    });

    const successes = await this.getMetricEvents({
      eventType: 'resurrection_success',
      days: 30,
    });

    return successes.length / attempts.length;
  }
}
```

---

## Phase 6: API Endpoints

### 6.1 New Trigger API Routes

**File**: `src/api/routes/triggers.ts`

```typescript
// GET /api/triggers/context
// Returns current user context for debugging
router.get('/context', async (req, res) => {
  const userId = req.user.id;
  const context = await contextService.getCurrentContext(userId);
  res.json(context);
});

// POST /api/triggers/manual
// Manually trigger a specific trigger type (for testing)
router.post('/manual', async (req, res) => {
  const { userId, triggerType } = req.body;

  switch (triggerType) {
    case 'proactive_surface':
      await proactiveSurfaceEngine.checkSurfaceOpportunities(userId);
      break;
    case 'synthesis':
      await synthesisEngine.generateDailySynthesis(userId);
      break;
    case 'resurrection':
      await resurrectionEngine.attemptResurrection(userId);
      break;
  }

  res.json({ success: true });
});

// GET /api/triggers/history
// Get trigger history for user
router.get('/history', async (req, res) => {
  const userId = req.user.id;
  const history = await metricService.getMetricEvents({
    userId,
    eventType: 'trigger_fired',
    limit: 50,
  });

  res.json(history);
});

// POST /api/onboarding/start
// Start the onboarding flow
router.post('/onboarding/start', async (req, res) => {
  const userId = req.user.id;
  await onboardingFlow.startOnboarding(userId);
  res.json({ success: true });
});

// POST /api/onboarding/soul-selection
// Submit soul selection answers
router.post('/onboarding/soul-selection', async (req, res) => {
  const { userId, answers } = req.body;
  await onboardingFlow.seedMemoryGraph(userId, answers);
  res.json({ success: true });
});

// POST /api/onboarding/identity-brief
// Generate identity brief
router.post('/onboarding/identity-brief', async (req, res) => {
  const userId = req.user.id;
  const brief = await onboardingFlow.generateIdentityBrief(userId);
  res.json(brief);
});
```

---

## Phase 7: Background Workers

### 7.1 Worker Architecture

**File**: `src/workers/TriggerWorker.ts`

```typescript
class TriggerWorker {
  private jobs = [
    { name: 'proactive_surface', interval: 5 * 60 * 1000 },      // Every 5 min
    { name: 'daily_synthesis', interval: 24 * 60 * 60 * 1000 },  // Daily at 8am
    { name: 'resurrection_check', interval: 60 * 60 * 1000 },    // Hourly
    { name: 'habit_tracker', interval: 60 * 60 * 1000 },         // Hourly
  ];

  async start(): Promise<void> {
    for (const job of this.jobs) {
      this.scheduleJob(job);
    }
  }

  private scheduleJob(job: { name: string; interval: number }): void {
    setInterval(async () => {
      try {
        await this.executeJob(job.name);
      } catch (error) {
        console.error(`Job ${job.name} failed:`, error);
      }
    }, job.interval);
  }

  private async executeJob(name: string): Promise<void> {
    switch (name) {
      case 'proactive_surface':
        await proactiveSurfaceEngine.run();
        break;
      case 'daily_synthesis':
        await this.runForAllUsers(user => synthesisEngine.generateDailySynthesis(user.id));
        break;
      case 'resurrection_check':
        await resurrectionEngine.identifyChurnRisk();
        break;
      case 'habit_tracker':
        await this.runForAllUsers(user => habitEngine.trackProgress(user.id));
        break;
    }
  }
}
```

---

## Implementation Roadmap

### Week 1-2: Foundation
- [ ] Database schema for memory nodes, edges, user profiles
- [ ] Vector store integration (Qdrant/Pinecone)
- [ ] Entity extraction service
- [ ] Basic context ingestion (chat history upload)

### Week 3-4: Browser Extension
- [ ] Chrome extension with passive monitoring
- [ ] Whisper mode (Option+Space hotkey)
- [ ] WebSocket connection to backend
- [ ] Privacy controls UI

### Week 5-6: Trigger Engines
- [ ] Proactive Surface Engine
- [ ] Habit Formation Engine (Day 1-7 flow)
- [ ] Synthesis Engine
- [ ] Notification service

### Week 7-8: Onboarding Flow
- [ ] Soul Selection UI
- [ ] OAuth integrations (Slack, Notion, Gmail)
- [ ] Synapse Mapping visual
- [ ] Mirror Algorithm implementation
- [ ] Identity Brief generation

### Week 9-10: Resurrection & Metrics
- [ ] Resurrection Engine
- [ ] Knowledge Collision detection
- [ ] Metrics dashboard
- [ ] A/B testing framework

### Week 11-12: Polish & Launch
- [ ] Performance optimization
- [ ] Security audit
- [ ] Beta testing
- [ ] Production deployment

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| **L7 Retention** | >45% | % of users active in last 7 days |
| **Contextual Recall Rate** | 3+ per day | Avg proactive surfaces per user per day |
| **Correction Ratio** | 1:10 | User corrections per 10 interactions |
| **DAU/MAU** | 60% | Daily Active / Monthly Active ratio |
| **Time to First Aha!** | <4 min | Time from signup to first identity brief |
| **Resurrection Rate** | >15% | % of churned users who return |

---

## Next Steps

1. **Review and approve this implementation plan**
2. **Set up database schemas in NebulaDB**
3. **Build MVP browser extension**
4. **Implement basic context ingestion**
5. **Test Day 1 onboarding flow with alpha users**

This architecture transforms KNIRVARENA from a static chat interface into a proactive, habit-forming system that becomes indispensable to users within the first week.
