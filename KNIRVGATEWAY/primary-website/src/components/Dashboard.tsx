'use client';

import React, { useState } from 'react';
import SettingsModal from "./SettingsModal";
import QRConnectionModal from "./QRConnectionModal";
import { MessageCircle, TrendingUp, Activity, Users, Bot, History, Monitor, Rocket, Clock, CheckCircle, XCircle, QrCode, Smartphone, Cpu, Download } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";

// props required for status/settings, now provided from parent Index
interface DashboardProps {
  connectedApp: {url: string, name: string, type: string} | null;
  isActive: boolean;
  setIsActive: (active: boolean) => void;
  settingsModalOpen: boolean;
  setSettingsModalOpen: (open: boolean) => void;
  onConnectApp?: (appData: {url: string, name: string, type: string}) => void;
  modelConfig?: any; // Add model config to show cortex.wasm info
  onReset?: () => void;
}

const Dashboard = ({
  connectedApp,
  isActive,
  setIsActive,
  settingsModalOpen,
  setSettingsModalOpen,
  onConnectApp,
  modelConfig,
  onReset,
}: DashboardProps) => {
  const [qrModalOpen, setQrModalOpen] = useState(false);
  const [controllerConnected, setControllerConnected] = useState(false);

  const handleAppConnected = (appData: {url: string, name: string, type: string}) => {
    if (onConnectApp) {
      onConnectApp(appData);
    }
    setQrModalOpen(false);
  };
  const stats = [
    { label: 'Total Conversations', value: '1,247', change: '+12%', icon: MessageCircle },
    { label: 'Active Users', value: '342', change: '+8%', icon: Users },
    { label: 'Response Rate', value: '97%', change: '+3%', icon: TrendingUp },
    { label: 'Uptime', value: '99.9%', change: '0%', icon: Activity },
  ];

  const recentActivity = [
    { time: '2m ago', event: 'User asked about pricing', response: 'Provided pricing information' },
    { time: '5m ago', event: 'Integration test completed', response: 'All systems operational' },
    { time: '12m ago', event: 'New user onboarded', response: 'Guided through features' },
    { time: '18m ago', event: 'Feature request submitted', response: 'Logged for review' },
  ];

  const deployHistory = [
    {
      id: '1',
      version: 'v1.2.3',
      timestamp: '2024-01-15 14:30:00',
      status: 'success',
      environment: 'production',
      duration: '2m 34s',
      deployedBy: 'John Doe'
    },
    {
      id: '2',
      version: 'v1.2.2',
      timestamp: '2024-01-14 09:15:00',
      status: 'success',
      environment: 'staging',
      duration: '1m 45s',
      deployedBy: 'Jane Smith'
    },
    {
      id: '3',
      version: 'v1.2.1',
      timestamp: '2024-01-13 16:20:00',
      status: 'failed',
      environment: 'production',
      duration: '45s',
      deployedBy: 'Bob Johnson'
    },
    {
      id: '4',
      version: 'v1.2.0',
      timestamp: '2024-01-12 11:00:00',
      status: 'success',
      environment: 'production',
      duration: '3m 12s',
      deployedBy: 'Alice Brown'
    },
  ];

  const monitoringStats = [
    { label: 'CPU Usage', value: '45%', status: 'good', trend: 'stable' },
    { label: 'Memory Usage', value: '67%', status: 'warning', trend: 'increasing' },
    { label: 'Response Time', value: '0.8s', status: 'good', trend: 'decreasing' },
    { label: 'Error Rate', value: '0.2%', status: 'good', trend: 'stable' },
    { label: 'Throughput', value: '1.2k/min', status: 'good', trend: 'increasing' },
    { label: 'Active Connections', value: '342', status: 'good', trend: 'stable' },
  ];

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
      {/* QR Connection Modal */}
      <QRConnectionModal
        isOpen={qrModalOpen}
        onClose={() => setQrModalOpen(false)}
        onConnected={(appData) => {
          handleAppConnected(appData);
          setControllerConnected(true);
        }}
      />

      {connectedApp ? (
        <>
          <div className="mb-8 flex items-center justify-between">
            <div>
              <h2 className="text-4xl font-bold text-white mb-4">
                <span className="knirv-gradient-text">KNIRV</span> Model Deployment Dashboard
              </h2>
              <p className="text-xl text-white/70">
                Manage your {connectedApp.name} <span className="knirv-text-primary">Neural Intelligence Model</span> deployments
              </p>
            </div>
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

          {/* Cortex.WASM Configuration Display */}
          {modelConfig && (
            <div className="mb-8">
              <Card className="bg-slate-800/50 border-slate-700">
                <CardContent className="p-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-3">
                      <div className="w-10 h-10 bg-green-500/20 rounded-lg flex items-center justify-center">
                        <Cpu className="w-5 h-5 text-green-400" />
                      </div>
                      <div>
                        <h3 className="text-white font-medium">cortex.wasm Ready</h3>
                        <p className="text-slate-300 text-sm">{modelConfig.name} • {modelConfig.template?.parameters?.toLocaleString() || '0'} parameters</p>
                      </div>
                    </div>
                    <Badge variant="outline" className="border-green-400/50 text-green-400">
                      Ready for Deployment
                    </Badge>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}

          {/* Deployment Action Cards */}
          <div className="grid md:grid-cols-4 gap-6 mb-8">
            <Card className="bg-gradient-to-br from-blue-900/20 to-cyan-900/20 border-blue-500/30 backdrop-blur-lg cursor-pointer hover:from-blue-800/30 hover:to-cyan-800/30 transition-all"
                  onClick={() => setQrModalOpen(true)}>
              <CardContent className="p-6 text-center">
                <div className="w-12 h-12 bg-blue-500/20 rounded-full flex items-center justify-center mx-auto mb-4">
                  <QrCode className="w-6 h-6 text-blue-400" />
                </div>
                <h3 className="text-lg font-bold text-white mb-2">Connect Your Controller</h3>
                <p className="text-slate-300 text-sm">
                  Link your KNIRVCONTROLLER mobile app to manage models on-the-go
                </p>
              </CardContent>
            </Card>

            <Card className="bg-gradient-to-br from-purple-900/20 to-indigo-900/20 border-purple-500/30 backdrop-blur-lg cursor-pointer hover:from-purple-800/30 hover:to-indigo-800/30 transition-all"
                  onClick={() => window.open('https://knirv.com/nexus-portal', '_blank')}>
              <CardContent className="p-6 text-center">
                <div className="w-12 h-12 bg-purple-500/20 rounded-full flex items-center justify-center mx-auto mb-4">
                  <Rocket className="w-6 h-6 text-purple-400" />
                </div>
                <h3 className="text-lg font-bold text-white mb-2">Deploy to DVE</h3>
                <p className="text-slate-300 text-sm">
                  Deploy to KNIRV-NEXUS distributed validation environment
                </p>
              </CardContent>
            </Card>

            <Card className="bg-gradient-to-br from-orange-900/20 to-red-900/20 border-orange-500/30 backdrop-blur-lg cursor-pointer hover:from-orange-800/30 hover:to-red-800/30 transition-all"
                  onClick={() => window.open('https://dash.cloudflare.com', '_blank')}>
              <CardContent className="p-6 text-center">
                <div className="w-12 h-12 bg-orange-500/20 rounded-full flex items-center justify-center mx-auto mb-4">
                  <div className="w-6 h-6 bg-orange-400 rounded-sm"></div>
                </div>
                <h3 className="text-lg font-bold text-white mb-2">Deploy to Cloudflare</h3>
                <p className="text-slate-300 text-sm">
                  Deploy to Cloudflare Workers for global edge deployment
                </p>
              </CardContent>
            </Card>

            <Card className="bg-gradient-to-br from-green-900/20 to-emerald-900/20 border-green-500/30 backdrop-blur-lg cursor-pointer hover:from-green-800/30 hover:to-emerald-800/30 transition-all">
              <CardContent className="p-6 text-center">
                <div className="w-12 h-12 bg-green-500/20 rounded-full flex items-center justify-center mx-auto mb-4">
                  <Download className="w-6 h-6 text-green-400" />
                </div>
                <h3 className="text-lg font-bold text-white mb-2">Download Your Cortex</h3>
                <p className="text-slate-300 text-sm">
                  Download cortex.wasm file for local deployment
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Bottom Section - Hidden until controller connected */}
          {controllerConnected && (
            <>
              {/* Stats Grid */}
              <div className="grid md:grid-cols-4 gap-6 mb-8">
                {stats.map((stat, index) => (
                  <Card key={index} className="bg-white/5 border-white/10 backdrop-blur-lg">
                    <CardContent className="p-6">
                      <div className="flex items-center justify-between mb-4">
                        <stat.icon className="h-8 w-8 text-purple-400" />
                        <Badge variant="outline" className="border-green-400/50 text-green-400">
                          {stat.change}
                        </Badge>
                      </div>
                      <div>
                        <p className="text-2xl font-bold text-white">{stat.value}</p>
                        <p className="text-white/70 text-sm">{stat.label}</p>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>

              <Tabs defaultValue="overview" className="w-full">
                <TabsList className="grid w-full grid-cols-4 bg-white/5 border border-white/10">
                  <TabsTrigger value="overview" className="data-[state=active]:bg-purple-500/20">
                    Overview
                  </TabsTrigger>
                  <TabsTrigger value="deployments" className="data-[state=active]:bg-purple-500/20">
                    Deployments
                  </TabsTrigger>
                  <TabsTrigger value="monitoring" className="data-[state=active]:bg-purple-500/20">
                    <Monitor className="h-4 w-4 mr-2" />
                    Monitoring
                  </TabsTrigger>
                  <TabsTrigger value="settings" className="data-[state=active]:bg-purple-500/20">
                    Settings
                  </TabsTrigger>
                </TabsList>

                <TabsContent value="overview" className="space-y-6">
                  <div className="grid lg:grid-cols-2 gap-6">
                    <Card className="knirv-card-gradient backdrop-blur-lg">
                      <CardHeader>
                        <CardTitle className="text-white flex items-center">
                          <Bot className="h-5 w-5 mr-2 knirv-text-primary" />
                          Deployment Status
                        </CardTitle>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <div className="flex items-center justify-between">
                          <span className="text-white/70">DVE Deployment</span>
                          <span className="text-green-400">Active</span>
                        </div>
                        <Progress value={100} className="w-full" />

                        <div className="space-y-3">
                          <div className="flex justify-between">
                            <span className="text-white/70">Cloudflare Status</span>
                            <span className="text-blue-400">Ready</span>
                          </div>
                          <div className="flex justify-between">
                            <span className="text-white/70">Local cortex.wasm</span>
                            <span className="text-white">Available</span>
                          </div>
                        </div>
                      </CardContent>
                    </Card>

                    <Card className="bg-white/5 border-white/10 backdrop-blur-lg">
                      <CardHeader>
                        <CardTitle className="text-white flex items-center">
                          <Activity className="h-5 w-5 mr-2 text-purple-400" />
                          Recent Activity
                        </CardTitle>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        {recentActivity.map((activity, index) => (
                          <div key={index} className="border-b border-white/10 pb-3 last:border-b-0">
                            <div className="flex justify-between items-start mb-1">
                              <span className="text-white text-sm">{activity.event}</span>
                              <span className="text-white/50 text-xs">{activity.time}</span>
                            </div>
                            <p className="text-white/70 text-xs">{activity.response}</p>
                          </div>
                        ))}
                      </CardContent>
                    </Card>
                  </div>
                </TabsContent>

                <TabsContent value="deployments" className="space-y-6">
                  <Card className="bg-white/5 border-white/10 backdrop-blur-lg">
                    <CardHeader>
                      <CardTitle className="text-white flex items-center">
                        <History className="h-5 w-5 mr-2 text-purple-400" />
                        Deployment History
                      </CardTitle>
                      <CardDescription className="text-white/70">
                        Track all model deployments and their status
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-4">
                        {deployHistory.map((deployment) => (
                          <div key={deployment.id} className="bg-white/10 rounded-lg p-4 flex items-center justify-between">
                            <div className="flex items-center space-x-4">
                              <div className={`w-3 h-3 rounded-full ${
                                deployment.status === 'success' ? 'bg-green-400' : 'bg-red-400'
                              }`} />
                              <div>
                                <div className="flex items-center space-x-2">
                                  <h4 className="text-white font-medium">{deployment.version}</h4>
                                  <Badge
                                    variant="outline"
                                    className={`text-xs ${
                                      deployment.environment === 'production'
                                        ? 'border-purple-400/50 text-purple-400'
                                        : 'border-blue-400/50 text-blue-400'
                                    }`}
                                  >
                                    {deployment.environment}
                                  </Badge>
                                </div>
                                <p className="text-white/60 text-sm">
                                  Deployed by {deployment.deployedBy} • {deployment.timestamp}
                                </p>
                              </div>
                            </div>
                            <div className="flex items-center space-x-4">
                              <div className="text-right">
                                <p className="text-white/70 text-sm">Duration</p>
                                <p className="text-white text-sm">{deployment.duration}</p>
                              </div>
                              {deployment.status === 'success' ? (
                                <CheckCircle className="h-5 w-5 text-green-400" />
                              ) : (
                                <XCircle className="h-5 w-5 text-red-400" />
                              )}
                            </div>
                          </div>
                        ))}
                      </div>
                    </CardContent>
                  </Card>
                </TabsContent>

                <TabsContent value="monitoring" className="space-y-6">
                  <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
                    {monitoringStats.map((stat, index) => (
                      <Card key={index} className="bg-white/5 border-white/10 backdrop-blur-lg">
                        <CardContent className="p-6">
                          <div className="flex items-center justify-between mb-4">
                            <Monitor className="h-6 w-6 text-purple-400" />
                            <Badge
                              variant="outline"
                              className={`text-xs ${
                                stat.status === 'good' ? 'border-green-400/50 text-green-400' :
                                stat.status === 'warning' ? 'border-yellow-400/50 text-yellow-400' :
                                'border-red-400/50 text-red-400'
                              }`}
                            >
                              {stat.status}
                            </Badge>
                          </div>
                          <div>
                            <p className="text-2xl font-bold text-white">{stat.value}</p>
                            <p className="text-white/70 text-sm">{stat.label}</p>
                            <p className={`text-xs mt-1 ${
                              stat.trend === 'increasing' ? 'text-green-400' :
                              stat.trend === 'decreasing' ? 'text-blue-400' :
                              'text-gray-400'
                            }`}>
                              {stat.trend === 'increasing' ? '↗ Increasing' :
                               stat.trend === 'decreasing' ? '↘ Decreasing' :
                               '→ Stable'}
                            </p>
                          </div>
                        </CardContent>
                      </Card>
                    ))}
                  </div>
                </TabsContent>

                <TabsContent value="settings" className="space-y-6">
                  <Card className="bg-white/5 border-white/10 backdrop-blur-lg">
                    <CardHeader>
                      <CardTitle className="text-white">Deployment Settings</CardTitle>
                      <CardDescription className="text-white/70">
                        Configure your deployment preferences and target settings
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="text-center py-12">
                        <p className="text-white/70">Settings panel coming soon</p>
                      </div>
                    </CardContent>
                  </Card>
                </TabsContent>
              </Tabs>
            </>
          )}

          {/* Controller Connection Required Message */}
          {!controllerConnected && (
            <Card className="bg-slate-800/50 border-slate-700">
              <CardContent className="p-8 text-center">
                <div className="w-16 h-16 bg-blue-500/20 rounded-full flex items-center justify-center mx-auto mb-4">
                  <QrCode className="w-8 h-8 text-blue-400" />
                </div>
                <h3 className="text-xl font-bold text-white mb-2">Connect Your Controller to Continue</h3>
                <p className="text-slate-300 mb-6">
                  Connect your KNIRVCONTROLLER mobile app to unlock deployment management features and monitor your model deployments in real-time.
                </p>
                <Button
                  onClick={() => setQrModalOpen(true)}
                  size="lg"
                  className="bg-gradient-to-r from-blue-500 to-cyan-500 hover:from-blue-600 hover:to-cyan-600 text-white"
                >
                  <QrCode className="w-5 h-5 mr-2" />
                  Connect Controller
                </Button>
              </CardContent>
            </Card>
          )}

          <SettingsModal
            open={settingsModalOpen}
            onOpenChange={setSettingsModalOpen}
          />
        </>
      ) : (
        <div className="flex flex-col items-center justify-center min-h-[60vh] text-center">
          <div className="bg-slate-800/50 border border-slate-700 rounded-2xl p-8 max-w-lg w-full">
            <div className="w-16 h-16 bg-blue-500/20 rounded-full flex items-center justify-center mx-auto mb-4">
              <Smartphone className="w-8 h-8 text-blue-400" />
            </div>
            <h3 className="text-2xl font-bold text-white mb-2">Connect Your Controller</h3>
            <p className="text-slate-300 mb-6">
              Connect your KNIRV Controller app to monitor and manage your deployed models
            </p>

            <div className="space-y-4">
              <Button
                onClick={() => setQrModalOpen(true)}
                size="lg"
                className="bg-gradient-to-r from-blue-500 to-cyan-500 hover:from-blue-600 hover:to-cyan-600 text-white w-full"
              >
                <QrCode className="w-5 h-5 mr-2" />
                Connect with QR Code
              </Button>

              <div className="flex items-center space-x-4">
                <div className="flex-1 border-t border-slate-600"></div>
                <span className="text-slate-400 text-sm">or download the app</span>
                <div className="flex-1 border-t border-slate-600"></div>
              </div>

              <div className="flex justify-center space-x-4">
                <Button
                  size="lg"
                  variant="outline"
                  className="border-slate-600 text-slate-300 hover:bg-slate-700 flex-1 max-w-[120px]"
                  onClick={() => window.open('https://apps.apple.com/app/knirv-controller', '_blank')}
                >
                  <div className="flex flex-col items-center">
                    <span className="text-xs">Download on the</span>
                    <span className="font-bold">App Store</span>
                  </div>
                </Button>
                <Button
                  size="lg"
                  variant="outline"
                  className="border-slate-600 text-slate-300 hover:bg-slate-700 flex-1 max-w-[120px]"
                  onClick={() => window.open('https://play.google.com/store/apps/details?id=com.knirv.controller', '_blank')}
                >
                  <div className="flex flex-col items-center">
                    <span className="text-xs">Get it on</span>
                    <span className="font-bold">Google Play</span>
                  </div>
                </Button>
              </div>

              <p className="text-sm text-slate-400">
                Open the KNIRV Controller app and scan the QR code to connect
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default Dashboard;
