import React, { useState, useCallback } from 'react';
import { Code, Zap, Settings, CheckCircle, AlertCircle, Play, Download } from 'lucide-react';

interface SkillTool {
  name: string;
  description: string;
  parameters: ToolParameter[];
  implementation: string;
  sourceType: 'inline' | 'external' | 'template';
}

interface ToolParameter {
  name: string;
  type: string;
  required: boolean;
  description: string;
  defaultValue?: any;
}

interface SkillCompilerProps {
  cognitiveEngine?: any;
  onSkillCompiled?: (result: any) => void;
  onCompilationError?: (error: string) => void;
}

const SkillCompiler: React.FC<SkillCompilerProps> = ({
  cognitiveEngine,
  onSkillCompiled,
  onCompilationError
}) => {
  const [skillName, setSkillName] = useState('');
  const [skillDescription, setSkillDescription] = useState('');
  const [skillAuthor, setSkillAuthor] = useState('');
  const [tools, setTools] = useState<SkillTool[]>([]);
  const [isCompiling, setIsCompiling] = useState(false);
  const [compilationResult, setCompilationResult] = useState<any>(null);
  const [activeTab, setActiveTab] = useState<'basic' | 'tools' | 'advanced'>('basic');

  const addTool = useCallback(() => {
    const newTool: SkillTool = {
      name: `tool${tools.length + 1}`,
      description: '',
      parameters: [],
      implementation: '// Tool implementation\nreturn { result: "Hello from tool!" };',
      sourceType: 'inline'
    };
    setTools([...tools, newTool]);
  }, [tools]);

  const updateTool = useCallback((index: number, updates: Partial<SkillTool>) => {
    const updatedTools = [...tools];
    updatedTools[index] = { ...updatedTools[index], ...updates };
    setTools(updatedTools);
  }, [tools]);

  const removeTool = useCallback((index: number) => {
    setTools(tools.filter((_, i) => i !== index));
  }, [tools]);

  const addParameter = useCallback((toolIndex: number) => {
    const newParameter: ToolParameter = {
      name: 'param',
      type: 'string',
      required: false,
      description: ''
    };
    
    const updatedTools = [...tools];
    updatedTools[toolIndex].parameters.push(newParameter);
    setTools(updatedTools);
  }, [tools]);

  const updateParameter = useCallback((toolIndex: number, paramIndex: number, updates: Partial<ToolParameter>) => {
    const updatedTools = [...tools];
    updatedTools[toolIndex].parameters[paramIndex] = {
      ...updatedTools[toolIndex].parameters[paramIndex],
      ...updates
    };
    setTools(updatedTools);
  }, [tools]);

  const removeParameter = useCallback((toolIndex: number, paramIndex: number) => {
    const updatedTools = [...tools];
    updatedTools[toolIndex].parameters.splice(paramIndex, 1);
    setTools(updatedTools);
  }, [tools]);

  const compileSkill = useCallback(async () => {
    if (!cognitiveEngine || !skillName.trim()) {
      onCompilationError?.('Missing skill name or cognitive engine not available');
      return;
    }

    setIsCompiling(true);
    setCompilationResult(null);

    try {
      const config = {
        skillId: `skill-${skillName.toLowerCase().replace(/[^a-z0-9]/g, '-')}-${Date.now()}`,
        skillName,
        description: skillDescription || 'Custom compiled skill',
        version: '1.0.0',
        author: skillAuthor || 'Unknown',
        tools,
        parameters: {},
        buildTarget: 'typescript' as const,
        optimizationLevel: 'basic' as const
      };

      const result = await cognitiveEngine.compileSkillFromTemplate(config);
      setCompilationResult(result);
      
      if (result.success) {
        onSkillCompiled?.(result);
      } else {
        onCompilationError?.(result.errors?.join(', ') || 'Compilation failed');
      }

    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : 'Compilation failed';
      setCompilationResult({ success: false, errors: [errorMsg] });
      onCompilationError?.(errorMsg);
    } finally {
      setIsCompiling(false);
    }
  }, [cognitiveEngine, skillName, skillDescription, skillAuthor, tools, onSkillCompiled, onCompilationError]);

  const testSkill = useCallback(async () => {
    if (!compilationResult?.success || !cognitiveEngine) return;

    try {
      // Test the compiled skill
      const testResult = await cognitiveEngine.processInput('test input', 'text');
      console.log('Skill test result:', testResult);
    } catch (error) {
      console.error('Skill test failed:', error);
    }
  }, [compilationResult, cognitiveEngine]);

  const downloadSkill = useCallback(() => {
    if (!compilationResult?.success) return;

    const blob = new Blob([compilationResult.compiledCode], { type: 'text/typescript' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${skillName || 'skill'}.ts`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, [compilationResult, skillName]);

  return (
    <div className="bg-white rounded-lg shadow-lg p-6 max-w-4xl mx-auto">
      <div className="flex items-center space-x-3 mb-6">
        <Code className="w-6 h-6 text-blue-600" />
        <h2 className="text-2xl font-bold text-gray-900">TypeScript Skill Compiler</h2>
      </div>

      {/* Tab Navigation */}
      <div className="flex space-x-1 mb-6 bg-gray-100 rounded-lg p-1">
        {[
          { id: 'basic', label: 'Basic Info', icon: Settings },
          { id: 'tools', label: 'Tools', icon: Zap },
          { id: 'advanced', label: 'Advanced', icon: Code }
        ].map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            onClick={() => setActiveTab(id as any)}
            className={`flex items-center space-x-2 px-4 py-2 rounded-md transition-colors ${
              activeTab === id
                ? 'bg-white text-blue-600 shadow-sm'
                : 'text-gray-600 hover:text-gray-900'
            }`}
          >
            <Icon className="w-4 h-4" />
            <span>{label}</span>
          </button>
        ))}
      </div>

      {/* Basic Info Tab */}
      {activeTab === 'basic' && (
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Skill Name *
            </label>
            <input
              type="text"
              value={skillName}
              onChange={(e) => setSkillName(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="My Custom Skill"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              value={skillDescription}
              onChange={(e) => setSkillDescription(e.target.value)}
              rows={3}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Describe what this skill does..."
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Author
            </label>
            <input
              type="text"
              value={skillAuthor}
              onChange={(e) => setSkillAuthor(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Your Name"
            />
          </div>
        </div>
      )}

      {/* Tools Tab */}
      {activeTab === 'tools' && (
        <div className="space-y-6">
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-medium text-gray-900">Skill Tools</h3>
            <button
              onClick={addTool}
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
            >
              Add Tool
            </button>
          </div>

          {tools.length === 0 ? (
            <div className="text-center py-8 text-gray-500">
              <Code className="w-12 h-12 mx-auto mb-4 text-gray-300" />
              <p>No tools defined. Add a tool to get started.</p>
            </div>
          ) : (
            <div className="space-y-4">
              {tools.map((tool, toolIndex) => (
                <div key={toolIndex} className="border border-gray-200 rounded-lg p-4">
                  <div className="flex items-center justify-between mb-4">
                    <h4 className="text-md font-medium text-gray-900">Tool {toolIndex + 1}</h4>
                    <button
                      onClick={() => removeTool(toolIndex)}
                      className="text-red-600 hover:text-red-800"
                    >
                      Remove
                    </button>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">
                        Tool Name
                      </label>
                      <input
                        type="text"
                        value={tool.name}
                        onChange={(e) => updateTool(toolIndex, { name: e.target.value })}
                        className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">
                        Description
                      </label>
                      <input
                        type="text"
                        value={tool.description}
                        onChange={(e) => updateTool(toolIndex, { description: e.target.value })}
                        className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </div>
                  </div>

                  <div className="mb-4">
                    <div className="flex items-center justify-between mb-2">
                      <label className="block text-sm font-medium text-gray-700">
                        Parameters
                      </label>
                      <button
                        onClick={() => addParameter(toolIndex)}
                        className="text-sm text-blue-600 hover:text-blue-800"
                      >
                        Add Parameter
                      </button>
                    </div>
                    
                    {tool.parameters.map((param, paramIndex) => (
                      <div key={paramIndex} className="flex items-center space-x-2 mb-2">
                        <input
                          type="text"
                          value={param.name}
                          onChange={(e) => updateParameter(toolIndex, paramIndex, { name: e.target.value })}
                          placeholder="Parameter name"
                          className="flex-1 px-2 py-1 border border-gray-300 rounded text-sm"
                        />
                        <select
                          value={param.type}
                          onChange={(e) => updateParameter(toolIndex, paramIndex, { type: e.target.value })}
                          className="px-2 py-1 border border-gray-300 rounded text-sm"
                        >
                          <option value="string">string</option>
                          <option value="number">number</option>
                          <option value="boolean">boolean</option>
                          <option value="any">any</option>
                        </select>
                        <label className="flex items-center text-sm">
                          <input
                            type="checkbox"
                            checked={param.required}
                            onChange={(e) => updateParameter(toolIndex, paramIndex, { required: e.target.checked })}
                            className="mr-1"
                          />
                          Required
                        </label>
                        <button
                          onClick={() => removeParameter(toolIndex, paramIndex)}
                          className="text-red-600 hover:text-red-800 text-sm"
                        >
                          Remove
                        </button>
                      </div>
                    ))}
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      Implementation
                    </label>
                    <textarea
                      value={tool.implementation}
                      onChange={(e) => updateTool(toolIndex, { implementation: e.target.value })}
                      rows={6}
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm"
                      placeholder="// Tool implementation code..."
                    />
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Advanced Tab */}
      {activeTab === 'advanced' && (
        <div className="space-y-4">
          <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
            <h3 className="text-lg font-medium text-yellow-800 mb-2">Advanced Configuration</h3>
            <p className="text-yellow-700">
              Advanced compilation options will be available in future versions.
              Current compilation uses TypeScript with basic optimization.
            </p>
          </div>
        </div>
      )}

      {/* Compilation Results */}
      {compilationResult && (
        <div className="mt-6 p-4 border rounded-lg">
          <div className="flex items-center space-x-2 mb-2">
            {compilationResult.success ? (
              <CheckCircle className="w-5 h-5 text-green-500" />
            ) : (
              <AlertCircle className="w-5 h-5 text-red-500" />
            )}
            <span className={`font-medium ${compilationResult.success ? 'text-green-700' : 'text-red-700'}`}>
              {compilationResult.success ? 'Compilation Successful' : 'Compilation Failed'}
            </span>
          </div>
          
          {compilationResult.success && (
            <div className="text-sm text-gray-600 space-y-1">
              <p>Compilation Time: {compilationResult.metadata.compilationTime}ms</p>
              <p>Code Size: {compilationResult.metadata.codeSize} characters</p>
              <p>Optimization: {compilationResult.metadata.optimizationLevel}</p>
            </div>
          )}
          
          {compilationResult.errors && (
            <div className="mt-2">
              <p className="text-sm font-medium text-red-700">Errors:</p>
              <ul className="text-sm text-red-600 list-disc list-inside">
                {compilationResult.errors.map((error: string, index: number) => (
                  <li key={index}>{error}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      {/* Action Buttons */}
      <div className="flex items-center justify-end space-x-3 mt-6 pt-6 border-t">
        {compilationResult?.success && (
          <>
            <button
              onClick={testSkill}
              className="px-4 py-2 text-blue-600 border border-blue-600 rounded-md hover:bg-blue-50 transition-colors flex items-center space-x-2"
            >
              <Play className="w-4 h-4" />
              <span>Test Skill</span>
            </button>
            <button
              onClick={downloadSkill}
              className="px-4 py-2 text-green-600 border border-green-600 rounded-md hover:bg-green-50 transition-colors flex items-center space-x-2"
            >
              <Download className="w-4 h-4" />
              <span>Download</span>
            </button>
          </>
        )}
        
        <button
          onClick={compileSkill}
          disabled={!skillName.trim() || isCompiling}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center space-x-2"
        >
          {isCompiling ? (
            <>
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white" />
              <span>Compiling...</span>
            </>
          ) : (
            <>
              <Zap className="w-4 h-4" />
              <span>Compile Skill</span>
            </>
          )}
        </button>
      </div>
    </div>
  );
};

export default SkillCompiler;
