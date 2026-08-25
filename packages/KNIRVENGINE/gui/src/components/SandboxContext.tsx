import React, {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  useRef,
} from 'react';
import {
  createSandboxSession,
  closeSandboxSession,
  listSandboxSessions,
  createSandboxStatusWebSocket,
  getSandboxDependencies,
  installSandboxDependencies,
  type SandboxSession,
  type SandboxLaunchConfig,
  type SandboxSessionStatus,
  type DependencyStatus,
  type SandboxProxyFlow,
} from '../services/sandboxService';

export type SandboxProjectFile = {
  name: string;
  path: string;
  file: File;
  handle?: {
    kind: 'file';
    name: string;
    getFile(): Promise<File>;
    createWritable?: () => Promise<{ write(contents: string): Promise<void>; close(): Promise<void> }>;
  };
  nativePath?: string;
  nativeSize?: number;
};

interface SandboxContextValue {
  session: SandboxSession | null;
  targetLabel: string;
  projectPath: string;
  projectFiles: SandboxProjectFile[];
  projectTargetPath: string;
  proxyFlows: SandboxProxyFlow[];
  status: SandboxSessionStatus | null;
  log: string[];
  isReady: boolean;
  error: string | null;
  launch: (config: SandboxLaunchConfig) => Promise<void>;
  stop: () => Promise<void>;
  clearLog: () => void;
  setTargetLabel: (label: string) => void;
  setProjectPath: (path: string) => void;
  setProjectFiles: (files: SandboxProjectFile[]) => void;
  setProjectTargetPath: (path: string) => void;
  refresh: () => Promise<void>;
  deps: DependencyStatus[] | null;
  depsInstalling: boolean;
  installDeps: () => Promise<void>;
}

const SandboxContext = createContext<SandboxContextValue | undefined>(undefined);

export const useSandbox = (): SandboxContextValue => {
  const context = useContext(SandboxContext);
  if (!context) {
    throw new Error('useSandbox must be used within a SandboxProvider');
  }
  return context;
};

export const SandboxProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [session, setSession] = useState<SandboxSession | null>(null);
  const [targetLabel, setTargetLabelState] = useState('');
  const [projectPath, setProjectPathState] = useState('');
  const [projectFiles, setProjectFilesState] = useState<SandboxProjectFile[]>([]);
  const [projectTargetPath, setProjectTargetPathState] = useState('');
  const [proxyFlows, setProxyFlows] = useState<SandboxProxyFlow[]>([]);
  const [status, setStatus] = useState<SandboxSessionStatus | null>(null);
  const [log, setLog] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [deps, setDeps] = useState<DependencyStatus[] | null>(null);
  const [depsInstalling, setDepsInstalling] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  const setTargetLabel = useCallback((label: string) => {
    setTargetLabelState(label);
  }, []);
  const setProjectPath = useCallback((path: string) => {
    setProjectPathState(path);
  }, []);
  const setProjectFiles = useCallback((files: SandboxProjectFile[]) => {
    setProjectFilesState(files);
  }, []);
  const setProjectTargetPath = useCallback((path: string) => {
    setProjectTargetPathState(path);
  }, []);
  const clearLog = useCallback(() => setLog([]), []);

  const connectStatusStream = useCallback((id: string) => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    const ws = createSandboxStatusWebSocket(id, {
      onMessage: (event: MessageEvent) => {
        try {
          const message = JSON.parse(event.data);
          if (message.type === 'status') {
            setStatus(message.status);
            setError(message.error || null);
            setSession((prev) =>
              prev ? { ...prev, status: message.status, error: message.error || undefined } : prev
            );
          } else if (message.type === 'log') {
            setLog((prev) => [...prev, message.data]);
          } else if (message.type === 'frontend' && typeof message.url === 'string') {
            setSession((prev) => prev ? { ...prev, frontendUrl: message.url } : prev);
          } else if (message.type === 'proxy_flow' && message.flow) {
            setProxyFlows((prev) => [...prev.filter((flow) => flow.id !== message.flow.id), message.flow].slice(-200));
          }
        } catch {
          /* ignore malformed frames */
        }
      },
      onClose: () => {
        wsRef.current = null;
      },
    });
    wsRef.current = ws;
  }, []);

  // Re-fetch server truth on mount instead of trusting a cached session, so a
  // reload never shows "running" for a session that actually died.
  const refresh = useCallback(async () => {
    try {
      const sessions = await listSandboxSessions();
      const live = sessions[0];
      if (live) {
        setSession(live);
        setStatus(live.status);
        setError(live.error || null);
        if (live.status === 'running' || live.status === 'provisioning') {
          connectStatusStream(live.id);
        }
      }
    } catch {
      /* backend not reachable — stay empty */
    }
  }, [connectStatusStream]);

  const installDeps = useCallback(async () => {
    setDepsInstalling(true);
    try {
      const report = await installSandboxDependencies();
      setDeps(report.dependencies ?? []);
    } catch {
      /* leave current state; the consumer surfaces the manual command */
    } finally {
      setDepsInstalling(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
    // Seamlessly verify + auto-install the sandbox system dependencies
    // (bubblewrap, Xvfb, x11vnc) per-OS on first load, so the user never
    // has to provision them by hand.
    (async () => {
      try {
        const report = await getSandboxDependencies();
        const missing = (report.dependencies ?? []).filter((d) => !d.present);
        if (missing.length > 0) {
          await installDeps();
        } else {
          setDeps(report.dependencies ?? []);
        }
      } catch {
        /* backend not reachable — stay empty */
      }
    })();
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [refresh, installDeps]);

  const launch = useCallback(
    async (config: SandboxLaunchConfig) => {
      setError(null);
      setLog([]);
      setProxyFlows([]);
      const created = await createSandboxSession(config);
      setSession(created);
      setStatus(created.status);
      setLog((prev) => [...prev, `[sandbox] session ${created.id} created`]);
      connectStatusStream(created.id);
    },
    [connectStatusStream]
  );

  const stop = useCallback(async () => {
    if (!session) return;
    try {
      await closeSandboxSession(session.id);
    } catch {
      /* best effort */
    } finally {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      setSession(null);
      setStatus(null);
      setLog([]);
      setError(null);
    }
  }, [session]);

  const isReady = status === 'running';

  return (
    <SandboxContext.Provider
      value={{
        session,
        targetLabel,
        projectPath,
        projectFiles,
        projectTargetPath,
        proxyFlows,
        status,
        log,
        isReady,
        error,
        launch,
        stop,
        clearLog,
        setTargetLabel,
        setProjectPath,
        setProjectFiles,
        setProjectTargetPath,
        refresh,
        deps,
        depsInstalling,
        installDeps,
      }}
    >
      {children}
    </SandboxContext.Provider>
  );
};

export default SandboxProvider;
