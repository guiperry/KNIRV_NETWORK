'use client';

import React, { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import {
  useOnboardingAdmin,
  useRecordKycDecision,
  useReviewOperatorApplication,
  useUpdateUserRole,
  useUpdateUserStatus,
  type OnboardingApplication,
  type OnboardingUser,
} from '@/hooks/use-onboarding-admin';
import { Activity, UserCheck, UserX } from 'lucide-react';

const ROLE_OPTIONS = ['admin', 'validator', 'observer'] as const;
const STATUS_OPTIONS = ['active', 'suspended', 'banned'] as const;

export function NetworkMonitorOnboarding() {
  const { applications, users, isLoading, error } = useOnboardingAdmin();
  const reviewApplication = useReviewOperatorApplication();
  const recordKyc = useRecordKycDecision();
  const [kycNotes, setKycNotes] = useState<Record<string, string>>({});
  const updateRole = useUpdateUserRole();
  const updateStatus = useUpdateUserStatus();
  const [actionError, setActionError] = useState<string | null>(null);

  const runAction = async (action: () => Promise<unknown>) => {
    setActionError(null);
    try {
      await action();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Action failed');
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12 text-gray-500">
        <Activity className="w-5 h-5 mr-2 animate-spin" />
        Loading onboarding data...
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-amber-500/20 bg-amber-500/10 p-4 text-amber-300">
        Onboarding proxy endpoint not yet available. This will be populated when ONBOARDING.KNIRV.COM admin endpoints are proxied.
      </div>
    );
  }

  // 'reviewed' and 'rejected' are terminal states (see review.ts / the
  // KNIRVGATEWAY handleAdminOperatorReview it mirrors) — everything else
  // ('pending', 'validated', ...) is still awaiting a decision. Only
  // 'validated' applications can actually be approved; attempting to
  // approve a merely-'pending' one surfaces the 409 KNIRVGATEWAY returns via
  // actionError below rather than being silently filtered out here.
  const pendingApps: OnboardingApplication[] =
    applications?.data?.applications?.filter((a) => a.status !== 'reviewed' && a.status !== 'rejected') ?? [];
  const kycQueue: OnboardingApplication[] =
    applications?.data?.applications?.filter((a) => a.kyc_status === 'pending' || a.kyc_status === 'manual_review') ?? [];
  const allUsers: OnboardingUser[] = users?.data?.users ?? [];

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-semibold text-gray-200">Onboarding & User Management</h3>
        <p className="text-sm text-gray-500">
          Operator applications and user accounts
        </p>
      </div>

      {actionError && (
        <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-300">
          {actionError}
        </div>
      )}

      <Card className="aether-bevel-dark rounded-xl">
        <CardHeader className="pb-2">
          <CardTitle className="text-sm text-gray-300 flex items-center gap-2">
            <UserCheck className="w-4 h-4 text-cyan-400" />
            KYC / AML Review ({kycQueue.length})
          </CardTitle>
          <CardDescription className="text-xs text-gray-500">
            Verify the applicant (business registry, sanctions lists, contact) before approving. Approval lets the
            operator provision a bootnode. Your note is written to the gateway log.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {kycQueue.length === 0 ? (
            <p className="text-sm text-gray-500">No applications awaiting KYC.</p>
          ) : (
            <div className="space-y-2">
              {kycQueue.map((app) => {
                const busy = recordKyc.isPending && recordKyc.variables?.id === app.id;
                const decide = (status: 'approved' | 'rejected' | 'manual_review') =>
                  runAction(() => recordKyc.mutateAsync({ id: app.id, status, note: kycNotes[app.id] }));
                return (
                  <div key={app.id} className="rounded-lg border border-slate-800 bg-slate-950/40 p-3 space-y-2">
                    <div className="text-sm font-medium text-gray-200">
                      {app.legal_name}
                      {app.business_name && app.business_name !== app.legal_name ? ` (${app.business_name})` : ''}
                    </div>
                    <div className="text-xs text-gray-500 grid grid-cols-1 sm:grid-cols-2 gap-x-4">
                      <span>Tax ID: {app.tax_id || '-'}</span>
                      <span>Country: {app.country || '-'}</span>
                      <span>Site: {app.corporate_site || '-'}</span>
                      <span>Contact: {app.contact_email || '-'}</span>
                      <span>KYC: {app.kyc_status}</span>
                      <span>Applied: {new Date(app.created_at).toLocaleDateString()}</span>
                    </div>
                    <input
                      type="text"
                      value={kycNotes[app.id] ?? ''}
                      onChange={(e) => setKycNotes((prev) => ({ ...prev, [app.id]: e.target.value }))}
                      placeholder="Review note (what was checked)"
                      className="w-full rounded border border-slate-800 bg-slate-950 px-2 py-1 text-xs text-gray-300"
                    />
                    <div className="flex gap-2">
                      <Button size="sm" variant="outline" disabled={busy}
                        className="border-green-700/50 text-green-400 hover:bg-green-600/20"
                        onClick={() => decide('approved')}>Approve KYC</Button>
                      <Button size="sm" variant="outline" disabled={busy}
                        className="border-amber-700/50 text-amber-400 hover:bg-amber-600/20"
                        onClick={() => decide('manual_review')}>Needs more info</Button>
                      <Button size="sm" variant="outline" disabled={busy}
                        className="border-red-700/50 text-red-400 hover:bg-red-600/20"
                        onClick={() => decide('rejected')}>Reject</Button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      <Card className="aether-bevel-dark rounded-xl">
        <CardHeader className="pb-2">
          <CardTitle className="text-sm text-gray-300 flex items-center gap-2">
            <UserCheck className="w-4 h-4 text-amber-400" />
            Pending Operator Applications ({pendingApps.length})
          </CardTitle>
          <CardDescription className="text-xs text-gray-500">
            Review and approve new operator registrations
          </CardDescription>
        </CardHeader>
        <CardContent>
          {pendingApps.length === 0 ? (
            <p className="text-sm text-gray-500">No pending applications.</p>
          ) : (
            <div className="space-y-2">
              {pendingApps.map((app) => {
                const isPending = reviewApplication.isPending && reviewApplication.variables?.id === app.id;
                return (
                  <div
                    key={app.id}
                    className="flex items-center justify-between rounded-lg border border-slate-800 bg-slate-950/40 p-3"
                  >
                    <div>
                      <div className="text-sm font-medium text-gray-200">{app.legal_name}</div>
                      <div className="text-xs text-gray-500">
                        Status: {app.status} | KYC: {app.kyc_status} | Applied: {new Date(app.created_at).toLocaleDateString()}
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        disabled={isPending}
                        className="border-green-700/50 text-green-400 hover:bg-green-600/20"
                        onClick={() => runAction(() => reviewApplication.mutateAsync({ id: app.id, decision: 'approved' }))}
                      >
                        Approve
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        disabled={isPending}
                        className="border-red-700/50 text-red-400 hover:bg-red-600/20"
                        onClick={() => runAction(() => reviewApplication.mutateAsync({ id: app.id, decision: 'rejected' }))}
                      >
                        Reject
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      <Card className="aether-bevel-dark rounded-xl">
        <CardHeader className="pb-2">
          <CardTitle className="text-sm text-gray-300 flex items-center gap-2">
            <UserX className="w-4 h-4 text-blue-400" />
            User Accounts ({allUsers.length})
          </CardTitle>
          <CardDescription className="text-xs text-gray-500">
            Registered accounts and their roles
          </CardDescription>
        </CardHeader>
        <CardContent>
          {allUsers.length === 0 ? (
            <p className="text-sm text-gray-500">No users found.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-800 text-left text-gray-500">
                    <th className="pb-2 font-medium">Username</th>
                    <th className="pb-2 font-medium">Email</th>
                    <th className="pb-2 font-medium">Role</th>
                    <th className="pb-2 font-medium">Status</th>
                    <th className="pb-2 font-medium">Plan</th>
                  </tr>
                </thead>
                <tbody>
                  {allUsers.map((user) => {
                    const roleUpdating = updateRole.isPending && updateRole.variables?.id === user.id;
                    const statusUpdating = updateStatus.isPending && updateStatus.variables?.id === user.id;
                    return (
                      <tr key={user.id} className="border-b border-gray-800/50">
                        <td className="py-2 text-gray-300">{user.username}</td>
                        <td className="py-2 text-gray-400">{user.email}</td>
                        <td className="py-2">
                          <Select
                            value={user.role}
                            disabled={roleUpdating}
                            onValueChange={(role) =>
                              runAction(() => updateRole.mutateAsync({ id: user.id, role: role as 'admin' | 'validator' | 'observer' }))
                            }
                          >
                            <SelectTrigger className="h-7 w-28 text-xs">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              {ROLE_OPTIONS.map((role) => (
                                <SelectItem key={role} value={role}>
                                  {role}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        </td>
                        <td className="py-2">
                          <Select
                            value={user.account_status}
                            disabled={statusUpdating}
                            onValueChange={(status) =>
                              runAction(() => updateStatus.mutateAsync({ id: user.id, status: status as 'active' | 'suspended' | 'banned' }))
                            }
                          >
                            <SelectTrigger
                              className={`h-7 w-28 text-xs ${
                                user.account_status === 'active'
                                  ? 'text-green-300'
                                  : user.account_status === 'suspended'
                                    ? 'text-red-300'
                                    : 'text-gray-300'
                              }`}
                            >
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              {STATUS_OPTIONS.map((status) => (
                                <SelectItem key={status} value={status}>
                                  {status}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        </td>
                        <td className="py-2 text-gray-400">{user.plan}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
