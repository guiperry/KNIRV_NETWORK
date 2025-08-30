# NebulaDB Migration Guide

This document outlines the complete migration of KNIRVCONTROLLER from SQLite to NebulaDB, providing a modern, TypeScript-first database solution with advanced features.

## Overview

NebulaDB has been successfully integrated into KNIRVCONTROLLER, replacing any existing SQLite implementations with a unified, performant persistence layer. The migration includes:

- ✅ **Core Integration**: Centralized database service with NebulaDB
- ✅ **API Layer**: Chat and Skills API endpoints using NebulaDB
- ✅ **Service Refactoring**: AgentManagementService updated to use persistent storage
- ✅ **Advanced Features**: Full-text search and enhanced client-side caching
- ✅ **Testing**: Comprehensive unit and integration tests
- ✅ **Migration Tools**: Scripts for data migration and backup

## Architecture

### Database Service
- **Location**: `src/core/services/databaseService.ts`
- **Features**: Singleton pattern, schema definitions, helper methods
- **Collections**: Agents, Skills, ChatSessions

### API Endpoints
- **Chat API**: `src/core/api/chat.ts`
- **Skills API**: `src/core/api/skills.ts`
- **Features**: CRUD operations, search, pagination

### Client-Side Caching
- **Location**: `src/hooks/useClientCache.ts`
- **Features**: Offline support, stale-while-revalidate, multiple cache types

## Configuration

### Database Configuration
Configuration is managed through `config/database.json`:

```json
{
  "development": {
    "filePath": "./data/knirvcontroller-dev.db",
    "autoload": true,
    "autosave": true,
    "autosaveInterval": 4000
  },
  "production": {
    "filePath": "./data/knirvcontroller.db",
    "autoload": true,
    "autosave": true,
    "autosaveInterval": 2000,
    "backup": {
      "enabled": true,
      "interval": "0 2 * * *",
      "retentionDays": 30
    }
  }
}
```

## Usage

### Basic Operations

```typescript
import { databaseService } from '../core/services/databaseService';

// Create an agent
const agent = await databaseService.createAgent({
  agentId: 'agent-1',
  name: 'My Agent',
  type: 'wasm',
  status: 'Available',
  // ... other properties
});

// Search skills
const results = await databaseService.searchSkills('machine learning', 10);

// Create chat session
const session = await databaseService.createChatSession({
  title: 'New Chat',
  messages: []
});
```

### Client-Side Caching

```typescript
import { useAgentCache, useSkillCache } from '../hooks/useClientCache';

function MyComponent() {
  const { agents, isLoading, refreshAgents } = useAgentCache();
  const { skills, searchSkills } = useSkillCache();
  
  // Use cached data with automatic refresh
  return (
    <div>
      {isLoading ? 'Loading...' : `${agents.length} agents loaded`}
    </div>
  );
}
```

## Migration

### From SQLite
If you have existing SQLite data, use the migration script:

```bash
npm run db:migrate
```

This script will:
1. Create a backup of existing data
2. Migrate chat sessions, agents, and skills
3. Preserve all relationships and metadata

### Manual Migration
For custom migration needs, extend the `DataMigrator` class in `scripts/migrate-to-nebuladb.ts`.

## Database Management

### Available Scripts

```bash
# Run migration from SQLite
npm run db:migrate

# Create database backup
npm run db:backup

# Check database status
npm run db:status

# Restore from backup (specify file)
npm run db:restore -- ./data/backup-file.db
```

### Backup Strategy

**Development**: Manual backups via npm scripts
**Production**: Automated daily backups via Docker compose

## Deployment

### Docker Deployment
The application includes Docker configuration with persistent volumes:

```bash
# Build and run with Docker Compose
docker-compose up -d

# Data persists in named volume 'knirvcontroller_data'
# Backups are stored in './backups' directory
```

### Environment Variables

```bash
NODE_ENV=production
DATABASE_PATH=/app/data/knirvcontroller.db
```

## Testing

### Unit Tests
```bash
npm test src/core/services/__tests__/databaseService.test.ts
```

### Integration Tests
```bash
npm test src/core/api/__tests__/chat.integration.test.ts
```

### Full Test Suite
```bash
npm run test:all
```

## Performance Considerations

### Indexing
- Full-text search is enabled on skill names and descriptions
- Automatic indexing on unique fields (agentId, skillId)

### Caching
- Client-side caching with configurable TTL
- Stale-while-revalidate strategy for offline support
- Automatic cache invalidation

### Autosave
- Development: 4-second intervals
- Production: 2-second intervals
- Configurable per environment

## Troubleshooting

### Common Issues

**Database file not found**
- Ensure data directory exists and has proper permissions
- Check DATABASE_PATH environment variable

**Migration fails**
- Verify SQLite file format and accessibility
- Check backup creation before migration
- Review migration logs for specific errors

**Performance issues**
- Adjust autosave intervals in configuration
- Monitor cache hit rates
- Consider database file size and disk I/O

### Debug Mode
Enable debug logging by setting:
```bash
DEBUG=nebuladb:*
```

## API Reference

### Database Service Methods

#### Agents
- `createAgent(agentData)` - Create new agent
- `getAgent(agentId)` - Get agent by ID
- `updateAgent(agentId, updateData)` - Update agent
- `deleteAgent(agentId)` - Delete agent
- `listAgents()` - Get all agents

#### Skills
- `createSkill(skillData)` - Create new skill
- `getSkill(skillId)` - Get skill by ID
- `updateSkill(skillId, updateData)` - Update skill
- `deleteSkill(skillId)` - Delete skill
- `listSkills()` - Get all skills
- `searchSkills(term, limit)` - Full-text search

#### Chat Sessions
- `createChatSession(sessionData)` - Create new session
- `getChatSession(sessionId)` - Get session by ID
- `updateChatSession(sessionId, updateData)` - Update session
- `deleteChatSession(sessionId)` - Delete session
- `listChatSessions()` - Get all sessions (sorted by updatedAt)

## Migration Checklist

- [x] Install NebulaDB dependency
- [x] Create centralized database service
- [x] Implement chat API endpoints
- [x] Refactor AgentManagementService
- [x] Add full-text search for skills
- [x] Enhance client-side caching
- [x] Create migration script
- [x] Add comprehensive tests
- [x] Configure deployment with data persistence
- [x] Document usage and troubleshooting

## Next Steps

1. **Monitor Performance**: Track database performance in production
2. **Optimize Queries**: Add indexes for frequently queried fields
3. **Backup Automation**: Implement automated backup verification
4. **Scaling**: Consider sharding strategies for large datasets
5. **Analytics**: Add database usage analytics and monitoring
