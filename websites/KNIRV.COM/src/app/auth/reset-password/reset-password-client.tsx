'use client';

import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Eye, EyeOff, Lock } from "lucide-react";
import { useToast } from "@/hooks/use-toast";
import { supabaseAuthService } from '@/services/supabase-auth-service';

type SearchParams = {
  access_token?: string;
  refresh_token?: string;
};

export default function ResetPasswordClient({
  searchParams,
}: {
  searchParams: SearchParams;
}) {
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const [isValidLink, setIsValidLink] = useState(true);

  const router = useRouter();
  const { toast } = useToast();

  useEffect(() => {
    const accessToken = searchParams.access_token;
    const refreshToken = searchParams.refresh_token;

    if (!accessToken || !refreshToken) {
      setIsValidLink(false);
      toast({
        title: "Invalid Reset Link",
        description: "This password reset link is invalid or has expired. Please request a new one.",
        variant: "destructive"
      });

      const timer = window.setTimeout(() => {
        router.push('/');
      }, 2500);

      return () => window.clearTimeout(timer);
    }

    return undefined;
  }, [searchParams.access_token, searchParams.refresh_token, router, toast]);

  const handleResetPassword = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!password.trim()) {
      toast({
        title: "Password Required",
        description: "Please enter a new password",
        variant: "destructive"
      });
      return;
    }

    if (password.length < 6) {
      toast({
        title: "Password Too Short",
        description: "Password must be at least 6 characters long",
        variant: "destructive"
      });
      return;
    }

    if (password !== confirmPassword) {
      toast({
        title: "Passwords Don't Match",
        description: "Please make sure both passwords match",
        variant: "destructive"
      });
      return;
    }

    setIsLoading(true);
    try {
      const supabase = supabaseAuthService.getSupabaseClient();
      const { error } = await supabase.auth.updateUser({
        password,
      });

      if (error) {
        throw new Error(error.message);
      }

      setIsSuccess(true);
      toast({
        title: "Password Updated",
        description: "Your password has been successfully updated. You can now sign in with your new password.",
      });

      setTimeout(() => {
        router.push('/');
      }, 3000);
    } catch (error) {
      console.error('Password reset failed:', error);
      toast({
        title: "Reset Failed",
        description: error instanceof Error ? error.message : "Failed to update password. Please try again.",
        variant: "destructive"
      });
    } finally {
      setIsLoading(false);
    }
  };

  if (!isValidLink && !isSuccess) {
    return (
      <div className="dve-page flex items-center justify-center p-4">
        <Card className="w-full max-w-md dve-card">
          <CardHeader className="text-center">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-red-500/10">
              <Lock className="h-8 w-8 text-red-400" />
            </div>
            <CardTitle className="text-white">Invalid Reset Link</CardTitle>
            <CardDescription className="text-white/70">
              This password reset link is invalid or has expired. Redirecting you home.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button
              onClick={() => router.push('/')}
              className="w-full bg-gradient-to-r from-knirv-primary to-knirv-secondary text-white hover:from-knirv-secondary hover:to-knirv-primary"
            >
              Go to Home
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (isSuccess) {
    return (
      <div className="dve-page flex items-center justify-center p-4">
        <Card className="w-full max-w-md dve-card">
          <CardHeader className="text-center">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-emerald-500/15">
              <Lock className="w-8 h-8 text-green-400" />
            </div>
            <CardTitle className="text-white">Password Updated!</CardTitle>
            <CardDescription className="text-white/70">
              Your password has been successfully updated. You will be redirected to the home page shortly.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button
              onClick={() => router.push('/')}
              className="w-full bg-gradient-to-r from-knirv-primary to-knirv-secondary text-white hover:from-knirv-secondary hover:to-knirv-primary"
            >
              Go to Home
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="dve-page flex items-center justify-center p-4">
      <Card className="w-full max-w-md dve-card">
        <CardHeader className="text-center">
          <CardTitle className="text-white">Reset Your Password</CardTitle>
          <CardDescription className="text-white/70">
            Enter your new password below
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleResetPassword} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="new-password" className="text-white">New Password</Label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-white/50" />
                <Input
                  id="new-password"
                  type={showPassword ? "text" : "password"}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Enter your new password"
                  className="bg-white/5 border-white/10 text-white pl-10 pr-10"
                  required
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="absolute right-0 top-0 h-full px-3 text-white/50 hover:bg-transparent hover:text-white"
                  onClick={() => setShowPassword(!showPassword)}
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </Button>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirm-password" className="text-white">Confirm Password</Label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-white/50" />
                <Input
                  id="confirm-password"
                  type={showConfirmPassword ? "text" : "password"}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder="Confirm your new password"
                  className="bg-white/5 border-white/10 text-white pl-10 pr-10"
                  required
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="absolute right-0 top-0 h-full px-3 text-white/50 hover:bg-transparent hover:text-white"
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                >
                  {showConfirmPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </Button>
              </div>
            </div>

            <Button
              type="submit"
              disabled={isLoading}
              className="w-full bg-gradient-to-r from-knirv-primary to-knirv-secondary text-white hover:from-knirv-secondary hover:to-knirv-primary disabled:opacity-50"
            >
              {isLoading ? "Updating Password..." : "Update Password"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
