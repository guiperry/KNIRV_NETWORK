'use client';

import React, { useState, useEffect } from 'react';
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { 
  Brain, 
  Zap, 
  CheckCircle, 
  XCircle, 
  Loader2, 
  Download, 
  Share, 
  Server, 
  Code, 
  TestTube,
  Clock,
  Cpu,
  Database,
  Target,
  TrendingUp,
  Activity
} from "lucide-react";
import { useToast } from "@/hooks/use-toast";
import { 
  CortexModelCompiler, 
  ModelConfiguration, 
  ModelCompilationRequest,
  ModelCompilationResponse,
  TrainingProgress,
  cortexModelCompiler 
} from "@/lib/cortex-compiler/CortexModelCompiler";

interface ModelDeployerProps {
  modelConfig: ModelConfiguration;
  onDeployed: () => void;
  onConnectToTargets?: () => void;
  onCompilationSuccess?: () => void;
  onReset?: () => void;
}

const ModelDeployer = ({ modelConfig, onDeployed, onConnectToTargets, onCompilationSuccess, onReset }: ModelDeployerProps) => {
  const { toast } = useToast();
  
  // Debug logging
  console.log('ModelDeployer rendering with modelConfig:', modelConfig);
  
  // Validate modelConfig structure
  if (!modelConfig || !modelConfig.template) {
    console.error('Invalid modelConfig:', modelConfig);
    return (
      <div className="min-h-screen bg-gradient-to-br from-slate-900 via-blue-900 to-slate-900 p-6">
        <div className="max-w-6xl mx-auto text-center text-white">
          <h1 className="text-3xl font-bold mb-4">Configuration Error</h1>
          <p>Invalid model configuration. Please go back and configure your model again.</p>
        </div>
      </div>
    );
  }
  
  // Compilation state
  const [isCompiling, setIsCompiling] = useState(false);
  const [compilationProgress, setCompilationProgress] = useState<TrainingProgress | null>(null);
  const [compilationResult, setCompilationResult] = useState<ModelCompilationResponse | null>(null);
  const [activeTab, setActiveTab] = useState('compilation');

  // Training metrics
  const [trainingMetrics, setTrainingMetrics] = useState({
    currentLoss: 0,
    currentAccuracy: 0,
    learningRate: modelConfig.training_config?.learning_rate || 0.001,
    epochsCompleted: 0,
    totalEpochs: modelConfig.training_config?.epochs || 10
  });

  const handleStartCompilation = async () => {
    setIsCompiling(true);
    setCompilationResult(null);
    setActiveTab('compilation');

    const request: ModelCompilationRequest = {
      model_config: modelConfig,
      training_data: {
        text_data: ["Sample training data"], // In real implementation, this would come from the form
      },
      deployment_targets: {
        knirvcontroller: true,
        knirvserver: false,
        cloud_hosting: {
          provider: 'vercel'
        }
      }
    };

    try {
      const result = await cortexModelCompiler.compileModel(
        request,
        (progress: TrainingProgress) => {
          setCompilationProgress(progress);
          setTrainingMetrics({
            currentLoss: progress.loss,
            currentAccuracy: progress.accuracy || 0,
            learningRate: progress.learning_rate,
            epochsCompleted: progress.epoch,
            totalEpochs: progress.total_epochs
          });
        }
      );

      setCompilationResult(result);
      
      if (result.success) {
        toast({
          title: "Compilation Successful",
          description: `Model "${modelConfig.name}" compiled successfully`,
        });
        setActiveTab('results');
      } else {
        toast({
          title: "Compilation Failed",
          description: result.message,
          variant: "destructive",
        });
      }
    } catch (error) {
      toast({
        title: "Compilation Error",
        description: `Error: ${error}`,
        variant: "destructive",
      });
    } finally {
      setIsCompiling(false);
    }
  };

  const handleDownload = () => {
    if (compilationResult?.cortex_wasm) {
      const blob = new Blob([new Uint8Array(compilationResult.cortex_wasm)], { type: 'application/wasm' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${modelConfig.name.toLowerCase().replace(/\s+/g, '-')}-cortex.wasm`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      toast({
        title: "Download Started",
        description: "Your cortex.wasm file is downloading",
      });
    }
  };

  const handleDeploy = () => {
    console.log('handleDeploy called, onConnectToTargets function:', onConnectToTargets);

    // Call the onConnectToTargets callback to navigate to the dashboard
    if (typeof onConnectToTargets === 'function') {
      onConnectToTargets();
    } else {
      console.error('onConnectToTargets is not a function:', onConnectToTargets);
      // Fallback to the old behavior if callback is not provided
      toast({
        title: "Navigation Error",
        description: "Unable to navigate to deployment dashboard",
        variant: "destructive",
      });
    }
  };

  const getStageIcon = (stage: string) => {
    switch (stage) {
      case 'preprocessing': return <Database className="w-4 h-4" />;
      case 'training': return <Brain className="w-4 h-4" />;
      case 'validation': return <TestTube className="w-4 h-4" />;
      case 'compilation': return <Cpu className="w-4 h-4" />;
      case 'deployment': return <Target className="w-4 h-4" />;
      default: return <Activity className="w-4 h-4" />;
    }
  };

  const getStageColor = (stage: string) => {
    switch (stage) {
      case 'preprocessing': return 'text-yellow-400';
      case 'training': return 'text-blue-400';
      case 'validation': return 'text-purple-400';
      case 'compilation': return 'text-green-400';
      case 'deployment': return 'text-cyan-400';
      default: return 'text-slate-400';
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-blue-900 to-slate-900 p-6">
      <div className="max-w-6xl mx-auto">
        <div className="mb-8">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-white mb-2">
                Compile & Pre-Train Model
              </h1>
              <p className="text-slate-300">
                Model: <span className="text-blue-400 font-medium">{modelConfig.name}</span>
              </p>
            </div>
            {compilationResult?.success && (
              <div className="flex items-center space-x-4">
                <Button
                  onClick={handleDownload}
                  className="bg-gradient-to-r from-green-500 to-emerald-500 hover:from-green-600 hover:to-emerald-600 text-white"
                >
                  <Download className="w-4 h-4 mr-2" />
                  Download cortex.wasm
                </Button>
                <Button
                  onClick={handleDeploy}
                  className="bg-gradient-to-r from-blue-500 to-cyan-500 hover:from-blue-600 hover:to-cyan-600 text-white"
                >
                  <Server className="w-4 h-4 mr-2" />
                  Connect to Targets
                </Button>
                {onReset && (
                  <Button
                    onClick={onReset}
                    variant="outline"
                    className="border-orange-600/50 text-orange-300 hover:bg-orange-900/20 hover:border-orange-500"
                    title="Clear all progress and return to home"
                  >
                    Reset
                  </Button>
                )}
              </div>
            )}
            {!compilationResult?.success && (
              <div className="flex items-center space-x-4">
                <Button
                  onClick={handleStartCompilation}
                  disabled={isCompiling}
                  size="lg"
                  className="bg-gradient-to-r from-blue-500 to-cyan-500 hover:from-blue-600 hover:to-cyan-600 text-white px-8"
                >
                  {isCompiling ? (
                    <>
                      <Loader2 className="w-5 h-5 mr-2 animate-spin" />
                      Compiling Model...
                    </>
                  ) : (
                    <>
                      <Zap className="w-5 h-5 mr-2" />
                      Start Compilation
                    </>
                  )}
                </Button>
                {onReset && (
                  <Button
                    onClick={onReset}
                    variant="outline"
                    className="border-orange-600/50 text-orange-300 hover:bg-orange-900/20 hover:border-orange-500"
                    title="Clear all progress and return to home"
                  >
                    Reset
                  </Button>
                )}
              </div>
            )}
          </div>
        </div>

        <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-6">
          <TabsList className="grid w-full grid-cols-3 bg-slate-800 border-slate-700">
            <TabsTrigger value="compilation" className="data-[state=active]:bg-blue-600">
              <Cpu className="w-4 h-4 mr-2" />
              Compilation
            </TabsTrigger>
            <TabsTrigger value="monitoring" disabled={!isCompiling && !compilationResult} className="data-[state=active]:bg-blue-600">
              <Brain className="w-4 h-4 mr-2" />
              Training
            </TabsTrigger>
            <TabsTrigger value="results" disabled={!compilationResult} className="data-[state=active]:bg-blue-600">
              <Target className="w-4 h-4 mr-2" />
              Results
            </TabsTrigger>
          </TabsList>

          <TabsContent value="compilation" className="space-y-6">
            <Card className="bg-slate-800 border-slate-700">
              <CardHeader>
                <CardTitle className="text-white">Model Configuration Summary</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                  <div>
                    <h4 className="text-white font-medium mb-2">Template</h4>
                    <p className="text-slate-300">{modelConfig.template?.name || 'Unknown'}</p>
                    <Badge variant="outline" className="mt-1 text-blue-400 border-blue-400">
                      {modelConfig.template?.size || 'Unknown'}
                    </Badge>
                  </div>
                  <div>
                    <h4 className="text-white font-medium mb-2">Parameters</h4>
                    <p className="text-slate-300">{modelConfig.template?.parameters?.toLocaleString() || '0'}</p>
                    <p className="text-sm text-slate-400">{modelConfig.template?.type || 'Unknown'} architecture</p>
                  </div>
                  <div>
                    <h4 className="text-white font-medium mb-2">Training Config</h4>
                    <p className="text-slate-300">
                      {modelConfig.training_config?.epochs || 0} epochs,
                      LR: {modelConfig.training_config?.learning_rate || 0}
                    </p>
                    <p className="text-sm text-slate-400">
                      Batch size: {modelConfig.training_config?.batch_size || 0}
                    </p>
                  </div>
                </div>

                {modelConfig.lora_config?.enabled && (
                  <div className="mt-4 p-3 bg-blue-900/20 border border-blue-500/30 rounded">
                    <h4 className="text-blue-400 font-medium mb-1">LoRA Configuration</h4>
                    <p className="text-sm text-slate-300">
                      Rank: {modelConfig.lora_config.rank}, Alpha: {modelConfig.lora_config.alpha}
                    </p>
                  </div>
                )}

                {modelConfig.quantization?.enabled && (
                  <div className="mt-4 p-3 bg-green-900/20 border border-green-500/30 rounded">
                    <h4 className="text-green-400 font-medium mb-1">Quantization</h4>
                    <p className="text-sm text-slate-300">
                      {modelConfig.quantization.bits}-bit {modelConfig.quantization.method}
                    </p>
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="monitoring" className="space-y-6">
            {compilationProgress && (
              <>
                <Card className="bg-slate-800 border-slate-700">
                  <CardHeader>
                    <CardTitle className="text-white flex items-center">
                      <span className={getStageColor(compilationProgress.stage)}>
                        {getStageIcon(compilationProgress.stage)}
                      </span>
                      <span className="ml-2">
                        {compilationProgress.stage.charAt(0).toUpperCase() + compilationProgress.stage.slice(1)}
                      </span>
                    </CardTitle>
                    <CardDescription className="text-slate-300">
                      Epoch {compilationProgress.epoch} of {compilationProgress.total_epochs}
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-4">
                      <div>
                        <div className="flex justify-between text-sm mb-2">
                          <span className="text-slate-300">Training Progress</span>
                          <span className="text-blue-400">
                            {Math.round((compilationProgress.epoch / compilationProgress.total_epochs) * 100)}%
                          </span>
                        </div>
                        <Progress 
                          value={(compilationProgress.epoch / compilationProgress.total_epochs) * 100} 
                          className="h-2"
                        />
                      </div>

                      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                        <div className="bg-slate-700 p-3 rounded">
                          <div className="text-sm text-slate-400">Loss</div>
                          <div className="text-lg font-bold text-white">
                            {compilationProgress.loss.toFixed(4)}
                          </div>
                        </div>
                        {compilationProgress.accuracy && (
                          <div className="bg-slate-700 p-3 rounded">
                            <div className="text-sm text-slate-400">Accuracy</div>
                            <div className="text-lg font-bold text-green-400">
                              {(compilationProgress.accuracy * 100).toFixed(1)}%
                            </div>
                          </div>
                        )}
                        <div className="bg-slate-700 p-3 rounded">
                          <div className="text-sm text-slate-400">Learning Rate</div>
                          <div className="text-lg font-bold text-blue-400">
                            {compilationProgress.learning_rate.toFixed(6)}
                          </div>
                        </div>
                        <div className="bg-slate-700 p-3 rounded">
                          <div className="text-sm text-slate-400">ETA</div>
                          <div className="text-lg font-bold text-yellow-400">
                            {Math.round(compilationProgress.estimated_time_remaining_ms / 1000)}s
                          </div>
                        </div>
                      </div>
                    </div>
                  </CardContent>
                </Card>

                <Card className="bg-slate-800 border-slate-700">
                  <CardHeader>
                    <CardTitle className="text-white flex items-center">
                      <TrendingUp className="w-5 h-5 mr-2 text-green-400" />
                      Training Metrics
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-3">
                      <div className="flex justify-between items-center">
                        <span className="text-slate-300">Current Loss</span>
                        <span className="text-white font-mono">{trainingMetrics.currentLoss.toFixed(4)}</span>
                      </div>
                      <div className="flex justify-between items-center">
                        <span className="text-slate-300">Current Accuracy</span>
                        <span className="text-green-400 font-mono">
                          {(trainingMetrics.currentAccuracy * 100).toFixed(1)}%
                        </span>
                      </div>
                      <div className="flex justify-between items-center">
                        <span className="text-slate-300">Epochs Completed</span>
                        <span className="text-blue-400 font-mono">
                          {trainingMetrics.epochsCompleted} / {trainingMetrics.totalEpochs}
                        </span>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </>
            )}
          </TabsContent>

          <TabsContent value="results" className="space-y-6">
            {compilationResult && (
              <>
                <Card className="bg-slate-800 border-slate-700">
                  <CardHeader>
                    <CardTitle className="text-white flex items-center">
                      {compilationResult.success ? (
                        <CheckCircle className="w-5 h-5 mr-2 text-green-400" />
                      ) : (
                        <XCircle className="w-5 h-5 mr-2 text-red-400" />
                      )}
                      Compilation {compilationResult.success ? 'Successful' : 'Failed'}
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-slate-300 mb-4">{compilationResult.message}</p>
                    
                    {compilationResult.success && compilationResult.model_metrics && (
                      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                        <div className="bg-slate-700 p-3 rounded">
                          <div className="text-sm text-slate-400">Model Size</div>
                          <div className="text-lg font-bold text-white">
                            {compilationResult.model_metrics.size_mb.toFixed(1)} MB
                          </div>
                        </div>
                        <div className="bg-slate-700 p-3 rounded">
                          <div className="text-sm text-slate-400">Parameters</div>
                          <div className="text-lg font-bold text-blue-400">
                            {compilationResult.model_metrics.parameters.toLocaleString()}
                          </div>
                        </div>
                        <div className="bg-slate-700 p-3 rounded">
                          <div className="text-sm text-slate-400">Inference Speed</div>
                          <div className="text-lg font-bold text-green-400">
                            {compilationResult.model_metrics.inference_speed_ms}ms
                          </div>
                        </div>
                        <div className="bg-slate-700 p-3 rounded">
                          <div className="text-sm text-slate-400">Memory Usage</div>
                          <div className="text-lg font-bold text-yellow-400">
                            {compilationResult.model_metrics.memory_usage_mb} MB
                          </div>
                        </div>
                      </div>
                    )}
                  </CardContent>
                </Card>

                {compilationResult.success && (
                  <Card className="bg-slate-800 border-slate-700">
                    <CardHeader>
                      <CardTitle className="text-white">Next Steps</CardTitle>
                      <CardDescription className="text-slate-300">
                        Your model compilation is complete! Here's what happens next:
                      </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <div className="bg-blue-900/20 border border-blue-500/30 rounded-lg p-4">
                        <h4 className="text-blue-400 font-medium mb-2 flex items-center">
                          <Target className="w-4 h-4 mr-2" />
                          Model Deployment Dashboard
                        </h4>
                        <p className="text-slate-300 text-sm mb-3">
                          When you click "Connect to Targets", you'll be taken to the Model Deployment Dashboard where you can:
                        </p>
                        <ul className="text-sm text-slate-300 space-y-1 mb-4">
                          <li>• View all available deployment targets</li>
                          <li>• Manage your model deployments</li>
                          <li>• Monitor deployment status and health</li>
                          <li>• Configure target-specific settings</li>
                        </ul>
                        <div className="bg-slate-800/50 rounded p-3 border-l-4 border-blue-400">
                          <p className="text-blue-300 text-sm">
                            <strong>💡 Tip:</strong> Look for the "Connect Your Controller" card in the dashboard to set up your KNIRVCONTROLLER mobile app connection.
                          </p>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                )}
              </>
            )}
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
};

export default ModelDeployer;
