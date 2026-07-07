"use client";

import React, { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { useToast } from '@/hooks/use-toast';
import { 
  Globe, 
  Plus, 
  Edit, 
  Trash2, 
  RefreshCw, 
  Server, 
  Activity,
  CheckCircle,
  AlertTriangle,
  Clock,
  Eye,
  Settings
} from 'lucide-react';
import { useDNSManagement } from '@/hooks/use-dns-management';
import type { DNSRecord, CreateDNSRecordRequest } from '@/types/api';

interface DNSManagementProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function DNSManagement({ isOpen, onClose }: DNSManagementProps) {
  const { toast } = useToast();
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [editingRecord, setEditingRecord] = useState<DNSRecord | null>(null);
  const [selectedZone, setSelectedZone] = useState<string>('all');
  const [selectedType, setSelectedType] = useState<string>('all');

  const {
    records,
    zones,
    status,
    isLoading,
    error,
    createRecord,
    updateRecord,
    deleteRecord,
    refreshAll,
    getRecordsByZone,
    getRecordsByType,
    getRecordTypesSummary
  } = useDNSManagement();

  // Form state for creating/editing records
  const [formData, setFormData] = useState<CreateDNSRecordRequest>({
    name: '',
    type: 'A',
    value: '',
    ttl: 300,
    zone: '',
    proxied: false,
    priority: 0,
    comment: ''
  });

  const handleCreateRecord = async () => {
    if (!formData.name || !formData.value) {
      toast({
        title: "Validation Error",
        description: "Name and value are required",
        variant: "destructive",
      });
      return;
    }

    const result = await createRecord(formData);
    if (result) {
      toast({
        title: "DNS Record Created",
        description: `Successfully created ${formData.type} record for ${formData.name}`,
      });
      setShowCreateForm(false);
      setFormData({
        name: '',
        type: 'A',
        value: '',
        ttl: 300,
        zone: '',
        proxied: false,
        priority: 0,
        comment: ''
      });
    }
  };

  const handleDeleteRecord = async (recordId: string, recordName: string) => {
    const confirmed = window.confirm(`Are you sure you want to delete the DNS record for ${recordName}?`);
    if (!confirmed) return;

    const success = await deleteRecord(recordId);
    if (success) {
      toast({
        title: "DNS Record Deleted",
        description: `Successfully deleted record for ${recordName}`,
      });
    }
  };

  const getStatusBadge = (serviceStatus: string) => {
    switch (serviceStatus) {
      case "running":
        return <Badge className="bg-green-500"><CheckCircle className="w-3 h-3 mr-1" /> Running</Badge>;
      case "stopped":
        return <Badge className="bg-red-500"><AlertTriangle className="w-3 h-3 mr-1" /> Stopped</Badge>;
      case "error":
        return <Badge className="bg-red-500"><AlertTriangle className="w-3 h-3 mr-1" /> Error</Badge>;
      default:
        return <Badge className="bg-yellow-500"><Clock className="w-3 h-3 mr-1" /> Unknown</Badge>;
    }
  };

  const getRecordTypeBadge = (type: string) => {
    const colors: Record<string, string> = {
      'A': 'bg-blue-500',
      'AAAA': 'bg-purple-500',
      'CNAME': 'bg-green-500',
      'MX': 'bg-orange-500',
      'TXT': 'bg-gray-500',
      'NS': 'bg-indigo-500',
      'SRV': 'bg-pink-500'
    };
    return <Badge className={colors[type] || 'bg-gray-500'}>{type}</Badge>;
  };

  const filteredRecords = records.filter(record => {
    const zoneMatch = selectedZone === 'all' || record.zone === selectedZone;
    const typeMatch = selectedType === 'all' || record.type === selectedType;
    return zoneMatch && typeMatch;
  });

  const recordTypesSummary = getRecordTypesSummary() || {};

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-8 pt-16 pb-16">
      <div className="w-full max-w-6xl max-h-[90vh] bg-background border shadow-2xl rounded-lg overflow-hidden">
        <div className="flex flex-col h-full">
          {/* Header */}
          <div className="flex items-center justify-between p-6 border-b bg-gradient-to-r from-primary/10 to-secondary/10">
            <div className="flex items-center space-x-4">
              <div className="w-12 h-12 bg-gradient-to-r from-primary to-secondary rounded-lg flex items-center justify-center">
                <Globe className="w-6 h-6 text-white" />
              </div>
              <div>
                <h2 className="text-2xl font-bold">DNS Management</h2>
                <p className="text-muted-foreground">
                  Manage DNS records and zones for KNIRV infrastructure
                </p>
              </div>
            </div>
            <div className="flex items-center space-x-2">
              <Button variant="outline" size="sm" onClick={refreshAll} disabled={isLoading}>
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
                <TabsTrigger value="records">DNS Records</TabsTrigger>
                <TabsTrigger value="zones">DNS Zones</TabsTrigger>
                <TabsTrigger value="settings">Settings</TabsTrigger>
              </TabsList>

              <TabsContent value="overview" className="space-y-4">
                {/* Service Status */}
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Activity className="h-5 w-5" />
                      DNS Service Status
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    {status ? (
                      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                        <div>
                          <p className="text-sm text-muted-foreground">Service Status</p>
                          {getStatusBadge(status.status)}
                        </div>
                        <div>
                          <p className="text-sm text-muted-foreground">Total Zones</p>
                          <p className="text-2xl font-bold">{status.zones}</p>
                        </div>
                        <div>
                          <p className="text-sm text-muted-foreground">Total Records</p>
                          <p className="text-2xl font-bold">{status.records}</p>
                        </div>
                        <div>
                          <p className="text-sm text-muted-foreground">Current IP</p>
                          <p className="font-mono text-sm">{status.current_ip || 'N/A'}</p>
                        </div>
                      </div>
                    ) : (
                      <p className="text-muted-foreground">Loading DNS service status...</p>
                    )}
                  </CardContent>
                </Card>

                {/* Record Types Summary */}
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Server className="h-5 w-5" />
                      Record Types Summary
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
                      {Object.entries(recordTypesSummary).map(([type, count]) => (
                        <div key={type} className="text-center">
                          {getRecordTypeBadge(type)}
                          <p className="text-2xl font-bold mt-2">{count}</p>
                          <p className="text-xs text-muted-foreground">records</p>
                        </div>
                      ))}
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="records" className="space-y-4">
                {/* Filters and Actions */}
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-4">
                    <div>
                      <Label htmlFor="zone-filter">Zone</Label>
                      <Select value={selectedZone} onValueChange={setSelectedZone}>
                        <SelectTrigger className="w-40">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="all">All Zones</SelectItem>
                          {zones.map(zone => (
                            <SelectItem key={zone.id} value={zone.name}>{zone.name}</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <div>
                      <Label htmlFor="type-filter">Type</Label>
                      <Select value={selectedType} onValueChange={setSelectedType}>
                        <SelectTrigger className="w-32">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="all">All Types</SelectItem>
                          <SelectItem value="A">A</SelectItem>
                          <SelectItem value="AAAA">AAAA</SelectItem>
                          <SelectItem value="CNAME">CNAME</SelectItem>
                          <SelectItem value="MX">MX</SelectItem>
                          <SelectItem value="TXT">TXT</SelectItem>
                          <SelectItem value="NS">NS</SelectItem>
                          <SelectItem value="SRV">SRV</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                  <Button onClick={() => setShowCreateForm(true)}>
                    <Plus className="w-4 h-4 mr-2" />
                    Add Record
                  </Button>
                </div>

                {/* DNS Records Table */}
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle>DNS Records ({filteredRecords.length})</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-2">
                      {filteredRecords.map((record) => (
                        <div key={record.id} className="flex items-center justify-between p-3 border rounded-lg">
                          <div className="flex-1 grid grid-cols-1 md:grid-cols-4 gap-4">
                            <div>
                              <p className="font-medium">{record.name}</p>
                              <p className="text-sm text-muted-foreground">{record.zone}</p>
                            </div>
                            <div className="flex items-center gap-2">
                              {getRecordTypeBadge(record.type)}
                              {record.proxied && (
                                <Badge className="bg-green-500">Proxied</Badge>
                              )}
                            </div>
                            <div>
                              <p className="font-mono text-sm">{record.value}</p>
                              <p className="text-xs text-muted-foreground">TTL: {record.ttl}s</p>
                            </div>
                            <div className="flex items-center space-x-2">
                              <Button variant="outline" size="sm" onClick={() => setEditingRecord(record)}>
                                <Edit className="w-3 h-3" />
                              </Button>
                              <Button 
                                variant="outline" 
                                size="sm" 
                                onClick={() => handleDeleteRecord(record.id, record.name)}
                              >
                                <Trash2 className="w-3 h-3" />
                              </Button>
                            </div>
                          </div>
                        </div>
                      ))}
                      {filteredRecords.length === 0 && (
                        <div className="text-center py-8 text-muted-foreground">
                          No DNS records found matching the current filters.
                        </div>
                      )}
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="zones" className="space-y-4">
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle>DNS Zones</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-2">
                      {zones.map((zone) => (
                        <div key={zone.id} className="flex items-center justify-between p-3 border rounded-lg">
                          <div>
                            <p className="font-medium">{zone.name}</p>
                            <p className="text-sm text-muted-foreground">Type: {zone.type}</p>
                          </div>
                          <div className="flex items-center space-x-2">
                            <Badge variant="outline">{zone.type}</Badge>
                            <Button variant="outline" size="sm">
                              <Eye className="w-3 h-3 mr-1" />
                              View
                            </Button>
                          </div>
                        </div>
                      ))}
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="settings" className="space-y-4">
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Settings className="h-5 w-5" />
                      DNS Service Configuration
                    </CardTitle>
                    <CardDescription>
                      Configure DNS service settings and CloudFlare integration
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-4">
                      <div className="p-4 border rounded-lg bg-muted/50">
                        <p className="text-sm text-muted-foreground">
                          DNS service configuration is managed through the backend configuration files.
                          Contact your system administrator to modify DNS service settings.
                        </p>
                      </div>
                      {status && (
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                          <div>
                            <Label>Update Count</Label>
                            <p className="font-mono">{status.update_count || 0}</p>
                          </div>
                          <div>
                            <Label>Error Count</Label>
                            <p className="font-mono">{status.error_count || 0}</p>
                          </div>
                          <div>
                            <Label>Last Update</Label>
                            <p className="font-mono text-sm">{status.last_update || 'Never'}</p>
                          </div>
                          <div>
                            <Label>Service Timestamp</Label>
                            <p className="font-mono text-sm">{new Date(status.timestamp).toLocaleString()}</p>
                          </div>
                        </div>
                      )}
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>
            </Tabs>
          </div>
        </div>
      </div>

      {/* Create Record Modal would go here - simplified for now */}
      {showCreateForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-60 p-8">
          <Card className="w-full max-w-md">
            <CardHeader>
              <CardTitle>Create DNS Record</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label htmlFor="record-name">Name</Label>
                <Input
                  id="record-name"
                  value={formData.name}
                  onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                  placeholder="subdomain.example.com"
                />
              </div>
              <div>
                <Label htmlFor="record-type">Type</Label>
                <Select value={formData.type} onValueChange={(value) => setFormData(prev => ({ ...prev, type: value }))}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="A">A</SelectItem>
                    <SelectItem value="AAAA">AAAA</SelectItem>
                    <SelectItem value="CNAME">CNAME</SelectItem>
                    <SelectItem value="MX">MX</SelectItem>
                    <SelectItem value="TXT">TXT</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Label htmlFor="record-value">Value</Label>
                <Input
                  id="record-value"
                  value={formData.value}
                  onChange={(e) => setFormData(prev => ({ ...prev, value: e.target.value }))}
                  placeholder="192.168.1.1"
                />
              </div>
              <div className="flex space-x-2">
                <Button onClick={handleCreateRecord} disabled={isLoading}>
                  Create
                </Button>
                <Button variant="outline" onClick={() => setShowCreateForm(false)}>
                  Cancel
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
