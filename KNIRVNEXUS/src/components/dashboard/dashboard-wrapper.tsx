'use client';

import React from 'react';
import { useAuth } from '@/lib/auth-context';
import { LoginForm } from '@/components/auth/login-form';
import { UserProfile } from '@/components/auth/user-profile';
import { RoleGuard, DVEAccess, ValidationAccess, SystemAccess } from '@/components/auth/role-guard';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { 
  Shield, 
  Server, 
  Activity, 
  Eye, 
  User, 
  Settings,
  BarChart3,
  Lock,
  Unlock
} from 'lucide-react';

interface DashboardWrapperProps {
  children: React.ReactNode;
}

export function DashboardWrapper({ children }: DashboardWrapperProps) {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
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
                  <p className="text-sm text-muted-foreground">Decentralized Validation Environment</p>
                </div>
              </div>
            </div>
            
            <div className="flex items-center space-x-4">
              <div className="flex items-center space-x-2 text-sm">
                <span className="text-muted-foreground">Welcome,</span>
                <span className="font-medium">{user.user}</span>
                <Badge variant={user.role === 'admin' ? 'destructive' : user.role === 'validator' ? 'secondary' : 'outline'}>
                  {user.role.toUpperCase()}
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
            <TabsList className="grid w-full grid-cols-6 h-12">
              <TabsTrigger value="overview" className="flex items-center space-x-2">
                <BarChart3 className="w-4 h-4" />
                <span>Overview</span>
              </TabsTrigger>
              
              <DVEAccess operation="read" showError={false}>
                <TabsTrigger value="nodes" className="flex items-center space-x-2">
                  <Server className="w-4 h-4" />
                  <span>DVE Nodes</span>
                </TabsTrigger>
              </DVEAccess>
              
              <ValidationAccess operation="read" showError={false}>
                <TabsTrigger value="validation" className="flex items-center space-x-2">
                  <Activity className="w-4 h-4" />
                  <span>Validation</span>
                </TabsTrigger>
              </ValidationAccess>
              
              <SystemAccess operation="read" showError={false}>
                <TabsTrigger value="system" className="flex items-center space-x-2">
                  <Settings className="w-4 h-4" />
                  <span>System</span>
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
              
              <TabsContent value="nodes">
                <DVEAccess operation="read">
                  <div className="space-y-4">
                    <h2 className="text-2xl font-bold">DVE Nodes Management</h2>
                    <p className="text-muted-foreground">
                      Monitor and manage Decentralized Validation Environment nodes.
                    </p>
                    {/* DVE Nodes content would go here */}
                    <Alert>
                      <Server className="h-4 w-4" />
                      <AlertDescription>
                        DVE Nodes interface is being loaded...
                      </AlertDescription>
                    </Alert>
                  </div>
                </DVEAccess>
              </TabsContent>
              
              <TabsContent value="validation">
                <ValidationAccess operation="read">
                  <div className="space-y-4">
                    <h2 className="text-2xl font-bold">Validation Tasks</h2>
                    <p className="text-muted-foreground">
                      View and manage validation tasks and results.
                    </p>
                    
                    <ValidationAccess operation="execute" showError={false}>
                      <Alert>
                        <Unlock className="h-4 w-4" />
                        <AlertDescription>
                          You have permission to execute validation tasks.
                        </AlertDescription>
                      </Alert>
                    </ValidationAccess>
                    
                    <ValidationAccess operation="execute" fallback={
                      <Alert>
                        <Lock className="h-4 w-4" />
                        <AlertDescription>
                          You have read-only access to validation tasks.
                        </AlertDescription>
                      </Alert>
                    } showError={false}>
                      <div className="mt-4">
                        <p className="text-sm text-muted-foreground">
                          Execute validation tasks and manage validation workflows.
                        </p>
                      </div>
                    </ValidationAccess>
                  </div>
                </ValidationAccess>
              </TabsContent>
              
              <TabsContent value="system">
                <SystemAccess operation="read">
                  <div className="space-y-4">
                    <h2 className="text-2xl font-bold">System Status</h2>
                    <p className="text-muted-foreground">
                      Monitor system health, metrics, and performance.
                    </p>
                    {/* System status content would go here */}
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
                <div className="space-y-4">
                  <h2 className="text-2xl font-bold">User Profile</h2>
                  <p className="text-muted-foreground">
                    Your account information and permissions.
                  </p>
                  
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
                </div>
              </TabsContent>
            </div>
          </Tabs>
        </div>
      </nav>
    </div>
  );
}
