"use client";

import { useEffect, useState } from 'react';
import { io, Socket } from 'socket.io-client';

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
  const [socket, setSocket] = useState<Socket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [dveNodeUpdates, setDveNodeUpdates] = useState<DVENodeUpdate[]>([]);
  const [validationTaskUpdates, setValidationTaskUpdates] = useState<ValidationTaskUpdate[]>([]);
  const [cognitiveEngineUpdates, setCognitiveEngineUpdates] = useState<CognitiveEngineUpdate[]>([]);
  const [teeSecurityUpdates, setTeeSecurityUpdates] = useState<TEESecurityUpdate[]>([]);
  const [nrnStakingUpdates, setNrnStakingUpdates] = useState<NRNStakingUpdate[]>([]);
  const [securityAlerts, setSecurityAlerts] = useState<SecurityAlert[]>([]);
  const [systemNotifications, setSystemNotifications] = useState<SystemNotification[]>([]);

  useEffect(() => {
    // Initialize socket connection
    const socketInstance = io('http://localhost:3001', {
      transports: ['websocket', 'polling']
    });

    socketInstance.on('connect', () => {
      console.log('Connected to KNIRV-NEXUS DVE WebSocket');
      setIsConnected(true);
      
      // Join monitoring room
      socketInstance.emit('join-room', 'dve-monitoring');
      
      // Request initial sync
      socketInstance.emit('request-sync');
      
      // Subscribe to all metrics
      socketInstance.emit('subscribe-metrics', ['dve-nodes', 'validation-tasks', 'cognitive-engine', 'tee-security', 'nrn-staking']);
    });

    socketInstance.on('disconnect', () => {
      console.log('Disconnected from KNIRV-NEXUS DVE WebSocket');
      setIsConnected(false);
    });

    // Listen for DVE node updates
    socketInstance.on('dve-node-updated', (update: DVENodeUpdate) => {
      setDveNodeUpdates(prev => {
        const existing = prev.find(u => u.id === update.id);
        if (existing) {
          return prev.map(u => u.id === update.id ? update : u);
        }
        return [...prev, update];
      });
    });

    // Listen for validation task updates
    socketInstance.on('validation-task-updated', (update: ValidationTaskUpdate) => {
      setValidationTaskUpdates(prev => {
        const existing = prev.find(u => u.id === update.id);
        if (existing) {
          return prev.map(u => u.id === update.id ? update : u);
        }
        return [...prev, update];
      });
    });

    // Listen for cognitive engine updates
    socketInstance.on('cognitive-engine-updated', (update: CognitiveEngineUpdate) => {
      setCognitiveEngineUpdates(prev => {
        const existing = prev.find(u => u.status === update.status);
        if (existing) {
          return prev.map(u => u.status === update.status ? update : u);
        }
        return [...prev, update];
      });
    });

    // Listen for TEE security updates
    socketInstance.on('tee-security-updated', (update: TEESecurityUpdate) => {
      setTeeSecurityUpdates(prev => {
        const existing = prev.find(u => u.attestation_status === update.attestation_status);
        if (existing) {
          return prev.map(u => u.attestation_status === update.attestation_status ? update : u);
        }
        return [...prev, update];
      });
    });

    // Listen for NRN staking updates
    socketInstance.on('nrn-staking-updated', (update: NRNStakingUpdate) => {
      setNrnStakingUpdates(prev => {
        const existing = prev.find(u => u.total_staked === update.total_staked);
        if (existing) {
          return prev.map(u => u.total_staked === update.total_staked ? update : u);
        }
        return [...prev, update];
      });
    });

    // Listen for security alerts
    socketInstance.on('security-alert', (alert: SecurityAlert) => {
      setSecurityAlerts(prev => [alert, ...prev].slice(0, 50)); // Keep last 50 alerts
    });

    // Listen for system notifications
    socketInstance.on('system-notification', (notification: SystemNotification) => {
      setSystemNotifications(prev => [notification, ...prev].slice(0, 20)); // Keep last 20 notifications
    });

    setSocket(socketInstance);

    // Cleanup on unmount
    return () => {
      socketInstance.disconnect();
    };
  }, []);

  // Function to send DVE node update
  const sendDVENodeUpdate = (update: DVENodeUpdate) => {
    if (socket && isConnected) {
      socket.emit('dve-node-update', update);
    }
  };

  // Function to send validation task update
  const sendValidationTaskUpdate = (update: ValidationTaskUpdate) => {
    if (socket && isConnected) {
      socket.emit('validation-task-update', update);
    }
  };

  // Function to send cognitive engine update
  const sendCognitiveEngineUpdate = (update: CognitiveEngineUpdate) => {
    if (socket && isConnected) {
      socket.emit('cognitive-engine-update', update);
    }
  };

  // Function to send TEE security update
  const sendTEESecurityUpdate = (update: TEESecurityUpdate) => {
    if (socket && isConnected) {
      socket.emit('tee-security-update', update);
    }
  };

  // Function to send NRN staking update
  const sendNRNStakingUpdate = (update: NRNStakingUpdate) => {
    if (socket && isConnected) {
      socket.emit('nrn-staking-update', update);
    }
  };

  // Function to send security alert
  const sendSecurityAlert = (alert: SecurityAlert) => {
    if (socket && isConnected) {
      socket.emit('security-alert', alert);
    }
  };

  // Function to send system notification
  const sendSystemNotification = (notification: SystemNotification) => {
    if (socket && isConnected) {
      socket.emit('system-notification', notification);
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
    socket,
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