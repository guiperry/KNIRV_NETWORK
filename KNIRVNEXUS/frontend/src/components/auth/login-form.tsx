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
  const { login, loginWithCredentials, isLoading } = useAuth();
  const [loginMode, setLoginMode] = useState<'credentials' | 'token'>('credentials');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [token, setToken] = useState('');
  const [error, setError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (loginMode === 'credentials') {
      if (!username.trim() || !password.trim()) {
        setError('Please enter both username and password');
        return;
      }
    } else {
      if (!token.trim()) {
        setError('Please enter a token');
        return;
      }
    }

    setIsSubmitting(true);
    setError('');

    try {
      let success = false;
      if (loginMode === 'credentials') {
        success = await loginWithCredentials(username.trim(), password.trim());
        if (!success) {
          setError('Invalid username or password. Please check your credentials.');
        }
      } else {
        success = await login(token.trim());
        if (!success) {
          setError('Invalid token. Please check your credentials.');
        }
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
              Deterministic Validation Environment
            </CardDescription>
          </CardHeader>
          <CardContent>
            {/* Login Mode Toggle */}
            <div className="flex space-x-1 mb-6 p-1 bg-muted rounded-lg">
              <button
                type="button"
                onClick={() => setLoginMode('credentials')}
                className={`flex-1 py-2 px-3 rounded-md text-sm font-medium transition-colors ${
                  loginMode === 'credentials'
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                Username & Password
              </button>
              <button
                type="button"
                onClick={() => setLoginMode('token')}
                className={`flex-1 py-2 px-3 rounded-md text-sm font-medium transition-colors ${
                  loginMode === 'token'
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                Access Token
              </button>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              {loginMode === 'credentials' ? (
                <>
                  <div className="space-y-2">
                    <Label htmlFor="username">Username</Label>
                    <Input
                      id="username"
                      type="text"
                      placeholder="Enter your username"
                      value={username}
                      onChange={(e) => setUsername(e.target.value)}
                      disabled={isSubmitting || isLoading}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="password">Password</Label>
                    <Input
                      id="password"
                      type="password"
                      placeholder="Enter your password"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      disabled={isSubmitting || isLoading}
                    />
                  </div>
                </>
              ) : (
                <div className="space-y-2">
                  <Label htmlFor="token">Access Token</Label>
                  <Input
                    id="token"
                    type="password"
                    placeholder="Enter your access token"
                    value={token}
                    onChange={(e) => setToken(e.target.value)}
                    disabled={isSubmitting || isLoading}
                  />
                </div>
              )}

              {error && (
                <Alert className="border-destructive">
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}

              <Button
                type="submit"
                className="w-full"
                disabled={isSubmitting || (loginMode === 'credentials' ? (!username.trim() || !password.trim()) : !token.trim())}
              >
                {isSubmitting ? 'Authenticating...' : 'Login'}
              </Button>
            </form>

            {/* Testnet tokens for development - only show in token mode */}
            {loginMode === 'token' && (
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
            )}
          </CardContent>
        </Card>

        <div className="text-center text-sm text-muted-foreground">
          <p>Secure access to the KNIRV Deterministic Validation Environment</p>
          <p className="mt-1">Role-based authentication with TEE attestation</p>
        </div>
      </div>
    </div>
  );
}
