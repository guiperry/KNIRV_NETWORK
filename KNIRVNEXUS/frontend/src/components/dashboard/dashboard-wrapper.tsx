'use client';

import React, { useState } from 'react';
import { useAuth, ROLES } from '@/lib/auth-context';
import { LoginForm } from '@/components/auth/login-form';
import { UserProfile } from '@/components/auth/user-profile';
import { RoleGuard, DVEAccess, ValidationAccess, SystemAccess } from '@/components/auth/role-guard';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { CDEAccessModal } from '@/components/cde/cde-access-modal';
import { KNIRVEngineModal } from '@/components/knirvengine/knirvengine-modal';
import { CognitiveEnginePanel } from '@/components/dashboard/cognitive-engine-panel';
import { DVENodesPanel } from '@/components/dashboard/dve-nodes-panel';
import {
  Shield,
  Server,
  Activity,
  Eye,
  User,
  Settings,
  BarChart3,
  Lock,
  Unlock,
  Network,
  Cpu,
  HardDrive,
  Wifi,
  Globe,
  Database,
  Monitor,
  Zap,
  Download,
  Share2
} from 'lucide-react';

interface DashboardWrapperProps {
  children: React.ReactNode;
  onRentDVE?: () => void;
}

export function DashboardWrapper({ children, onRentDVE }: DashboardWrapperProps) {
  const { user, isLoading } = useAuth();
  const [cdeModalOpen, setCdeModalOpen] = useState(false);
  const [knirvEngineModalOpen, setKnirvEngineModalOpen] = useState(false);
  const [selectedNode, setSelectedNode] = useState<{id: string, name: string} | null>(null);

  const handleNodeAccess = (nodeId: string, nodeName: string) => {
    setSelectedNode({ id: nodeId, name: nodeName });
    setCdeModalOpen(true);
  };

  const handleOpenKNIRVEngine = () => {
    setCdeModalOpen(false);
    setKnirvEngineModalOpen(true);
  };

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div data-testid="loading-spinner" className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
      </div>
    );
  }

  if (!user?.authenticated) {
    return <LoginForm />;
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-background to-muted">
      {/* Header with user profile */}
      <header className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <div className="flex items-center space-x-2">
                <Shield className="w-8 h-8 text-primary" />
                <div>
                  <h1 className="text-2xl font-bold">KNIRV NEXUS</h1>
                  <p className="text-sm text-muted-foreground">Deterministic Validation Environment</p>
                </div>
              </div>
            </div>
            
            <div className="flex items-center space-x-4">
              <div className="flex items-center space-x-2 text-sm">
                <span className="text-muted-foreground">Welcome,</span>
                <span className="font-medium">{user.user}</span>
                <Badge variant={user.role === 'admin' ? 'destructive' : user.role === 'validator' ? 'secondary' : 'outline'}>
                  {ROLES[user.role]?.displayName || user.role.toUpperCase()}
                </Badge>
              </div>
              <UserProfile />
            </div>
          </div>
        </div>
      </header>

      {/* Role-based navigation */}
      <nav className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="container mx-auto px-4">
          <Tabs defaultValue="overview" className="w-full">
            <TabsList className="grid w-full grid-cols-4 h-12">
              <TabsTrigger value="overview" className="flex items-center space-x-2">
                <BarChart3 className="w-4 h-4" />
                <span>Overview</span>
              </TabsTrigger>

              <SystemAccess operation="read" showError={false}>
                <TabsTrigger value="system" className="flex items-center space-x-2">
                  <Settings className="w-4 h-4" />
                  <span>Network & Resources</span>
                </TabsTrigger>
              </SystemAccess>

              <RoleGuard allowedRoles={['admin']} showError={false}>
                <TabsTrigger value="admin" className="flex items-center space-x-2">
                  <User className="w-4 h-4" />
                  <span>Admin</span>
                </TabsTrigger>
              </RoleGuard>

              <TabsTrigger value="profile" className="flex items-center space-x-2">
                <Eye className="w-4 h-4" />
                <span>Profile</span>
              </TabsTrigger>
            </TabsList>
            
            <div className="py-6">
              <TabsContent value="overview">
                {children}
              </TabsContent>
              

              
              <TabsContent value="system">
                <SystemAccess operation="read">
                  <div className="space-y-6">
                    <div className="flex items-center justify-between">
                      <div>
                        <h2 className="text-2xl font-bold">Network & Resource Explorer</h2>
                        <p className="text-muted-foreground">
                          Monitor network topology, resource allocation, and system performance across the KNIRV ecosystem.
                        </p>
                      </div>
                      <Button variant="outline" className="flex items-center space-x-2">
                        <Download className="w-4 h-4" />
                        <span>Export Report</span>
                      </Button>
                    </div>

                    {/* Network Overview Cards */}
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                      <Card className="knirv-card-gradient">
                        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                          <CardTitle className="text-sm font-medium">Network Nodes</CardTitle>
                          <Network className="h-4 w-4 text-muted-foreground" />
                        </CardHeader>
                        <CardContent>
                          <div className="text-2xl font-bold">24</div>
                          <p className="text-xs text-muted-foreground">
                            18 active, 6 standby
                          </p>
                        </CardContent>
                      </Card>

                      <Card className="knirv-card-gradient">
                        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                          <CardTitle className="text-sm font-medium">CPU Utilization</CardTitle>
                          <Cpu className="h-4 w-4 text-muted-foreground" />
                        </CardHeader>
                        <CardContent>
                          <div className="text-2xl font-bold">67%</div>
                          <p className="text-xs text-muted-foreground">
                            Across all nodes
                          </p>
                        </CardContent>
                      </Card>

                      <Card className="knirv-card-gradient">
                        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                          <CardTitle className="text-sm font-medium">Storage Used</CardTitle>
                          <HardDrive className="h-4 w-4 text-muted-foreground" />
                        </CardHeader>
                        <CardContent>
                          <div className="text-2xl font-bold">2.4TB</div>
                          <p className="text-xs text-muted-foreground">
                            Of 5.2TB total
                          </p>
                        </CardContent>
                      </Card>

                      <Card className="knirv-card-gradient">
                        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                          <CardTitle className="text-sm font-medium">Network Bandwidth</CardTitle>
                          <Wifi className="h-4 w-4 text-muted-foreground" />
                        </CardHeader>
                        <CardContent>
                          <div className="text-2xl font-bold">1.2GB/s</div>
                          <p className="text-xs text-muted-foreground">
                            Current throughput
                          </p>
                        </CardContent>
                      </Card>
                    </div>

                    {/* Resource Explorer Tabs */}
                    <Tabs defaultValue="nodes" className="space-y-4">
                      <TabsList className="grid w-full grid-cols-4">
                        <TabsTrigger value="nodes">DVE Nodes</TabsTrigger>
                        <TabsTrigger value="validation">Validation Tasks</TabsTrigger>
                        <TabsTrigger value="tee">TEE Security</TabsTrigger>
                        <TabsTrigger value="cognitive">Cognitive Engine</TabsTrigger>
                      </TabsList>

                      <TabsContent value="nodes" className="space-y-4">
                        <DVENodesPanel onRentClick={onRentDVE} onNodeConnect={handleNodeAccess} />
                      </TabsContent>

                      <TabsContent value="validation" className="space-y-4">
                        <div className="flex items-center justify-between">
                          <h3 className="text-lg font-semibold">Validation Task Status</h3>
                          <div className="flex space-x-2">
                            <Button variant="outline" size="sm">
                              <Download className="w-4 h-4 mr-2" />
                              Export
                            </Button>
                            <Button variant="outline" size="sm">
                              <Share2 className="w-4 h-4 mr-2" />
                              Share
                            </Button>
                          </div>
                        </div>
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                          <Card className="knirv-card-gradient">
                            <CardHeader>
                              <CardTitle className="flex items-center space-x-2">
                                <Activity className="w-5 h-5" />
                                <span>Active Tasks</span>
                              </CardTitle>
                            </CardHeader>
                            <CardContent>
                              <div className="text-3xl font-bold">127</div>
                              <p className="text-sm text-muted-foreground">Currently processing</p>
                              <Button variant="outline" size="sm" className="w-full mt-4">
                                <Download className="w-3 h-3 mr-1" />
                                Task Reports
                              </Button>
                            </CardContent>
                          </Card>
                          <Card className="knirv-card-gradient">
                            <CardHeader>
                              <CardTitle className="flex items-center space-x-2">
                                <BarChart3 className="w-5 h-5" />
                                <span>Completed Today</span>
                              </CardTitle>
                            </CardHeader>
                            <CardContent>
                              <div className="text-3xl font-bold">1,847</div>
                              <p className="text-sm text-muted-foreground">98.2% success rate</p>
                              <Button variant="outline" size="sm" className="w-full mt-4">
                                <Download className="w-3 h-3 mr-1" />
                                Performance Reports
                              </Button>
                            </CardContent>
                          </Card>
                        </div>
                      </TabsContent>

                      <TabsContent value="tee" className="space-y-4">
                        <div className="flex items-center justify-between">
                          <h3 className="text-lg font-semibold">TEE Security Status</h3>
                          <div className="flex space-x-2">
                            <Button variant="outline" size="sm">
                              <Download className="w-4 h-4 mr-2" />
                              Export
                            </Button>
                            <Button variant="outline" size="sm">
                              <Share2 className="w-4 h-4 mr-2" />
                              Share
                            </Button>
                          </div>
                        </div>
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                          <Card className="knirv-card-gradient">
                            <CardHeader>
                              <CardTitle className="flex items-center space-x-2">
                                <Shield className="w-5 h-5" />
                                <span>SGX Enclaves</span>
                              </CardTitle>
                            </CardHeader>
                            <CardContent>
                              <div className="text-2xl font-bold">18</div>
                              <p className="text-sm text-muted-foreground">Active secure enclaves</p>
                              <Button variant="outline" size="sm" className="w-full mt-4">
                                <Download className="w-3 h-3 mr-1" />
                                Security Reports
                              </Button>
                            </CardContent>
                          </Card>
                          <Card className="knirv-card-gradient">
                            <CardHeader>
                              <CardTitle className="flex items-center space-x-2">
                                <Lock className="w-5 h-5" />
                                <span>SEV-SNP</span>
                              </CardTitle>
                            </CardHeader>
                            <CardContent>
                              <div className="text-2xl font-bold">6</div>
                              <p className="text-sm text-muted-foreground">Secure VMs running</p>
                              <Button variant="outline" size="sm" className="w-full mt-4">
                                <Download className="w-3 h-3 mr-1" />
                                VM Reports
                              </Button>
                            </CardContent>
                          </Card>
                          <Card className="knirv-card-gradient">
                            <CardHeader>
                              <CardTitle className="flex items-center space-x-2">
                                <Zap className="w-5 h-5" />
                                <span>TDX</span>
                              </CardTitle>
                            </CardHeader>
                            <CardContent>
                              <div className="text-2xl font-bold">3</div>
                              <p className="text-sm text-muted-foreground">Trust domains active</p>
                              <Button variant="outline" size="sm" className="w-full mt-4">
                                <Download className="w-3 h-3 mr-1" />
                                Trust Reports
                              </Button>
                            </CardContent>
                          </Card>
                        </div>
                      </TabsContent>

                      <TabsContent value="cognitive" className="space-y-4">
                        <CognitiveEnginePanel />
                      </TabsContent>
                    </Tabs>
                  </div>
                </SystemAccess>
              </TabsContent>
              
              <TabsContent value="admin">
                <RoleGuard allowedRoles={['admin']}>
                  <div className="space-y-4">
                    <h2 className="text-2xl font-bold">Administration</h2>
                    <p className="text-muted-foreground">
                      Administrative controls and system configuration.
                    </p>
                    
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                      <Card>
                        <CardHeader>
                          <CardTitle className="flex items-center space-x-2">
                            <User className="w-5 h-5" />
                            <span>User Management</span>
                          </CardTitle>
                          <CardDescription>
                            Manage user roles and permissions
                          </CardDescription>
                        </CardHeader>
                        <CardContent>
                          <Button variant="outline" className="w-full">
                            Manage Users
                          </Button>
                        </CardContent>
                      </Card>
                      
                      <Card>
                        <CardHeader>
                          <CardTitle className="flex items-center space-x-2">
                            <Settings className="w-5 h-5" />
                            <span>System Config</span>
                          </CardTitle>
                          <CardDescription>
                            Configure system parameters
                          </CardDescription>
                        </CardHeader>
                        <CardContent>
                          <Button variant="outline" className="w-full">
                            Configure
                          </Button>
                        </CardContent>
                      </Card>
                      
                      <Card>
                        <CardHeader>
                          <CardTitle className="flex items-center space-x-2">
                            <Shield className="w-5 h-5" />
                            <span>Security</span>
                          </CardTitle>
                          <CardDescription>
                            Security settings and audit logs
                          </CardDescription>
                        </CardHeader>
                        <CardContent>
                          <Button variant="outline" className="w-full">
                            Security Center
                          </Button>
                        </CardContent>
                      </Card>
                    </div>
                  </div>
                </RoleGuard>
              </TabsContent>
              
              <TabsContent value="profile">
                <div className="space-y-6">
                  <div className="flex items-center justify-between">
                    <div>
                      <h2 className="text-2xl font-bold">User Profile</h2>
                      <p className="text-muted-foreground">
                        Your account information, permissions, and usage analytics.
                      </p>
                    </div>
                    <Button variant="outline" className="flex items-center space-x-2">
                      <Download className="w-4 h-4" />
                      <span>Export Profile</span>
                    </Button>
                  </div>

                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    {/* Account Information */}
                    <Card>
                      <CardHeader>
                        <CardTitle>Account Information</CardTitle>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <div className="grid grid-cols-2 gap-4">
                          <div>
                            <p className="text-sm font-medium text-muted-foreground">Username</p>
                            <p className="text-lg">{user.user}</p>
                          </div>
                          <div>
                            <p className="text-sm font-medium text-muted-foreground">Role</p>
                            <Badge variant={user.role === 'admin' ? 'destructive' : user.role === 'validator' ? 'secondary' : 'outline'}>
                              {user.role.toUpperCase()}
                            </Badge>
                          </div>
                        </div>

                        {user.node_id && (
                          <div>
                            <p className="text-sm font-medium text-muted-foreground">Assigned Node</p>
                            <Badge variant="outline">{user.node_id}</Badge>
                          </div>
                        )}

                        <div>
                          <p className="text-sm font-medium text-muted-foreground mb-2">NEXUS Permissions</p>
                          <div className="flex flex-wrap gap-2">
                            {user.nexus_access.map((permission, index) => (
                              <Badge key={index} variant="secondary" className="text-xs">
                                {permission}
                              </Badge>
                            ))}
                          </div>
                        </div>
                      </CardContent>
                    </Card>

                    {/* Billing & Usage Summary */}
                    <Card>
                      <CardHeader>
                        <CardTitle className="flex items-center space-x-2">
                          <BarChart3 className="w-5 h-5" />
                          <span>Usage Summary</span>
                        </CardTitle>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <div className="grid grid-cols-2 gap-4">
                          <div>
                            <p className="text-sm font-medium text-muted-foreground">Compute Hours</p>
                            <p className="text-2xl font-bold">247.3</p>
                            <p className="text-xs text-muted-foreground">This month</p>
                          </div>
                          <div>
                            <p className="text-sm font-medium text-muted-foreground">Storage Used</p>
                            <p className="text-2xl font-bold">1.2TB</p>
                            <p className="text-xs text-muted-foreground">Current usage</p>
                          </div>
                        </div>
                        <div className="grid grid-cols-2 gap-4">
                          <div>
                            <p className="text-sm font-medium text-muted-foreground">API Calls</p>
                            <p className="text-2xl font-bold">12.4K</p>
                            <p className="text-xs text-muted-foreground">This month</p>
                          </div>
                          <div>
                            <p className="text-sm font-medium text-muted-foreground">NRN Spent</p>
                            <p className="text-2xl font-bold">1,847</p>
                            <p className="text-xs text-muted-foreground">Total this month</p>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  </div>

                  {/* Billing & Usage Reports */}
                  <Card>
                    <CardHeader>
                      <div className="flex items-center justify-between">
                        <CardTitle className="flex items-center space-x-2">
                          <Download className="w-5 h-5" />
                          <span>Billing & Usage Reports</span>
                        </CardTitle>
                        <div className="flex space-x-2">
                          <Button variant="outline" size="sm">
                            <Download className="w-4 h-4 mr-2" />
                            Download All
                          </Button>
                          <Button variant="outline" size="sm">
                            <Share2 className="w-4 h-4 mr-2" />
                            Share Reports
                          </Button>
                        </div>
                      </div>
                      <CardDescription>
                        Generate and download detailed usage and billing reports for your account.
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                        <Card className="knirv-card-gradient">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm">Monthly Usage Report</CardTitle>
                            <CardDescription className="text-xs">
                              Detailed breakdown of compute, storage, and API usage
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span>Last Generated:</span>
                              <span>Dec 15, 2024</span>
                            </div>
                            <div className="flex space-x-2">
                              <Button variant="outline" size="sm" className="flex-1">
                                <Download className="w-3 h-3 mr-1" />
                                Download
                              </Button>
                              <Button variant="outline" size="sm" className="flex-1">
                                <Share2 className="w-3 h-3 mr-1" />
                                Share
                              </Button>
                            </div>
                          </CardContent>
                        </Card>

                        <Card className="knirv-card-gradient">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm">Billing Statement</CardTitle>
                            <CardDescription className="text-xs">
                              NRN transactions and payment history
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span>Last Generated:</span>
                              <span>Dec 1, 2024</span>
                            </div>
                            <div className="flex space-x-2">
                              <Button variant="outline" size="sm" className="flex-1">
                                <Download className="w-3 h-3 mr-1" />
                                Download
                              </Button>
                              <Button variant="outline" size="sm" className="flex-1">
                                <Share2 className="w-3 h-3 mr-1" />
                                Share
                              </Button>
                            </div>
                          </CardContent>
                        </Card>

                        <Card className="knirv-card-gradient">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm">Performance Analytics</CardTitle>
                            <CardDescription className="text-xs">
                              Task completion rates and efficiency metrics
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span>Last Generated:</span>
                              <span>Dec 16, 2024</span>
                            </div>
                            <div className="flex space-x-2">
                              <Button variant="outline" size="sm" className="flex-1">
                                <Download className="w-3 h-3 mr-1" />
                                Download
                              </Button>
                              <Button variant="outline" size="sm" className="flex-1">
                                <Share2 className="w-3 h-3 mr-1" />
                                Share
                              </Button>
                            </div>
                          </CardContent>
                        </Card>

                        <Card className="knirv-card-gradient">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm">Security Audit Log</CardTitle>
                            <CardDescription className="text-xs">
                              Access logs and security events
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span>Last Generated:</span>
                              <span>Dec 16, 2024</span>
                            </div>
                            <div className="flex space-x-2">
                              <Button variant="outline" size="sm" className="flex-1">
                                <Download className="w-3 h-3 mr-1" />
                                Download
                              </Button>
                              <Button variant="outline" size="sm" className="flex-1">
                                <Share2 className="w-3 h-3 mr-1" />
                                Share
                              </Button>
                            </div>
                          </CardContent>
                        </Card>

                        <Card className="knirv-card-gradient">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm">Resource Utilization</CardTitle>
                            <CardDescription className="text-xs">
                              Detailed resource consumption patterns
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span>Last Generated:</span>
                              <span>Dec 15, 2024</span>
                            </div>
                            <div className="flex space-x-2">
                              <Button variant="outline" size="sm" className="flex-1">
                                <Download className="w-3 h-3 mr-1" />
                                Download
                              </Button>
                              <Button variant="outline" size="sm" className="flex-1">
                                <Share2 className="w-3 h-3 mr-1" />
                                Share
                              </Button>
                            </div>
                          </CardContent>
                        </Card>

                        <Card className="knirv-card-gradient">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm">Custom Report</CardTitle>
                            <CardDescription className="text-xs">
                              Generate custom reports with specific parameters
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span>Status:</span>
                              <span>Ready to generate</span>
                            </div>
                            <Button variant="outline" size="sm" className="w-full">
                              <Settings className="w-3 h-3 mr-1" />
                              Configure & Generate
                            </Button>
                          </CardContent>
                        </Card>
                      </div>
                    </CardContent>
                  </Card>
                </div>
              </TabsContent>
            </div>
          </Tabs>
        </div>
      </nav>

      {/* CDE Access Modal */}
      {selectedNode && (
        <CDEAccessModal
          isOpen={cdeModalOpen}
          onClose={() => setCdeModalOpen(false)}
          nodeId={selectedNode.id}
          nodeName={selectedNode.name}
          onOpenKNIRVEngine={handleOpenKNIRVEngine}
        />
      )}

      {/* KNIRVENGINE Modal */}
      {selectedNode && (
        <KNIRVEngineModal
          isOpen={knirvEngineModalOpen}
          onClose={() => setKnirvEngineModalOpen(false)}
          nodeId={selectedNode.id}
        />
      )}
    </div>
  );
}
