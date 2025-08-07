'use client';

import React from 'react';
import { useAuth, ROLES } from '@/lib/auth-context';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { 
  DropdownMenu, 
  DropdownMenuContent, 
  DropdownMenuItem, 
  DropdownMenuLabel, 
  DropdownMenuSeparator, 
  DropdownMenuTrigger 
} from '@/components/ui/dropdown-menu';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import { User, Shield, Eye, LogOut, Settings, Key } from 'lucide-react';

export function UserProfile() {
  const { user, logout } = useAuth();

  if (!user?.authenticated) {
    return null;
  }

  const getRoleIcon = (role: string) => {
    switch (role) {
      case 'admin':
        return <User className="w-4 h-4" />;
      case 'validator':
        return <Shield className="w-4 h-4" />;
      case 'observer':
        return <Eye className="w-4 h-4" />;
      default:
        return <User className="w-4 h-4" />;
    }
  };

  const getRoleBadgeVariant = (role: string) => {
    switch (role) {
      case 'admin':
        return 'destructive' as const;
      case 'validator':
        return 'secondary' as const;
      case 'observer':
        return 'outline' as const;
      default:
        return 'outline' as const;
    }
  };

  const getUserInitials = (username: string) => {
    return username
      .split('-')
      .map(part => part.charAt(0).toUpperCase())
      .join('')
      .slice(0, 2);
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" className="relative h-10 w-10 rounded-full">
          <Avatar className="h-10 w-10">
            <AvatarFallback className="bg-primary/10">
              {getUserInitials(user.user)}
            </AvatarFallback>
          </Avatar>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-80" align="end" forceMount>
        <DropdownMenuLabel className="font-normal">
          <div className="flex flex-col space-y-2">
            <div className="flex items-center space-x-2">
              {getRoleIcon(user.role)}
              <p className="text-sm font-medium leading-none">{user.user}</p>
            </div>
            <div className="flex items-center space-x-2">
              <Badge variant={getRoleBadgeVariant(user.role)} className="text-xs">
                {user.role.toUpperCase()}
              </Badge>
              {user.node_id && (
                <Badge variant="outline" className="text-xs">
                  Node: {user.node_id}
                </Badge>
              )}
            </div>
            <p className="text-xs leading-none text-muted-foreground">
              {ROLES[user.role]?.description}
            </p>
          </div>
        </DropdownMenuLabel>
        
        <DropdownMenuSeparator />
        
        <div className="p-2">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm">Permissions</CardTitle>
              <CardDescription className="text-xs">
                Your access level in the NEXUS system
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-0">
              <div className="space-y-2">
                <div>
                  <p className="text-xs font-medium text-muted-foreground mb-1">NEXUS Access:</p>
                  <div className="flex flex-wrap gap-1">
                    {user.nexus_access.map((permission, index) => (
                      <Badge key={index} variant="outline" className="text-xs">
                        {permission}
                      </Badge>
                    ))}
                  </div>
                </div>
                
                {user.permissions && user.permissions.length > 0 && (
                  <div>
                    <p className="text-xs font-medium text-muted-foreground mb-1">General:</p>
                    <div className="flex flex-wrap gap-1">
                      {user.permissions.slice(0, 3).map((permission, index) => (
                        <Badge key={index} variant="secondary" className="text-xs">
                          {permission}
                        </Badge>
                      ))}
                      {user.permissions.length > 3 && (
                        <Badge variant="secondary" className="text-xs">
                          +{user.permissions.length - 3} more
                        </Badge>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
        
        <DropdownMenuSeparator />
        
        <DropdownMenuItem disabled>
          <Settings className="mr-2 h-4 w-4" />
          <span>Settings</span>
        </DropdownMenuItem>
        
        <DropdownMenuItem disabled>
          <Key className="mr-2 h-4 w-4" />
          <span>API Keys</span>
        </DropdownMenuItem>
        
        <DropdownMenuSeparator />
        
        <DropdownMenuItem onClick={logout} className="text-destructive">
          <LogOut className="mr-2 h-4 w-4" />
          <span>Log out</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

// Compact version for smaller spaces
export function UserProfileCompact() {
  const { user, logout } = useAuth();

  if (!user?.authenticated) {
    return null;
  }

  const getRoleIcon = (role: string) => {
    switch (role) {
      case 'admin':
        return <User className="w-3 h-3" />;
      case 'validator':
        return <Shield className="w-3 h-3" />;
      case 'observer':
        return <Eye className="w-3 h-3" />;
      default:
        return <User className="w-3 h-3" />;
    }
  };

  const getRoleBadgeVariant = (role: string) => {
    switch (role) {
      case 'admin':
        return 'destructive' as const;
      case 'validator':
        return 'secondary' as const;
      case 'observer':
        return 'outline' as const;
      default:
        return 'outline' as const;
    }
  };

  return (
    <div className="flex items-center space-x-2 text-sm">
      <div className="flex items-center space-x-1">
        {getRoleIcon(user.role)}
        <span className="font-medium">{user.user}</span>
      </div>
      <Badge variant={getRoleBadgeVariant(user.role)} className="text-xs">
        {user.role}
      </Badge>
      <Button variant="ghost" size="sm" onClick={logout}>
        <LogOut className="w-3 h-3" />
      </Button>
    </div>
  );
}
