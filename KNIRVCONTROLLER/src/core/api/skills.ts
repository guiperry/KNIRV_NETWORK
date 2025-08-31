import { Request, Response } from 'express';
import { databaseService } from '../services/databaseService';

export interface Skill {
  _id?: string;
  skillId: string;
  name: string;
  description?: string;
  loraAdapter?: {
    rank: number;
    alpha: number;
    weightsUri: string;
  };
  version: number;
  createdAt: Date;
  updatedAt?: Date;
}

/**
 * Get all skills
 */
export const getSkills = async (req: Request, res: Response) => {
  try {
    const skills = await databaseService.listSkills();
    res.json({ skills });
  } catch (error) {
    console.error('Error fetching skills:', error);
    res.status(500).json({ error: 'Failed to fetch skills' });
  }
};

/**
 * Get a specific skill by ID
 */
export const getSkill = async (req: Request, res: Response) => {
  try {
    const { skillId } = req.params;
    const skill = await databaseService.getSkill(skillId);
    
    if (!skill) {
      return res.status(404).json({ error: 'Skill not found' });
    }
    
    res.json({ skill });
  } catch (error) {
    console.error('Error fetching skill:', error);
    res.status(500).json({ error: 'Failed to fetch skill' });
  }
};

/**
 * Create a new skill
 */
export const createSkill = async (req: Request, res: Response) => {
  try {
    const { skillId, name, description, loraAdapter, version } = req.body;
    
    if (!skillId || !name) {
      return res.status(400).json({ error: 'skillId and name are required' });
    }
    
    const skillData = {
      skillId,
      name,
      description,
      loraAdapter,
      version: version || 1,
      updatedAt: new Date().toISOString(),
    };
    
    const newSkill = await databaseService.createSkill(skillData);
    res.status(201).json({ skill: newSkill });
  } catch (error) {
    console.error('Error creating skill:', error);
    res.status(500).json({ error: 'Failed to create skill' });
  }
};

/**
 * Update a skill
 */
export const updateSkill = async (req: Request, res: Response) => {
  try {
    const { skillId } = req.params;
    const { name, description, loraAdapter, version } = req.body;
    
    const updateData: Record<string, unknown> = {
      updatedAt: new Date(),
    };
    
    if (name !== undefined) updateData.name = name;
    if (description !== undefined) updateData.description = description;
    if (loraAdapter !== undefined) updateData.loraAdapter = loraAdapter;
    if (version !== undefined) updateData.version = version;
    
    const updatedSkill = await databaseService.updateSkill(skillId, updateData);
    
    if (!updatedSkill) {
      return res.status(404).json({ error: 'Skill not found' });
    }
    
    res.json({ skill: updatedSkill });
  } catch (error) {
    console.error('Error updating skill:', error);
    res.status(500).json({ error: 'Failed to update skill' });
  }
};

/**
 * Delete a skill
 */
export const deleteSkill = async (req: Request, res: Response) => {
  try {
    const { skillId } = req.params;
    
    const deletedSkill = await databaseService.deleteSkill(skillId);
    
    if (!deletedSkill) {
      return res.status(404).json({ error: 'Skill not found' });
    }
    
    res.json({ message: 'Skill deleted successfully' });
  } catch (error) {
    console.error('Error deleting skill:', error);
    res.status(500).json({ error: 'Failed to delete skill' });
  }
};

/**
 * Search skills using full-text search
 */
export const searchSkills = async (req: Request, res: Response) => {
  try {
    const { term, limit } = req.query;
    
    if (typeof term !== 'string') {
      return res.status(400).json({ error: 'Search term is required' });
    }

    const limitNum = limit ? parseInt(limit as string, 10) : 10;
    
    if (isNaN(limitNum) || limitNum < 1 || limitNum > 100) {
      return res.status(400).json({ error: 'Limit must be a number between 1 and 100' });
    }

    const results = await databaseService.searchSkills(term, limitNum);

    res.json({ 
      results,
      query: term,
      limit: limitNum,
      total: results.length
    });
  } catch (error) {
    console.error('Error searching skills:', error);
    res.status(500).json({ error: 'Failed to search skills' });
  }
};

/**
 * Get skills by LoRA adapter configuration
 */
export const getSkillsByLoRAConfig = async (req: Request, res: Response) => {
  try {
    const { rank, alpha } = req.query;
    
    const allSkills = await databaseService.listSkills();
    
    let filteredSkills = allSkills.filter(skill => skill.loraAdapter);
    
    if (rank) {
      const rankNum = parseInt(rank as string, 10);
      if (!isNaN(rankNum)) {
        filteredSkills = filteredSkills.filter(skill => 
          skill.loraAdapter && skill.loraAdapter.rank === rankNum
        );
      }
    }
    
    if (alpha) {
      const alphaNum = parseFloat(alpha as string);
      if (!isNaN(alphaNum)) {
        filteredSkills = filteredSkills.filter(skill => 
          skill.loraAdapter && skill.loraAdapter.alpha === alphaNum
        );
      }
    }
    
    res.json({ 
      skills: filteredSkills,
      filters: { rank, alpha },
      total: filteredSkills.length
    });
  } catch (error) {
    console.error('Error filtering skills by LoRA config:', error);
    res.status(500).json({ error: 'Failed to filter skills' });
  }
};

/**
 * Get skill statistics
 */
export const getSkillStats = async (req: Request, res: Response) => {
  try {
    const allSkills = await databaseService.listSkills();
    
    const stats = {
      total: allSkills.length,
      withLoRA: allSkills.filter(skill => skill.loraAdapter).length,
      withoutLoRA: allSkills.filter(skill => !skill.loraAdapter).length,
      versions: {} as Record<number, number>,
      averageVersion: 0,
    };
    
    // Calculate version statistics
    allSkills.forEach(skill => {
      const version = skill.version || 1;
      stats.versions[version] = (stats.versions[version] || 0) + 1;
    });
    
    if (allSkills.length > 0) {
      stats.averageVersion = allSkills.reduce((sum, skill) => sum + (skill.version || 1), 0) / allSkills.length;
    }
    
    res.json({ stats });
  } catch (error) {
    console.error('Error getting skill statistics:', error);
    res.status(500).json({ error: 'Failed to get skill statistics' });
  }
};
