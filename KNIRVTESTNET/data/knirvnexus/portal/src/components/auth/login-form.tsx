'use client';

import React, { useState } from 'react';
import { useAuth } from '@/lib/auth-context';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Shield, Key, User, Eye } from 'lucide-react';

export function LoginForm() {
  const { login, isLoading } = useAuth();
  const [token, setToken] = useState('');
  const [error, setError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token.trim()) {
      setError('Please enter a token');
      return;
    }

    setIsSubmitting(true);
    setError('');

    try {
      const success = await login(token.trim());
      if (!success) {
        setError('Invalid token. Please check your credentials.');
      }
    } catch (err) {
      setError('Login failed. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  };

  const setTestnetToken = (testToken: string) => {
    setToken(testToken);
    setError('');
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-background to-muted p-4">
      <div className="w-full max-w-md space-y-6">
        <Card>
          <CardHeader className="text-center">
            <div className="mx-auto w-12 h-12 bg-primary/10 rounded-full flex items-center justify-center mb-4">
              <Shield className="w-6 h-6 text-primary" />
            </div>
            <CardTitle className="text-2xl">KNIRV NEXUS</CardTitle>
            <CardDescription>
              Decentralized Validation Environment
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="token">Access Token</Label>
                <Input
                  id="token"
                  type="password"
                  placeholder="Enter your access token"
                  value={token}
                  onChange={(e) => setToken(e.target.value)}
                  disabled={isSubmitting}
                />
              </div>

              {error && (
                <Alert className="border-destructive">
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}

              <Button 
                type="submit" 
                className="w-full" 
                disabled={isSubmitting || !token.trim()}
              >
                {isSubmitting ? 'Authenticating...' : 'Login'}
              </Button>
            </form>

            {/* Testnet tokens for development */}
            <div className="mt-6 pt-6 border-t">
              <h3 className="text-sm font-medium mb-3 text-center text-muted-foreground">
                Testnet Tokens (Development)
              </h3>
              <div className="space-y-2">
                <div className="flex items-center justify-between p-2 rounded-lg bg-muted/50">
                  <div className="flex items-center space-x-2">
                    <User className="w-4 h-4" />
                    <span className="text-sm">Admin</span>
                    <Badge variant="destructive" className="text-xs">Full Access</Badge>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setTestnetToken('testnet-admin-123')}
                    disabled={isSubmitting}
                  >
                    Use
                  </Button>
                </div>

                <div className="flex items-center justify-between p-2 rounded-lg bg-muted/50">
                  <div className="flex items-center space-x-2">
                    <Shield className="w-4 h-4" />
                    <span className="text-sm">Validator</span>
                    <Badge variant="secondary" className="text-xs">Scoped</Badge>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setTestnetToken('testnet-validator-456')}
                    disabled={isSubmitting}
                  >
                    Use
                  </Button>
                </div>

                <div className="flex items-center justify-between p-2 rounded-lg bg-muted/50">
                  <div className="flex items-center space-x-2">
                    <Eye className="w-4 h-4" />
                    <span className="text-sm">Observer</span>
                    <Badge variant="outline" className="text-xs">Read Only</Badge>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setTestnetToken('testnet-observer-789')}
                    disabled={isSubmitting}
                  >
                    Use
                  </Button>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <div className="text-center text-sm text-muted-foreground">
          <p>Secure access to the KNIRV Decentralized Validation Environment</p>
          <p className="mt-1">Role-based authentication with TEE attestation</p>
        </div>
      </div>
    </div>
  );
}
