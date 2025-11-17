"use client";

import { useEffect, useState, useCallback } from 'react';
import { webSocketService } from '@/lib/websocket-service';

interface DVENodeUpdate {
  id: string;
  status: "online" | "offline" | "maintenance";
  cpu_usage: number;
  memory_usage: number;
  last_heartbeat: string;
}

interface ValidationTaskUpdate {
  id: string;
  status: "pending" | "running" | "completed" | "failed";
  progress: number;
  assigned_node: string;
  estimated_completion?: string;
}

interface CognitiveEngineUpdate {
  status: "active" | "idle" | "learning" | "error";
  accuracy: number;
  tasks_processed: number;
  adaptation_rate: number;
}

interface TEESecurityUpdate {
  attestation_status: "verified" | "pending" | "failed";
  security_score: number;
  threats_detected: number;
  last_audit: string;
}

interface NRNStakingUpdate {
  total_staked: number;
  apy: number;
  rewards_24h: number;
  slashing_events: number;
}

interface SecurityAlert {
  type: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  message: string;
  timestamp: string;
}

interface SystemNotification {
  type: 'info' | 'warning' | 'error' | 'success';
  title: string;
  message: string;
  timestamp: string;
}

export const useKnirvSocket = () => {
  const [isConnected, setIsConnected] = useState(false);
  const [dveNodeUpdates, setDveNodeUpdates] = useState<DVENodeUpdate[]>([]);
  const [validationTaskUpdates, setValidationTaskUpdates] = useState<ValidationTaskUpdate[]>([]);
  const [cognitiveEngineUpdates, setCognitiveEngineUpdates] = useState<CognitiveEngineUpdate[]>([]);
  const [teeSecurityUpdates, setTeeSecurityUpdates] = useState<TEESecurityUpdate[]>([]);
  const [nrnStakingUpdates, setNrnStakingUpdates] = useState<NRNStakingUpdate[]>([]);
  const [securityAlerts, setSecurityAlerts] = useState<SecurityAlert[]>([]);
  const [systemNotifications, setSystemNotifications] = useState<SystemNotification[]>([]);

  useEffect(() => {
    // Set initial connection status
    setIsConnected(webSocketService.getConnectionStatus());

    // Connection status handler
    const handleConnection = (data: { connected: boolean }) => {
      setIsConnected(data.connected);
      if (data.connected) {
        console.log('Connected to KNIRV-NEXUS DVE WebSocket');
        // Request initial sync
        webSocketService.send({
          type: 'request-sync',
          payload: { timestamp: new Date().toISOString() }
        });
      } else {
        console.log('Disconnected from KNIRV-NEXUS DVE WebSocket');
      }
    };

    // Event handlers
    const handleDVENodeUpdate = (nodeUpdate: DVENodeUpdate) => {
      setDveNodeUpdates(prev => {
        const existing = prev.find(u => u.id === nodeUpdate.id);
        if (existing) {
          return prev.map(u => u.id === nodeUpdate.id ? nodeUpdate : u);
        }
        return [...prev, nodeUpdate];
      });
    };

    const handleValidationTaskUpdate = (taskUpdate: ValidationTaskUpdate) => {
      setValidationTaskUpdates(prev => {
        const existing = prev.find(u => u.id === taskUpdate.id);
        if (existing) {
          return prev.map(u => u.id === taskUpdate.id ? taskUpdate : u);
        }
        return [...prev, taskUpdate];
      });
    };

    const handleCognitiveEngineUpdate = (cognitiveUpdate: CognitiveEngineUpdate) => {
      setCognitiveEngineUpdates(prev => {
        const existing = prev.find(u => u.status === cognitiveUpdate.status);
        if (existing) {
          return prev.map(u => u.status === cognitiveUpdate.status ? cognitiveUpdate : u);
        }
        return [...prev, cognitiveUpdate];
      });
    };

    const handleTEESecurityUpdate = (teeUpdate: TEESecurityUpdate) => {
      setTeeSecurityUpdates(prev => {
        const existing = prev.find(u => u.attestation_status === teeUpdate.attestation_status);
        if (existing) {
          return prev.map(u => u.attestation_status === teeUpdate.attestation_status ? teeUpdate : u);
        }
        return [...prev, teeUpdate];
      });
    };

    const handleNRNStakingUpdate = (stakingUpdate: NRNStakingUpdate) => {
      setNrnStakingUpdates(prev => {
        const existing = prev.find(u => u.total_staked === stakingUpdate.total_staked);
        if (existing) {
          return prev.map(u => u.total_staked === stakingUpdate.total_staked ? stakingUpdate : u);
        }
        return [...prev, stakingUpdate];
      });
    };

    const handleSecurityAlert = (alert: SecurityAlert) => {
      setSecurityAlerts(prev => [alert, ...prev].slice(0, 50)); // Keep last 50 alerts
    };

    const handleSystemNotification = (notification: SystemNotification) => {
      setSystemNotifications(prev => [notification, ...prev].slice(0, 20)); // Keep last 20 notifications
    };

    // Register event handlers
    webSocketService.on('connection', handleConnection);
    webSocketService.on('dve-node-updated', handleDVENodeUpdate);
    webSocketService.on('validation-task-updated', handleValidationTaskUpdate);
    webSocketService.on('cognitive-engine-updated', handleCognitiveEngineUpdate);
    webSocketService.on('tee-security-updated', handleTEESecurityUpdate);
    webSocketService.on('nrn-staking-updated', handleNRNStakingUpdate);
    webSocketService.on('security-alert', handleSecurityAlert);
    webSocketService.on('system-notification', handleSystemNotification);

    // Subscribe to events
    webSocketService.subscribe([
      'dve-node-updated',
      'validation-task-updated',
      'cognitive-engine-updated',
      'tee-security-updated',
      'nrn-staking-updated',
      'security-alert',
      'system-notification'
    ]);

    // Cleanup on unmount
    return () => {
      webSocketService.off('connection', handleConnection);
      webSocketService.off('dve-node-updated', handleDVENodeUpdate);
      webSocketService.off('validation-task-updated', handleValidationTaskUpdate);
      webSocketService.off('cognitive-engine-updated', handleCognitiveEngineUpdate);
      webSocketService.off('tee-security-updated', handleTEESecurityUpdate);
      webSocketService.off('nrn-staking-updated', handleNRNStakingUpdate);
      webSocketService.off('security-alert', handleSecurityAlert);
      webSocketService.off('system-notification', handleSystemNotification);
    };
  }, []);

  // Function to send DVE node update
  const sendDVENodeUpdate = (update: DVENodeUpdate) => {
    if (isConnected) {
      webSocketService.send({
        type: 'dve-node-update',
        payload: update
      });
    }
  };

  // Function to send validation task update
  const sendValidationTaskUpdate = (update: ValidationTaskUpdate) => {
    if (isConnected) {
      webSocketService.send({
        type: 'validation-task-update',
        payload: update
      });
    }
  };

  // Function to send cognitive engine update
  const sendCognitiveEngineUpdate = (update: CognitiveEngineUpdate) => {
    if (isConnected) {
      webSocketService.send({
        type: 'cognitive-engine-update',
        payload: update
      });
    }
  };

  // Function to send TEE security update
  const sendTEESecurityUpdate = (update: TEESecurityUpdate) => {
    if (isConnected) {
      webSocketService.send({
        type: 'tee-security-update',
        payload: update
      });
    }
  };

  // Function to send NRN staking update
  const sendNRNStakingUpdate = (update: NRNStakingUpdate) => {
    if (isConnected) {
      webSocketService.send({
        type: 'nrn-staking-update',
        payload: update
      });
    }
  };

  // Function to send security alert
  const sendSecurityAlert = (alert: SecurityAlert) => {
    if (isConnected) {
      webSocketService.send({
        type: 'security-alert',
        payload: alert
      });
    }
  };

  // Function to send system notification
  const sendSystemNotification = (notification: SystemNotification) => {
    if (isConnected) {
      webSocketService.send({
        type: 'system-notification',
        payload: notification
      });
    }
  };

  // Function to clear updates
  const clearUpdates = () => {
    setDveNodeUpdates([]);
    setValidationTaskUpdates([]);
    setCognitiveEngineUpdates([]);
    setTeeSecurityUpdates([]);
    setNrnStakingUpdates([]);
    setSecurityAlerts([]);
    setSystemNotifications([]);
  };

  return {
    isConnected,
    dveNodeUpdates,
    validationTaskUpdates,
    cognitiveEngineUpdates,
    teeSecurityUpdates,
    nrnStakingUpdates,
    securityAlerts,
    systemNotifications,
    sendDVENodeUpdate,
    sendValidationTaskUpdate,
    sendCognitiveEngineUpdate,
    sendTEESecurityUpdate,
    sendNRNStakingUpdate,
    sendSecurityAlert,
    sendSystemNotification,
    clearUpdates
  };
};