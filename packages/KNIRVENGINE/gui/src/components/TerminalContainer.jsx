import React, { useState, useEffect } from 'react';
import TerminalUI from './TerminalUI';
import terminalService from '../services/terminalService';
import { Button, Card, CardHeader, CardTitle, CardContent, CardFooter } from '../ui/components';
import { Terminal, Plus, List, X } from 'lucide-react';

/**
 * Terminal Container Component
 * 
 * This component manages multiple terminal sessions and provides a UI for creating,
 * switching between, and closing terminal sessions.
 */
const TerminalContainer = ({ 
  initialWorkingDir = '',
  initialEnv = {},
  theme = 'dark',
  height = '500px',
  width = '100%'
}) => {
  const [sessions, setSessions] = useState([]);
  const [activeSessionId, setActiveSessionId] = useState(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState(null);
  const [showSessionList, setShowSessionList] = useState(false);

  // Load existing sessions on mount
  useEffect(() => {
    loadSessions();
  }, []);

  // Create a new session if no sessions exist
  useEffect(() => {
    if (sessions.length === 0 && !isLoading) {
      createNewSession();
    } else if (sessions.length > 0 && !activeSessionId) {
      setActiveSessionId(sessions[0].id);
    }
  }, [sessions, isLoading]);

  // Load existing terminal sessions
  const loadSessions = async () => {
    try {
      setIsLoading(true);
      setError(null);
      
      const response = await terminalService.listTerminalSessions();
      setSessions(response.sessions || []);
      
      if (response.sessions && response.sessions.length > 0) {
        setActiveSessionId(response.sessions[0].id);
      }
    } catch (error) {
      setError('Failed to load terminal sessions: ' + error.message);
      console.error('Error loading terminal sessions:', error);
    } finally {
      setIsLoading(false);
    }
  };

  // Create a new terminal session
  const createNewSession = async () => {
    try {
      setIsLoading(true);
      setError(null);
      
      const response = await terminalService.createTerminalSession({
        workingDir: initialWorkingDir,
        env: initialEnv
      });
      
      setSessions(prevSessions => [...prevSessions, response.session]);
      setActiveSessionId(response.session.id);
    } catch (error) {
      setError('Failed to create terminal session: ' + error.message);
      console.error('Error creating terminal session:', error);
    } finally {
      setIsLoading(false);
    }
  };

  // Close a terminal session
  const closeSession = async (sessionId) => {
    try {
      await terminalService.closeTerminalSession(sessionId);
      
      setSessions(prevSessions => prevSessions.filter(session => session.id !== sessionId));
      
      // If the active session was closed, activate another session
      if (activeSessionId === sessionId) {
        const remainingSessions = sessions.filter(session => session.id !== sessionId);
        if (remainingSessions.length > 0) {
          setActiveSessionId(remainingSessions[0].id);
        } else {
          setActiveSessionId(null);
        }
      }
    } catch (error) {
      setError('Failed to close terminal session: ' + error.message);
      console.error('Error closing terminal session:', error);
    }
  };

  // Get the active session
  const getActiveSession = () => {
    return sessions.find(session => session.id === activeSessionId);
  };

  return (
    <Card className="terminal-container" style={{ 
      height, 
      width, 
      display: 'flex', 
      flexDirection: 'column',
      backgroundColor: theme === 'dark' ? '#1e1e1e' : '#f0f0f0',
      color: theme === 'dark' ? '#f0f0f0' : '#333',
      border: `1px solid ${theme === 'dark' ? '#333' : '#ccc'}`,
    }}>
      <CardHeader className="terminal-header" style={{
        padding: '8px 16px',
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        borderBottom: `1px solid ${theme === 'dark' ? '#333' : '#ccc'}`,
      }}>
        <div className="terminal-title" style={{ display: 'flex', alignItems: 'center' }}>
          <Terminal size={18} style={{ marginRight: '8px' }} />
          <CardTitle style={{ margin: 0, fontSize: '1rem' }}>
            {getActiveSession()?.workingDir || 'Terminal'}
          </CardTitle>
        </div>
        
        <div className="terminal-actions" style={{ display: 'flex', gap: '8px' }}>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setShowSessionList(!showSessionList)}
            title="Show Sessions"
          >
            <List size={16} />
          </Button>
          
          <Button
            variant="ghost"
            size="icon"
            onClick={createNewSession}
            title="New Terminal"
          >
            <Plus size={16} />
          </Button>
        </div>
      </CardHeader>
      
      {showSessionList && (
        <div className="terminal-sessions" style={{
          padding: '8px',
          borderBottom: `1px solid ${theme === 'dark' ? '#333' : '#ccc'}`,
          backgroundColor: theme === 'dark' ? '#252525' : '#e0e0e0',
        }}>
          <h3 style={{ margin: '0 0 8px 0', fontSize: '0.9rem' }}>Terminal Sessions</h3>
          
          {sessions.length === 0 ? (
            <p style={{ margin: 0, fontSize: '0.8rem' }}>No active sessions</p>
          ) : (
            <ul style={{ 
              margin: 0, 
              padding: 0, 
              listStyle: 'none',
              maxHeight: '200px',
              overflowY: 'auto'
            }}>
              {sessions.map(session => (
                <li key={session.id} style={{
                  padding: '8px',
                  margin: '4px 0',
                  backgroundColor: session.id === activeSessionId 
                    ? (theme === 'dark' ? '#333' : '#ddd') 
                    : 'transparent',
                  borderRadius: '4px',
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  cursor: 'pointer'
                }}
                onClick={() => setActiveSessionId(session.id)}>
                  <div>
                    <div style={{ fontWeight: session.id === activeSessionId ? 'bold' : 'normal' }}>
                      {session.workingDir || 'Terminal'}
                    </div>
                    <div style={{ fontSize: '0.7rem', color: theme === 'dark' ? '#aaa' : '#666' }}>
                      Status: {session.status}, Created: {new Date(session.createdAt).toLocaleTimeString()}
                    </div>
                  </div>
                  
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={(e) => {
                      e.stopPropagation();
                      closeSession(session.id);
                    }}
                    title="Close Session"
                  >
                    <X size={14} />
                  </Button>
                </li>
              ))}
            </ul>
          )}
          
          <div style={{ marginTop: '8px', display: 'flex', justifyContent: 'flex-end' }}>
            <Button
              variant="outline"
              size="sm"
              onClick={createNewSession}
            >
              New Terminal
            </Button>
          </div>
        </div>
      )}
      
      <CardContent className="terminal-content" style={{
        flex: 1,
        padding: 0,
        overflow: 'hidden',
        position: 'relative'
      }}>
        {isLoading ? (
          <div style={{
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            height: '100%',
            flexDirection: 'column'
          }}>
            <div className="loading-spinner" style={{
              border: `4px solid ${theme === 'dark' ? '#333' : '#eee'}`,
              borderTop: `4px solid ${theme === 'dark' ? '#f0f0f0' : '#333'}`,
              borderRadius: '50%',
              width: '40px',
              height: '40px',
              animation: 'spin 1s linear infinite',
              marginBottom: '16px'
            }} />
            <p>Loading terminal...</p>
            <style>{`
              @keyframes spin {
                0% { transform: rotate(0deg); }
                100% { transform: rotate(360deg); }
              }
            `}</style>
          </div>
        ) : error ? (
          <div style={{
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            height: '100%',
            flexDirection: 'column',
            padding: '16px',
            color: '#f44336'
          }}>
            <p>{error}</p>
            <Button
              variant="outline"
              onClick={loadSessions}
            >
              Retry
            </Button>
          </div>
        ) : activeSessionId ? (
          <TerminalUI
            sessionId={activeSessionId}
            apiEndpoint="/api/terminal"
            onClose={() => closeSession(activeSessionId)}
            workingDirectory={getActiveSession()?.workingDir || initialWorkingDir}
            theme={theme}
          />
        ) : (
          <div style={{
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            height: '100%',
            flexDirection: 'column'
          }}>
            <p>No active terminal session</p>
            <Button
              variant="outline"
              onClick={createNewSession}
            >
              Create Terminal
            </Button>
          </div>
        )}
      </CardContent>
      
      {sessions.length > 1 && (
        <CardFooter className="terminal-footer" style={{
          padding: '4px 8px',
          borderTop: `1px solid ${theme === 'dark' ? '#333' : '#ccc'}`,
          display: 'flex',
          overflowX: 'auto',
          backgroundColor: theme === 'dark' ? '#252525' : '#e0e0e0',
        }}>
          {sessions.map(session => (
            <Button
              key={session.id}
              variant={session.id === activeSessionId ? "default" : "ghost"}
              size="sm"
              onClick={() => setActiveSessionId(session.id)}
              style={{ 
                margin: '0 4px',
                padding: '2px 8px',
                minWidth: 'auto',
                fontSize: '0.8rem',
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                maxWidth: '150px'
              }}
            >
              {session.workingDir ? session.workingDir.split('/').pop() : 'Terminal'}
            </Button>
          ))}
        </CardFooter>
      )}
    </Card>
  );
};

export default TerminalContainer;