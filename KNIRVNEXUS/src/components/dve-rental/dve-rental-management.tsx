"use client";

import React, { useState } from 'react';
import { createPortal } from 'react-dom';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { useToast } from '@/hooks/use-toast';
import { 
  CreditCard, 
  Server, 
  Clock, 
  DollarSign, 
  RefreshCw, 
  Play,
  Square,
  Trash2,
  Eye,
  Settings,
  BarChart3,
  Zap,
  Cpu,
  HardDrive,
  Network,
  CheckCircle,
  AlertTriangle,
  Timer
} from 'lucide-react';
import { useDVERental } from '@/hooks/use-dve-rental';
import type { DVERentalPlan, DVERental, RentalRequest } from '@/types/api';
import { useAuth } from '@/lib/auth-context';

interface DVERentalManagementProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function DVERentalManagement({ isOpen, onClose }: DVERentalManagementProps) {
  const { toast } = useToast();
  const { user } = useAuth();
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [selectedPlan, setSelectedPlan] = useState<DVERentalPlan | null>(null);
  const [paymentTxHash, setPaymentTxHash] = useState('');
  const [rentalDuration, setRentalDuration] = useState(1);

  // Debug logging
  React.useEffect(() => {
    console.log('[DVE Rental Modal] isOpen changed:', isOpen);
  }, [isOpen]);

  const {
    plans,
    rentals,
    stats,
    isLoading,
    error,
    createRental,
    extendRental,
    cancelRental,
    getActiveRentals,
    getTotalCost,
    fetchRentals,
    fetchPlans,
    fetchStats
  } = useDVERental();

  const handleCreateRental = async () => {
    if (!selectedPlan || !paymentTxHash.trim()) {
      toast({
        title: "Validation Error",
        description: "Please select a plan and provide payment transaction hash",
        variant: "destructive",
      });
      return;
    }

    const rentalRequest: RentalRequest = {
      plan_id: selectedPlan.id,
      duration_hours: rentalDuration,
      payment_tx_hash: paymentTxHash.trim(),
      user_id: user?.user || 'anonymous'
    };

    const result = await createRental(rentalRequest);
    if (result) {
      toast({
        title: "DVE Rental Created",
        description: `Successfully rented ${selectedPlan.name} for ${rentalDuration} hours`,
      });
      setShowCreateForm(false);
      setSelectedPlan(null);
      setPaymentTxHash('');
      setRentalDuration(1);
    }
  };

  const handleExtendRental = async (rentalId: string, additionalHours: number, txHash: string) => {
    const success = await extendRental(rentalId, additionalHours, txHash);
    if (success) {
      toast({
        title: "Rental Extended",
        description: `Successfully extended rental by ${additionalHours} hours`,
      });
    }
  };

  const handleCancelRental = async (rentalId: string) => {
    const confirmed = window.confirm('Are you sure you want to cancel this rental?');
    if (!confirmed) return;

    const success = await cancelRental(rentalId);
    if (success) {
      toast({
        title: "Rental Cancelled",
        description: "Rental has been cancelled successfully",
      });
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "active":
        return <Badge className="bg-green-500"><CheckCircle className="w-3 h-3 mr-1" /> Active</Badge>;
      case "expired":
        return <Badge className="bg-red-500"><AlertTriangle className="w-3 h-3 mr-1" /> Expired</Badge>;
      case "cancelled":
        return <Badge className="bg-gray-500"><Square className="w-3 h-3 mr-1" /> Cancelled</Badge>;
      default:
        return <Badge className="bg-yellow-500"><Timer className="w-3 h-3 mr-1" /> {status}</Badge>;
    }
  };

  const formatCurrency = (amount: number | undefined | null) => {
    if (amount == null) return '0 NRN';
    return `${Math.round(amount).toLocaleString()} NRN`;
  };

  const formatDuration = (hours: number | undefined | null) => {
    if (hours == null || hours <= 0) return '--';
    if (hours < 24) return `${hours}h`;
    const days = Math.floor(hours / 24);
    const remainingHours = hours % 24;
    return remainingHours > 0 ? `${days}d ${remainingHours}h` : `${days}d`;
  };

  const formatDate = (dateString: string | undefined | null) => {
    if (!dateString) return '--';
    try {
      return new Date(dateString).toLocaleDateString();
    } catch {
      return '--';
    }
  };

  const calculateTotalCost = (plan: DVERentalPlan | null | undefined, duration: number) => {
    if (!plan || plan.price_per_hour == null) return 0;
    return plan.price_per_hour * duration;
  };

  const activeRentals = getActiveRentals() || [];

  if (!isOpen) return null;

  const modalContent = (
    <div 
      className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-[9999] p-8 pt-16 pb-16"
      onClick={(e) => {
        // Close modal when clicking backdrop
        if (e.target === e.currentTarget) {
          onClose();
        }
      }}
      style={{ 
        position: 'fixed', 
        top: 0, 
        left: 0, 
        right: 0, 
        bottom: 0,
        pointerEvents: 'auto',
        zIndex: 9999
      }}
    >
      <div 
        className="w-full max-w-7xl max-h-[90vh] bg-background border shadow-2xl rounded-lg overflow-hidden"
        onClick={(e) => e.stopPropagation()}
        style={{ pointerEvents: 'auto' }}
      >
        <div className="flex flex-col h-full">
          {/* Header */}
          <div className="flex items-center justify-between p-6 border-b bg-gradient-to-r from-primary/10 to-secondary/10">
            <div className="flex items-center space-x-4">
              <div className="w-12 h-12 bg-gradient-to-r from-primary to-secondary rounded-lg flex items-center justify-center">
                <CreditCard className="w-6 h-6 text-white" />
              </div>
              <div>
                <h2 className="text-2xl font-bold">DVE Rental Management</h2>
                <p className="text-muted-foreground">
                  Rent DVE nodes with NRN tokens and manage your cloud environments
                </p>
              </div>
            </div>
            <div className="flex items-center space-x-2">
              <Button variant="outline" size="sm" onClick={() => { fetchPlans(); fetchRentals(); fetchStats(); }} disabled={isLoading}>
                <RefreshCw className={`w-4 h-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
                Refresh
              </Button>
              <Button variant="ghost" size="sm" onClick={onClose}>
                ×
              </Button>
            </div>
          </div>

          {/* Content */}
          <div className="flex-1 overflow-auto p-6">
            {error && (
              <div className="mb-4 p-3 bg-destructive/10 border border-destructive/20 rounded-lg">
                <p className="text-sm text-destructive">{error}</p>
              </div>
            )}

            <Tabs defaultValue="overview" className="space-y-4">
              <TabsList className="grid w-full grid-cols-4">
                <TabsTrigger value="overview">Overview</TabsTrigger>
                <TabsTrigger value="plans">Rental Plans</TabsTrigger>
                <TabsTrigger value="rentals">My Rentals</TabsTrigger>
                <TabsTrigger value="analytics">Analytics</TabsTrigger>
              </TabsList>

              <TabsContent value="overview" className="space-y-4">
                {/* Rental Statistics */}
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <BarChart3 className="h-5 w-5" />
                      Rental Overview
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                      <div className="text-center">
                        <p className="text-2xl font-bold text-green-500">{activeRentals.length}</p>
                        <p className="text-sm text-muted-foreground">Active Rentals</p>
                      </div>
                      <div className="text-center">
                        <p className="text-2xl font-bold text-blue-500">{rentals.length}</p>
                        <p className="text-sm text-muted-foreground">Total Rentals</p>
                      </div>
                      <div className="text-center">
                        <p className="text-2xl font-bold text-purple-500">{formatCurrency(getTotalCost())}</p>
                        <p className="text-sm text-muted-foreground">Total Spent</p>
                      </div>
                      <div className="text-center">
                        <p className="text-2xl font-bold text-orange-500">{plans.length}</p>
                        <p className="text-sm text-muted-foreground">Available Plans</p>
                      </div>
                    </div>
                  </CardContent>
                </Card>

                {/* Quick Actions */}
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle>Quick Actions</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                      <Button 
                        className="h-20 flex flex-col items-center justify-center"
                        onClick={() => setShowCreateForm(true)}
                      >
                        <Play className="w-6 h-6 mb-2" />
                        Rent DVE Node
                      </Button>
                      <Button variant="outline" className="h-20 flex flex-col items-center justify-center">
                        <Settings className="w-6 h-6 mb-2" />
                        Manage CDEs
                      </Button>
                      <Button variant="outline" className="h-20 flex flex-col items-center justify-center">
                        <BarChart3 className="w-6 h-6 mb-2" />
                        Usage Analytics
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="plans" className="space-y-4">
                {/* Rental Plans */}
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {plans.map((plan) => (
                    <Card key={plan.id} className="knirv-card-gradient hover:shadow-lg transition-shadow">
                      <CardHeader>
                        <CardTitle className="flex items-center justify-between">
                          {plan.name}
                          <Badge variant="outline">{formatCurrency(plan.price_per_hour)}/hr</Badge>
                        </CardTitle>
                        <CardDescription>{plan.description}</CardDescription>
                      </CardHeader>
                      <CardContent>
                        <div className="space-y-3">
                          <div className="flex items-center gap-2">
                            <Cpu className="w-4 h-4" />
                            <span className="text-sm">{plan.cpu_cores} CPU Cores</span>
                          </div>
                          <div className="flex items-center gap-2">
                            <HardDrive className="w-4 h-4" />
                            <span className="text-sm">{plan.memory_gb}GB RAM, {plan.disk_gb}GB Storage</span>
                          </div>
                          <div className="flex items-center gap-2">
                            <Network className="w-4 h-4" />
                            <span className="text-sm">{plan.bandwidth_mbps} Mbps Bandwidth</span>
                          </div>
                          <div className="pt-2">
                            <p className="text-xs text-muted-foreground mb-2">Features:</p>
                            <div className="flex flex-wrap gap-1">
                              {plan.features.map((feature, index) => (
                                <Badge key={index} variant="secondary" className="text-xs">
                                  {feature}
                                </Badge>
                              ))}
                            </div>
                          </div>
                          <Button 
                            className="w-full mt-4"
                            onClick={() => {
                              setSelectedPlan(plan);
                              setShowCreateForm(true);
                            }}
                          >
                            Rent This Plan
                          </Button>
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </TabsContent>

              <TabsContent value="rentals" className="space-y-4">
                {/* My Rentals */}
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle>My Rentals ({rentals.length})</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-3">
                      {rentals.map((rental) => (
                        <div key={rental.id} className="flex items-center justify-between p-4 border rounded-lg hover:bg-muted/50">
                          <div className="flex-1 grid grid-cols-1 md:grid-cols-4 gap-4">
                            <div>
                              <p className="font-medium">Rental #{rental.id.slice(-8)}</p>
                              <p className="text-sm text-muted-foreground">Node: {rental.node_id}</p>
                            </div>
                            <div>
                              {getStatusBadge(rental.status)}
                              <p className="text-sm text-muted-foreground mt-1">
                                {formatDuration(rental.duration_hours)}
                              </p>
                            </div>
                            <div>
                              <p className="font-medium">{formatCurrency(rental.total_cost)}</p>
                              <p className="text-sm text-muted-foreground">
                                {formatDate(rental.start_time)}
                              </p>
                            </div>
                            <div className="flex items-center space-x-2">
                              {rental.status === 'active' && (
                                <>
                                  <Button variant="outline" size="sm">
                                    <Clock className="w-3 h-3 mr-1" />
                                    Extend
                                  </Button>
                                  <Button variant="outline" size="sm">
                                    <Eye className="w-3 h-3 mr-1" />
                                    Access CDE
                                  </Button>
                                </>
                              )}
                              <Button 
                                variant="outline" 
                                size="sm" 
                                onClick={() => handleCancelRental(rental.id)}
                                disabled={rental.status !== 'active'}
                              >
                                <Trash2 className="w-3 h-3" />
                              </Button>
                            </div>
                          </div>
                        </div>
                      ))}
                      {rentals.length === 0 && (
                        <div className="text-center py-8 text-muted-foreground">
                          No rentals found. Start by renting a DVE node from the Plans tab.
                        </div>
                      )}
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="analytics" className="space-y-4">
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <BarChart3 className="h-5 w-5" />
                      Rental Analytics
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    {stats ? (
                      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                        <div className="text-center">
                          <p className="text-2xl font-bold">{stats.total_rentals}</p>
                          <p className="text-sm text-muted-foreground">Total Rentals</p>
                        </div>
                        <div className="text-center">
                          <p className="text-2xl font-bold">{stats.active_rentals}</p>
                          <p className="text-sm text-muted-foreground">Active Rentals</p>
                        </div>
                        <div className="text-center">
                          <p className="text-2xl font-bold">{formatCurrency(stats.total_revenue)}</p>
                          <p className="text-sm text-muted-foreground">Total Revenue</p>
                        </div>
                        <div className="text-center">
                          <p className="text-2xl font-bold">{formatDuration(stats.average_duration)}</p>
                          <p className="text-sm text-muted-foreground">Avg Duration</p>
                        </div>
                      </div>
                    ) : (
                      <p className="text-muted-foreground">Loading analytics...</p>
                    )}
                  </CardContent>
                </Card>
              </TabsContent>
            </Tabs>
          </div>
        </div>
      </div>

      {/* Create Rental Modal */}
      {showCreateForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-60">
          <Card className="w-full max-w-md">
            <CardHeader>
              <CardTitle>Rent DVE Node</CardTitle>
              <CardDescription>
                {selectedPlan ? `Renting: ${selectedPlan.name}` : 'Select a plan to rent'}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {!selectedPlan && (
                <div>
                  <Label htmlFor="plan-select">Select Plan</Label>
                  <Select onValueChange={(value) => {
                    const plan = plans.find(p => p.id === value);
                    setSelectedPlan(plan || null);
                  }}>
                    <SelectTrigger>
                      <SelectValue placeholder="Choose a rental plan" />
                    </SelectTrigger>
                    <SelectContent>
                      {plans.map(plan => (
                        <SelectItem key={plan.id} value={plan.id}>
                          {plan.name} - {formatCurrency(plan.price_per_hour)}/hr
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}
              
              <div>
                <Label htmlFor="duration">Duration (hours)</Label>
                <Input
                  id="duration"
                  type="number"
                  min="1"
                  max="168"
                  value={rentalDuration}
                  onChange={(e) => setRentalDuration(parseInt(e.target.value) || 1)}
                />
              </div>

              <div>
                <Label htmlFor="payment-hash">NRN Payment Transaction Hash</Label>
                <Input
                  id="payment-hash"
                  value={paymentTxHash}
                  onChange={(e) => setPaymentTxHash(e.target.value)}
                  placeholder="0x..."
                />
              </div>

              {selectedPlan && (
                <div className="p-3 bg-muted rounded-lg">
                  <p className="text-sm font-medium">Total Cost: {formatCurrency(calculateTotalCost(selectedPlan, rentalDuration))}</p>
                  <p className="text-xs text-muted-foreground">
                    {selectedPlan.cpu_cores} cores, {selectedPlan.memory_gb}GB RAM, {selectedPlan.disk_gb}GB storage
                  </p>
                </div>
              )}

              <div className="flex space-x-2">
                <Button onClick={handleCreateRental} disabled={isLoading || !selectedPlan}>
                  {isLoading ? 'Creating...' : 'Create Rental'}
                </Button>
                <Button variant="outline" onClick={() => {
                  setShowCreateForm(false);
                  setSelectedPlan(null);
                  setPaymentTxHash('');
                  setRentalDuration(1);
                }}>
                  Cancel
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );

  // Use portal to render modal at document root, avoiding z-index issues with nested tabs
  // Ensure we're in browser environment before using createPortal
  if (typeof window === 'undefined') return null;
  
  return createPortal(modalContent, document.body);
}
