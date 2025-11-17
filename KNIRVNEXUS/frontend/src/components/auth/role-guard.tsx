'use client';

import React, { ReactNode } from 'react';
import { useAuth, Role } from '@/lib/auth-context';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Shield, Lock } from 'lucide-react';

interface RoleGuardProps {
  children: ReactNode;
  allowedRoles?: Role[];
  requiredPermission?: {
    service: string;
    operation: string;
  };
  nodeId?: string;
  fallback?: ReactNode;
  showError?: boolean;
}

export function RoleGuard({
  children,
  allowedRoles,
  requiredPermission,
  nodeId,
  fallback,
  showError = true
}: RoleGuardProps) {
  const { user, hasPermission, hasNodeAccess, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-4">
        <div data-testid="loading-spinner" className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  if (!user?.authenticated) {
    if (fallback) return <>{fallback}</>;
    
    if (showError) {
      return (
        <Alert className="border-destructive">
          <Lock className="h-4 w-4" />
          <AlertDescription>
            Authentication required to access this content.
          </AlertDescription>
        </Alert>
      );
    }
    
    return null;
  }

  // Check role-based access
  if (allowedRoles && !allowedRoles.includes(user.role)) {
    if (fallback) return <>{fallback}</>;
    
    if (showError) {
      return (
        <Alert className="border-destructive">
          <Shield className="h-4 w-4" />
          <AlertDescription>
            Insufficient permissions. Required role: {allowedRoles.join(' or ')}.
            Your role: {user.role}.
          </AlertDescription>
        </Alert>
      );
    }
    
    return null;
  }

  // Check permission-based access
  if (requiredPermission && !hasPermission(requiredPermission.service, requiredPermission.operation)) {
    if (fallback) return <>{fallback}</>;
    
    if (showError) {
      return (
        <Alert className="border-destructive">
          <Shield className="h-4 w-4" />
          <AlertDescription>
            Insufficient permissions for {requiredPermission.service}:{requiredPermission.operation}.
          </AlertDescription>
        </Alert>
      );
    }
    
    return null;
  }

  // Check node-specific access
  if (nodeId && !hasNodeAccess(nodeId)) {
    if (fallback) return <>{fallback}</>;
    
    if (showError) {
      return (
        <Alert className="border-destructive">
          <Shield className="h-4 w-4" />
          <AlertDescription>
            Access denied for node {nodeId}. You can only access nodes assigned to you.
          </AlertDescription>
        </Alert>
      );
    }
    
    return null;
  }

  return <>{children}</>;
}

// Convenience components for specific roles
export function AdminOnly({ children, fallback, showError = true }: Omit<RoleGuardProps, 'allowedRoles'>) {
  return (
    <RoleGuard allowedRoles={['admin']} fallback={fallback} showError={showError}>
      {children}
    </RoleGuard>
  );
}

export function ValidatorOnly({ children, fallback, showError = true }: Omit<RoleGuardProps, 'allowedRoles'>) {
  return (
    <RoleGuard allowedRoles={['validator']} fallback={fallback} showError={showError}>
      {children}
    </RoleGuard>
  );
}

export function ObserverOnly({ children, fallback, showError = true }: Omit<RoleGuardProps, 'allowedRoles'>) {
  return (
    <RoleGuard allowedRoles={['observer']} fallback={fallback} showError={showError}>
      {children}
    </RoleGuard>
  );
}

export function ValidatorOrAdmin({ children, fallback, showError = true }: Omit<RoleGuardProps, 'allowedRoles'>) {
  return (
    <RoleGuard allowedRoles={['validator', 'admin']} fallback={fallback} showError={showError}>
      {children}
    </RoleGuard>
  );
}

// Permission-based guards
export function DVEAccess({ 
  children, 
  operation = 'read', 
  fallback, 
  showError = true 
}: Omit<RoleGuardProps, 'requiredPermission'> & { operation?: string }) {
  return (
    <RoleGuard 
      requiredPermission={{ service: 'dve', operation }} 
      fallback={fallback} 
      showError={showError}
    >
      {children}
    </RoleGuard>
  );
}

export function ValidationAccess({ 
  children, 
  operation = 'read', 
  fallback, 
  showError = true 
}: Omit<RoleGuardProps, 'requiredPermission'> & { operation?: string }) {
  return (
    <RoleGuard 
      requiredPermission={{ service: 'validation', operation }} 
      fallback={fallback} 
      showError={showError}
    >
      {children}
    </RoleGuard>
  );
}

export function SystemAccess({ 
  children, 
  operation = 'read', 
  fallback, 
  showError = true 
}: Omit<RoleGuardProps, 'requiredPermission'> & { operation?: string }) {
  return (
    <RoleGuard 
      requiredPermission={{ service: 'system', operation }} 
      fallback={fallback} 
      showError={showError}
    >
      {children}
    </RoleGuard>
  );
}
