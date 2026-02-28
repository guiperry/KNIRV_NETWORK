'use client';

import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Shield, Server, Cpu, HardDrive, Key, CheckCircle, Loader2 } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import { API_BASE_URL } from '@/lib/api';

interface DVECreationFormProps {
  onClose: () => void;
  onCreated: () => void;
}

export const DVECreationForm: React.FC<DVECreationFormProps> = ({ onClose, onCreated }) => {
  const { toast } = useToast();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [formData, setFormData] = useState({
    dveName: '',
    teeType: 'sgx',
    cpuCores: '4',
    memoryGB: '8',
    storageGB: '50',
    walletName: '',
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!formData.dveName.trim()) {
      toast({
        title: "Validation Error",
        description: "Please enter a DVE name",
        variant: "destructive",
      });
      return;
    }

    setIsSubmitting(true);

    try {
      const token = localStorage.getItem('knirv_nexus_token');
      const response = await fetch(`${API_BASE_URL}/api/dve/create`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          name: formData.dveName,
          tee_type: formData.teeType,
          cpu_cores: parseInt(formData.cpuCores),
          memory_gb: parseInt(formData.memoryGB),
          storage_gb: parseInt(formData.storageGB),
          wallet_name: formData.walletName || formData.dveName,
        }),
      });

      if (response.ok) {
        const data = await response.json();
        toast({
          title: "DVE Created",
          description: `DVE instance "${formData.dveName}" has been created successfully.`,
        });
        onCreated();
      } else {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to create DVE');
      }
    } catch (error) {
      console.error('DVE creation error:', error);
      toast({
        title: "Creation Failed",
        description: error instanceof Error ? error.message : "Failed to create DVE instance. Please try again.",
        variant: "destructive",
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* DVE Name */}
      <div className="space-y-2">
        <Label htmlFor="dveName" className="text-slate-300">DVE Instance Name *</Label>
        <Input
          id="dveName"
          type="text"
          placeholder="e.g., ALPHA-STRATEGIC-01"
          value={formData.dveName}
          onChange={(e) => setFormData(prev => ({ ...prev, dveName: e.target.value.toUpperCase() }))}
          disabled={isSubmitting}
          className="bg-slate-800 border-slate-600 text-slate-100 placeholder:text-slate-500"
        />
      </div>

      {/* TEE Type */}
      <div className="space-y-2">
        <Label className="text-slate-300">Trusted Execution Environment</Label>
        <Select 
          value={formData.teeType} 
          onValueChange={(value) => setFormData(prev => ({ ...prev, teeType: value }))}
          disabled={isSubmitting}
        >
          <SelectTrigger className="bg-slate-800 border-slate-600 text-slate-100">
            <SelectValue />
          </SelectTrigger>
          <SelectContent className="bg-slate-800 border-slate-600">
            <SelectItem value="sgx" className="text-slate-100">
              <div className="flex items-center gap-2">
                <Shield className="w-4 h-4 text-blue-500" />
                <span>Intel SGX</span>
              </div>
            </SelectItem>
            <SelectItem value="sev-snp" className="text-slate-100">
              <div className="flex items-center gap-2">
                <Shield className="w-4 h-4 text-green-500" />
                <span>AMD SEV-SNP</span>
              </div>
            </SelectItem>
            <SelectItem value="tdx" className="text-slate-100">
              <div className="flex items-center gap-2">
                <Shield className="w-4 h-4 text-purple-500" />
                <span>Intel TDX</span>
              </div>
            </SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Resource Allocation */}
      <div className="grid grid-cols-3 gap-4">
        <div className="space-y-2">
          <Label className="text-slate-300 flex items-center gap-1">
            <Cpu className="w-3 h-3" /> CPU Cores
          </Label>
          <Select 
            value={formData.cpuCores} 
            onValueChange={(value) => setFormData(prev => ({ ...prev, cpuCores: value }))}
            disabled={isSubmitting}
          >
            <SelectTrigger className="bg-slate-800 border-slate-600 text-slate-100">
              <SelectValue />
            </SelectTrigger>
            <SelectContent className="bg-slate-800 border-slate-600">
              <SelectItem value="2" className="text-slate-100">2 Cores</SelectItem>
              <SelectItem value="4" className="text-slate-100">4 Cores</SelectItem>
              <SelectItem value="8" className="text-slate-100">8 Cores</SelectItem>
              <SelectItem value="16" className="text-slate-100">16 Cores</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label className="text-slate-300 flex items-center gap-1">
            <HardDrive className="w-3 h-3" /> Memory
          </Label>
          <Select 
            value={formData.memoryGB} 
            onValueChange={(value) => setFormData(prev => ({ ...prev, memoryGB: value }))}
            disabled={isSubmitting}
          >
            <SelectTrigger className="bg-slate-800 border-slate-600 text-slate-100">
              <SelectValue />
            </SelectTrigger>
            <SelectContent className="bg-slate-800 border-slate-600">
              <SelectItem value="4" className="text-slate-100">4 GB</SelectItem>
              <SelectItem value="8" className="text-slate-100">8 GB</SelectItem>
              <SelectItem value="16" className="text-slate-100">16 GB</SelectItem>
              <SelectItem value="32" className="text-slate-100">32 GB</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label className="text-slate-300 flex items-center gap-1">
            <Server className="w-3 h-3" /> Storage
          </Label>
          <Select 
            value={formData.storageGB} 
            onValueChange={(value) => setFormData(prev => ({ ...prev, storageGB: value }))}
            disabled={isSubmitting}
          >
            <SelectTrigger className="bg-slate-800 border-slate-600 text-slate-100">
              <SelectValue />
            </SelectTrigger>
            <SelectContent className="bg-slate-800 border-slate-600">
              <SelectItem value="20" className="text-slate-100">20 GB</SelectItem>
              <SelectItem value="50" className="text-slate-100">50 GB</SelectItem>
              <SelectItem value="100" className="text-slate-100">100 GB</SelectItem>
              <SelectItem value="200" className="text-slate-100">200 GB</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Wallet Name */}
      <div className="space-y-2">
        <Label htmlFor="walletName" className="text-slate-300 flex items-center gap-1">
          <Key className="w-3 h-3" /> Data Fabric Wallet Name
        </Label>
        <Input
          id="walletName"
          type="text"
          placeholder="Same as DVE name if empty"
          value={formData.walletName}
          onChange={(e) => setFormData(prev => ({ ...prev, walletName: e.target.value.toUpperCase() }))}
          disabled={isSubmitting}
          className="bg-slate-800 border-slate-600 text-slate-100 placeholder:text-slate-500"
        />
        <p className="text-xs text-slate-500">
          This will be used as the identifier for the DVE's data fabric
        </p>
      </div>

      {/* Actions */}
      <div className="flex justify-end gap-3 pt-4">
        <Button 
          type="button" 
          variant="outline" 
          onClick={onClose}
          disabled={isSubmitting}
          className="border-slate-600 text-slate-300 hover:bg-slate-800"
        >
          Cancel
        </Button>
        <Button 
          type="submit"
          disabled={isSubmitting || !formData.dveName.trim()}
          className="bg-blue-600 hover:bg-blue-700 text-white"
        >
          {isSubmitting ? (
            <>
              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              Creating...
            </>
          ) : (
            <>
              <CheckCircle className="w-4 h-4 mr-2" />
              Create DVE
            </>
          )}
        </Button>
      </div>
    </form>
  );
};

export default DVECreationForm;
