'use client';

import React, { useState, useEffect } from 'react';
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Slider } from "@/components/ui/slider";
import { Switch } from "@/components/ui/switch";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { Brain, Zap, Settings, Sparkles, Plus, Trash2, Server, Code, TestTube, Upload, FileText, CheckCircle, XCircle, Download, Share, Key, Loader2, Cpu, Database, Target, Github, ExternalLink, Search } from "lucide-react";
import { useToast } from "@/hooks/use-toast";
import { useAuth } from "@/contexts/AuthContext";
import { 
  CortexModelCompiler, 
  ModelTemplate, 
  ModelConfiguration, 
  ModelCompilationRequest,
  DEFAULT_SLM_TEMPLATES,
  cortexModelCompiler 
} from "@/lib/cortex-compiler/CortexModelCompiler";

export interface ModelConfigProps {
  connectedApp: {url: string, name: string, type: string} | null;
  onConfigured: (config: ModelConfiguration) => void;
  onConnectGitHub?: () => void;
  isActive: boolean;
  downloadModalOpen: boolean;
  setDownloadModalOpen: (open: boolean) => void;
  onDownload: (platform: 'windows' | 'mac' | 'linux') => void;
  settingsModalOpen: boolean;
  setSettingsModalOpen: (open: boolean) => void;
}

const ModelConfig = ({
  connectedApp,
  onConfigured,
  onConnectGitHub,
  isActive,
  downloadModalOpen,
  setDownloadModalOpen,
  onDownload,
  settingsModalOpen,
  setSettingsModalOpen
}: ModelConfigProps) => {
  const { toast } = useToast();
  const { isAuthenticated } = useAuth();

  // Model selection state
  const [modelSource, setModelSource] = useState<'template' | 'github' | 'huggingface'>('template');
  const [showImportSection, setShowImportSection] = useState(false);
  const [importUrl, setImportUrl] = useState('');
  const [isImporting, setIsImporting] = useState(false);
  const [importedModel, setImportedModel] = useState<any>(null);

  // Model configuration state
  const [selectedTemplate, setSelectedTemplate] = useState<ModelTemplate | null>(null);
  const [modelName, setModelName] = useState('');
  const [modelDescription, setModelDescription] = useState('');
  const [learningRate, setLearningRate] = useState([0.001]);
  const [batchSize, setBatchSize] = useState([32]);
  const [epochs, setEpochs] = useState([10]);
  const [optimizer, setOptimizer] = useState<'adam' | 'sgd' | 'adamw'>('adam');
  const [enableLoRA, setEnableLoRA] = useState(false);
  const [loraRank, setLoraRank] = useState([16]);
  const [loraAlpha, setLoraAlpha] = useState([32]);
  const [enableQuantization, setEnableQuantization] = useState(false);
  const [quantizationBits, setQuantizationBits] = useState<4 | 8 | 16>(8);
  const [exportTargets, setExportTargets] = useState<('cortex_wasm' | 'onnx' | 'safetensors' | 'pytorch')[]>(['cortex_wasm']);
  const [deploymentTargets, setDeploymentTargets] = useState({
    knirvcontroller: true,
    knirvengine: false,
    cloud_hosting: null as any
  });

  // Training data state
  const [trainingData, setTrainingData] = useState<string>('');
  const [uploadedFiles, setUploadedFiles] = useState<File[]>([]);

  // UI state
  const [activeTab, setActiveTab] = useState('model-selection');
  const [isConfiguring, setIsConfiguring] = useState(false);

  const handleTemplateSelect = (template: ModelTemplate) => {
    setSelectedTemplate(template);
    setModelName(`${template.name} Custom`);
    setModelDescription(`Custom ${template.description.toLowerCase()}`);

    // Set reasonable defaults based on template
    if (template.size === 'nano') {
      setEpochs([5]);
      setBatchSize([16]);
    } else if (template.size === 'small') {
      setEpochs([20]);
      setBatchSize([64]);
    }

    setActiveTab('configuration');
  };

  const handleImportModel = async () => {
    if (!importUrl.trim()) {
      toast({
        title: "Import Error",
        description: "Please enter a valid URL",
        variant: "destructive"
      });
      return;
    }

    setIsImporting(true);

    try {
      // Simulate import process
      await new Promise(resolve => setTimeout(resolve, 2000));

      // Create a mock imported model
      const importedModelData = {
        name: importUrl.split('/').pop() || 'Imported Model',
        description: `Imported from ${modelSource}`,
        url: importUrl,
        source: modelSource
      };

      setImportedModel(importedModelData);
      setModelName(importedModelData.name);
      setModelDescription(importedModelData.description);

      toast({
        title: "Model Imported",
        description: `Successfully imported model from ${modelSource}`,
      });

      setActiveTab('configuration');
    } catch (error) {
      toast({
        title: "Import Failed",
        description: `Failed to import model: ${error}`,
        variant: "destructive"
      });
    } finally {
      setIsImporting(false);
    }
  };

  const resetImportSection = () => {
    setShowImportSection(false);
    setImportUrl('');
    setImportedModel(null);
  };

  const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files || []);
    setUploadedFiles(prev => [...prev, ...files]);
    
    toast({
      title: "Files uploaded",
      description: `Added ${files.length} file(s) to training data`,
    });
  };

  const removeFile = (index: number) => {
    setUploadedFiles(prev => prev.filter((_, i) => i !== index));
  };

  const validateConfiguration = (): { valid: boolean; errors: string[] } => {
    const errors: string[] = [];

    if (!selectedTemplate) {
      errors.push('Please select a model template');
    }

    if (!modelName.trim()) {
      errors.push('Model name is required');
    }

    if (!modelDescription.trim()) {
      errors.push('Model description is required');
    }

    if (trainingData.trim().length === 0 && uploadedFiles.length === 0) {
      errors.push('Training data is required (either text input or uploaded files)');
    }

    if (exportTargets.length === 0) {
      errors.push('At least one export target must be selected');
    }

    return {
      valid: errors.length === 0,
      errors
    };
  };

  const handleConfigure = async () => {
    const validation = validateConfiguration();
    
    if (!validation.valid) {
      toast({
        title: "Configuration Error",
        description: validation.errors.join(', '),
        variant: "destructive",
      });
      return;
    }

    if (!selectedTemplate) return;

    setIsConfiguring(true);

    try {
      const modelConfig: ModelConfiguration = {
        template: selectedTemplate,
        name: modelName,
        description: modelDescription,
        training_config: {
          learning_rate: learningRate[0],
          batch_size: batchSize[0],
          epochs: epochs[0],
          optimizer: optimizer,
        },
        lora_config: enableLoRA ? {
          enabled: true,
          rank: loraRank[0],
          alpha: loraAlpha[0],
          target_modules: ['attention', 'feed_forward']
        } : undefined,
        quantization: enableQuantization ? {
          enabled: true,
          bits: quantizationBits,
          method: 'dynamic'
        } : undefined,
        export_targets: exportTargets
      };

      // Validate with compiler
      const compilerValidation = cortexModelCompiler.validateConfiguration(modelConfig);
      
      if (!compilerValidation.valid) {
        throw new Error(compilerValidation.errors.join(', '));
      }

      onConfigured(modelConfig);

      toast({
        title: "Model Configured",
        description: `${modelName} is ready for training and compilation`,
      });

    } catch (error) {
      toast({
        title: "Configuration Failed",
        description: `Error: ${error}`,
        variant: "destructive",
      });
    } finally {
      setIsConfiguring(false);
    }
  };

  const toggleExportTarget = (target: 'cortex_wasm' | 'onnx' | 'safetensors' | 'pytorch') => {
    setExportTargets(prev =>
      prev.includes(target)
        ? prev.filter(t => t !== target) as ('cortex_wasm' | 'onnx' | 'safetensors' | 'pytorch')[]
        : [...prev, target] as ('cortex_wasm' | 'onnx' | 'safetensors' | 'pytorch')[]
    );
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-blue-900 to-slate-900 p-6">
      <div className="max-w-6xl mx-auto">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-white mb-2">
            Configure Your <span className="knirv-gradient-text">KNIRV Neural Model</span>
          </h1>
          <p className="text-slate-300">
            Connected to: <span className="knirv-text-primary font-medium">{connectedApp?.name || 'No app connected'}</span>
          </p>
        </div>

        <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-6">
          <TabsList className="grid w-full grid-cols-5 bg-slate-800 border-slate-700">
            <TabsTrigger value="model-selection" className="data-[state=active]:bg-blue-600">
              <Search className="w-4 h-4 mr-2" />
              Model Selection
            </TabsTrigger>
            <TabsTrigger value="template" disabled={modelSource !== 'template'} className="data-[state=active]:bg-blue-600">
              <Brain className="w-4 h-4 mr-2" />
              Template
            </TabsTrigger>
            <TabsTrigger value="configuration" disabled={!selectedTemplate && !importedModel} className="data-[state=active]:bg-blue-600">
              <Settings className="w-4 h-4 mr-2" />
              Configuration
            </TabsTrigger>
            <TabsTrigger value="training" disabled={!selectedTemplate && !importedModel} className="data-[state=active]:bg-blue-600">
              <Database className="w-4 h-4 mr-2" />
              Training Data
            </TabsTrigger>
            <TabsTrigger value="deployment" disabled={!selectedTemplate && !importedModel} className="data-[state=active]:bg-blue-600">
              <Target className="w-4 h-4 mr-2" />
              Deployment
            </TabsTrigger>
          </TabsList>

          <TabsContent value="model-selection" className="space-y-6">
            <Card className="knirv-card-gradient">
              <CardHeader>
                <CardTitle className="text-white">Choose Your Model Source</CardTitle>
                <CardDescription className="text-white/70">
                  Select how you want to create your KNIRV neural model
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  {/* Template Option */}
                  <Card
                    className={`cursor-pointer transition-all border-2 ${
                      modelSource === 'template'
                        ? 'border-blue-500 bg-blue-900/20'
                        : 'border-slate-600 bg-slate-700 hover:border-slate-500'
                    }`}
                    onClick={() => {
                      setModelSource('template');
                      resetImportSection();
                      setActiveTab('template');
                    }}
                  >
                    <CardHeader className="text-center">
                      <Brain className="h-12 w-12 mx-auto knirv-text-primary mb-2" />
                      <CardTitle className="text-white">Pre-built Templates</CardTitle>
                      <CardDescription className="text-slate-300">
                        Choose from optimized SLM templates
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <ul className="text-sm text-slate-400 space-y-1">
                        <li>• Nano Transformer (1M params)</li>
                        <li>• Micro CNN (500K params)</li>
                        <li>• Small Hybrid (2M params)</li>
                        <li>• Custom architectures</li>
                      </ul>
                    </CardContent>
                  </Card>

                  {/* GitHub Import Option */}
                  <Card
                    className={`cursor-pointer transition-all border-2 ${
                      modelSource === 'github'
                        ? 'border-blue-500 bg-blue-900/20'
                        : 'border-slate-600 bg-slate-700 hover:border-slate-500'
                    }`}
                    onClick={() => {
                      setModelSource('github');
                      if (!connectedApp && onConnectGitHub) {
                        onConnectGitHub();
                      } else {
                        setShowImportSection(true);
                      }
                    }}
                  >
                    <CardHeader className="text-center">
                      <Github className="h-12 w-12 mx-auto knirv-text-primary mb-2" />
                      <CardTitle className="text-white">GitHub Repository</CardTitle>
                      <CardDescription className="text-slate-300">
                        Import open-source models from GitHub
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <ul className="text-sm text-slate-400 space-y-1">
                        <li>• PyTorch models</li>
                        <li>• TensorFlow models</li>
                        <li>• ONNX format</li>
                        <li>• Custom architectures</li>
                      </ul>
                    </CardContent>
                  </Card>

                  {/* Hugging Face Import Option */}
                  <Card
                    className={`cursor-pointer transition-all border-2 ${
                      modelSource === 'huggingface'
                        ? 'border-blue-500 bg-blue-900/20'
                        : 'border-slate-600 bg-slate-700 hover:border-slate-500'
                    }`}
                    onClick={() => {
                      setModelSource('huggingface');
                      setShowImportSection(true);
                    }}
                  >
                    <CardHeader className="text-center">
                      <div className="h-12 w-12 mx-auto knirv-text-primary mb-2 flex items-center justify-center bg-orange-500 rounded-lg">
                        <span className="text-white font-bold text-lg">🤗</span>
                      </div>
                      <CardTitle className="text-white">Hugging Face</CardTitle>
                      <CardDescription className="text-slate-300">
                        Import models from Hugging Face Hub
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <ul className="text-sm text-slate-400 space-y-1">
                        <li>• Pre-trained models</li>
                        <li>• Fine-tuned variants</li>
                        <li>• Community models</li>
                        <li>• Easy integration</li>
                      </ul>
                    </CardContent>
                  </Card>
                </div>

                {/* Import URL Section */}
                {showImportSection && (modelSource === 'github' || modelSource === 'huggingface') && (
                  <Card className="bg-slate-800/50 border-slate-700">
                    <CardHeader>
                      <CardTitle className="text-white flex items-center">
                        <ExternalLink className="h-5 w-5 mr-2 knirv-text-primary" />
                        Import from {modelSource === 'github' ? 'GitHub' : 'Hugging Face'}
                      </CardTitle>
                      <CardDescription className="text-slate-300">
                        {modelSource === 'github'
                          ? 'Enter the GitHub repository URL containing your model'
                          : 'Enter the Hugging Face model identifier or URL'
                        }
                      </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <div className="space-y-2">
                        <Label htmlFor="import-url" className="text-white">
                          {modelSource === 'github' ? 'Repository URL' : 'Model ID/URL'}
                        </Label>
                        <Input
                          id="import-url"
                          placeholder={
                            modelSource === 'github'
                              ? 'https://github.com/username/model-repo'
                              : 'microsoft/DialoGPT-medium or https://huggingface.co/...'
                          }
                          value={importUrl}
                          onChange={(e) => setImportUrl(e.target.value)}
                          className="bg-slate-700 border-slate-600 text-white"
                        />
                      </div>
                      <Button
                        onClick={handleImportModel}
                        disabled={isImporting || !importUrl.trim()}
                        className="knirv-gradient hover:opacity-90"
                      >
                        {isImporting ? (
                          <>
                            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                            Importing...
                          </>
                        ) : (
                          <>
                            <Download className="h-4 w-4 mr-2" />
                            Import Model
                          </>
                        )}
                      </Button>

                      {importedModel && (
                        <div className="mt-4 p-4 bg-green-900/20 border border-green-500/30 rounded-lg">
                          <div className="flex items-center text-green-400 mb-2">
                            <CheckCircle className="h-4 w-4 mr-2" />
                            Model imported successfully
                          </div>
                          <p className="text-white font-medium">{importedModel.name}</p>
                          <p className="text-slate-300 text-sm">{importedModel.description}</p>
                        </div>
                      )}
                    </CardContent>
                  </Card>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="template" className="space-y-6">
            <Card className="knirv-card-gradient">
              <CardHeader>
                <CardTitle className="text-white">Choose a KNIRV Model Template</CardTitle>
                <CardDescription className="text-slate-300">
                  Select a pre-configured KNIRV neural template or start with a custom architecture
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {DEFAULT_SLM_TEMPLATES.map((template) => (
                    <Card 
                      key={template.id}
                      className={`cursor-pointer transition-all border-2 ${
                        selectedTemplate?.id === template.id 
                          ? 'border-blue-500 bg-blue-900/20' 
                          : 'border-slate-600 bg-slate-700 hover:border-slate-500'
                      }`}
                      onClick={() => handleTemplateSelect(template)}
                    >
                      <CardHeader className="pb-3">
                        <div className="flex items-center justify-between">
                          <CardTitle className="text-white text-lg">{template.name}</CardTitle>
                          <Badge variant="outline" className="knirv-text-primary knirv-border-primary">
                            {template.size}
                          </Badge>
                        </div>
                        <CardDescription className="text-slate-300">
                          {template.description}
                        </CardDescription>
                      </CardHeader>
                      <CardContent className="pt-0">
                        <div className="space-y-2 text-sm">
                          <div className="flex justify-between text-slate-300">
                            <span>Parameters:</span>
                            <span className="text-blue-400">{template.parameters.toLocaleString()}</span>
                          </div>
                          <div className="flex justify-between text-slate-300">
                            <span>Type:</span>
                            <span className="text-blue-400">{template.type}</span>
                          </div>
                          <div className="flex justify-between text-slate-300">
                            <span>Layers:</span>
                            <span className="text-blue-400">{template.architecture.layers}</span>
                          </div>
                        </div>
                        <div className="mt-3">
                          <div className="text-xs text-slate-400 mb-1">Capabilities:</div>
                          <div className="flex flex-wrap gap-1">
                            {template.capabilities.slice(0, 3).map((cap) => (
                              <Badge key={cap} variant="secondary" className="text-xs bg-slate-600 text-slate-200">
                                {cap}
                              </Badge>
                            ))}
                            {template.capabilities.length > 3 && (
                              <Badge variant="secondary" className="text-xs bg-slate-600 text-slate-200">
                                +{template.capabilities.length - 3}
                              </Badge>
                            )}
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="configuration" className="space-y-6">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <Card className="bg-slate-800 border-slate-700">
                <CardHeader>
                  <CardTitle className="text-white">Model Details</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <label className="text-sm font-medium text-slate-300 mb-2 block">Model Name</label>
                    <Input
                      value={modelName}
                      onChange={(e) => setModelName(e.target.value)}
                      placeholder="Enter model name"
                      className="bg-slate-700 border-slate-600 text-white"
                    />
                  </div>
                  <div>
                    <label className="text-sm font-medium text-slate-300 mb-2 block">Description</label>
                    <Textarea
                      value={modelDescription}
                      onChange={(e) => setModelDescription(e.target.value)}
                      placeholder="Describe your model's purpose and capabilities"
                      className="bg-slate-700 border-slate-600 text-white"
                      rows={3}
                    />
                  </div>
                </CardContent>
              </Card>

              <Card className="bg-slate-800 border-slate-700">
                <CardHeader>
                  <CardTitle className="text-white">Training Configuration</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <label className="text-sm font-medium text-slate-300 mb-2 block">
                      Learning Rate: {learningRate[0]}
                    </label>
                    <Slider
                      value={learningRate}
                      onValueChange={setLearningRate}
                      min={0.0001}
                      max={0.01}
                      step={0.0001}
                      className="w-full"
                    />
                  </div>
                  <div>
                    <label className="text-sm font-medium text-slate-300 mb-2 block">
                      Batch Size: {batchSize[0]}
                    </label>
                    <Slider
                      value={batchSize}
                      onValueChange={setBatchSize}
                      min={8}
                      max={128}
                      step={8}
                      className="w-full"
                    />
                  </div>
                  <div>
                    <label className="text-sm font-medium text-slate-300 mb-2 block">
                      Epochs: {epochs[0]}
                    </label>
                    <Slider
                      value={epochs}
                      onValueChange={setEpochs}
                      min={1}
                      max={50}
                      step={1}
                      className="w-full"
                    />
                  </div>
                  <div>
                    <label className="text-sm font-medium text-slate-300 mb-2 block">Optimizer</label>
                    <Select value={optimizer} onValueChange={(value: any) => setOptimizer(value)}>
                      <SelectTrigger className="bg-slate-700 border-slate-600 text-white">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent className="bg-slate-700 border-slate-600">
                        <SelectItem value="adam">Adam</SelectItem>
                        <SelectItem value="adamw">AdamW</SelectItem>
                        <SelectItem value="sgd">SGD</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </CardContent>
              </Card>
            </div>

            <Card className="bg-slate-800 border-slate-700">
              <CardHeader>
                <CardTitle className="text-white">Advanced Options</CardTitle>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="flex items-center justify-between">
                  <div>
                    <h4 className="text-white font-medium">LoRA Adapters</h4>
                    <p className="text-sm text-slate-400">Enable Low-Rank Adaptation for efficient fine-tuning</p>
                  </div>
                  <Switch checked={enableLoRA} onCheckedChange={setEnableLoRA} />
                </div>

                {enableLoRA && (
                  <div className="grid grid-cols-2 gap-4 pl-4 border-l-2 border-blue-500">
                    <div>
                      <label className="text-sm font-medium text-slate-300 mb-2 block">
                        LoRA Rank: {loraRank[0]}
                      </label>
                      <Slider
                        value={loraRank}
                        onValueChange={setLoraRank}
                        min={4}
                        max={64}
                        step={4}
                        className="w-full"
                      />
                    </div>
                    <div>
                      <label className="text-sm font-medium text-slate-300 mb-2 block">
                        LoRA Alpha: {loraAlpha[0]}
                      </label>
                      <Slider
                        value={loraAlpha}
                        onValueChange={setLoraAlpha}
                        min={8}
                        max={128}
                        step={8}
                        className="w-full"
                      />
                    </div>
                  </div>
                )}

                <div className="flex items-center justify-between">
                  <div>
                    <h4 className="text-white font-medium">Quantization</h4>
                    <p className="text-sm text-slate-400">Reduce model size with quantization</p>
                  </div>
                  <Switch checked={enableQuantization} onCheckedChange={setEnableQuantization} />
                </div>

                {enableQuantization && (
                  <div className="pl-4 border-l-2 border-blue-500">
                    <label className="text-sm font-medium text-slate-300 mb-2 block">Quantization Bits</label>
                    <Select value={quantizationBits.toString()} onValueChange={(value) => setQuantizationBits(parseInt(value) as any)}>
                      <SelectTrigger className="bg-slate-700 border-slate-600 text-white w-32">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent className="bg-slate-700 border-slate-600">
                        <SelectItem value="4">4-bit</SelectItem>
                        <SelectItem value="8">8-bit</SelectItem>
                        <SelectItem value="16">16-bit</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="training" className="space-y-6">
            <Card className="bg-slate-800 border-slate-700">
              <CardHeader>
                <CardTitle className="text-white">Training Data</CardTitle>
                <CardDescription className="text-slate-300">
                  Provide training data for your model
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <label className="text-sm font-medium text-slate-300 mb-2 block">Text Data</label>
                  <Textarea
                    value={trainingData}
                    onChange={(e) => setTrainingData(e.target.value)}
                    placeholder="Enter training text data here..."
                    className="bg-slate-700 border-slate-600 text-white min-h-[200px]"
                  />
                </div>

                <div>
                  <label className="text-sm font-medium text-slate-300 mb-2 block">Upload Files</label>
                  <div className="border-2 border-dashed border-slate-600 rounded-lg p-6 text-center">
                    <Upload className="w-8 h-8 text-slate-400 mx-auto mb-2" />
                    <p className="text-slate-400 mb-2">Drop files here or click to upload</p>
                    <input
                      type="file"
                      multiple
                      accept=".txt,.json,.csv"
                      onChange={handleFileUpload}
                      className="hidden"
                      id="file-upload"
                    />
                    <Button asChild variant="outline" className="border-slate-600 text-slate-300">
                      <label htmlFor="file-upload" className="cursor-pointer">
                        Choose Files
                      </label>
                    </Button>
                  </div>
                </div>

                {uploadedFiles.length > 0 && (
                  <div>
                    <label className="text-sm font-medium text-slate-300 mb-2 block">Uploaded Files</label>
                    <div className="space-y-2">
                      {uploadedFiles.map((file, index) => (
                        <div key={index} className="flex items-center justify-between bg-slate-700 p-3 rounded">
                          <div className="flex items-center">
                            <FileText className="w-4 h-4 text-blue-400 mr-2" />
                            <span className="text-white">{file.name}</span>
                            <span className="text-slate-400 ml-2">({(file.size / 1024).toFixed(1)} KB)</span>
                          </div>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => removeFile(index)}
                            className="text-red-400 hover:text-red-300"
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="deployment" className="space-y-6">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <Card className="bg-slate-800 border-slate-700">
                <CardHeader>
                  <CardTitle className="text-white">Export Formats</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  {[
                    { id: 'cortex_wasm', name: 'Cortex WASM', desc: 'Optimized for KNIRV ecosystem' },
                    { id: 'onnx', name: 'ONNX', desc: 'Cross-platform inference' },
                    { id: 'safetensors', name: 'SafeTensors', desc: 'Safe model serialization' },
                    { id: 'pytorch', name: 'PyTorch', desc: 'Native PyTorch format' }
                  ].map((format) => (
                    <div key={format.id} className="flex items-center space-x-3">
                      <input
                        type="checkbox"
                        id={format.id}
                        checked={exportTargets.includes(format.id as 'cortex_wasm' | 'onnx' | 'safetensors' | 'pytorch')}
                        onChange={() => toggleExportTarget(format.id as 'cortex_wasm' | 'onnx' | 'safetensors' | 'pytorch')}
                        className="rounded border-slate-600"
                      />
                      <div>
                        <label htmlFor={format.id} className="text-white font-medium cursor-pointer">
                          {format.name}
                        </label>
                        <p className="text-sm text-slate-400">{format.desc}</p>
                      </div>
                    </div>
                  ))}
                </CardContent>
              </Card>

              <Card className="bg-slate-800 border-slate-700">
                <CardHeader>
                  <CardTitle className="text-white">Deployment Targets</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="text-white font-medium">KNIRV Controller</h4>
                      <p className="text-sm text-slate-400">Deploy to TypeScript runtime</p>
                    </div>
                    <Switch 
                      checked={deploymentTargets.knirvcontroller} 
                      onCheckedChange={(checked) => setDeploymentTargets(prev => ({...prev, knirvcontroller: checked}))}
                    />
                  </div>
                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="text-white font-medium">KNIRV Engine</h4>
                      <p className="text-sm text-slate-400">Deploy to Go runtime</p>
                    </div>
                    <Switch 
                      checked={deploymentTargets.knirvengine} 
                      onCheckedChange={(checked) => setDeploymentTargets(prev => ({...prev, knirvengine: checked}))}
                    />
                  </div>
                </CardContent>
              </Card>
            </div>

            <div className="flex justify-end space-x-4">
              <Button
                variant="outline"
                onClick={() => setActiveTab('training')}
                className="border-slate-600 text-slate-300"
              >
                Back
              </Button>
              <Button
                onClick={handleConfigure}
                disabled={isConfiguring}
                className="bg-gradient-to-r from-blue-500 to-cyan-500 hover:from-blue-600 hover:to-cyan-600 text-white"
              >
                {isConfiguring ? (
                  <>
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                    Configuring...
                  </>
                ) : (
                  <>
                    <Cpu className="w-4 h-4 mr-2" />
                    Configure Model
                  </>
                )}
              </Button>
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
};

export default ModelConfig;
