import React, { useState } from 'react';
import { useRole } from '../contexts/RoleContext';
import styles from './RoleSwitcher.module.css';

const RoleSwitcher = () => {
  const { role, network, logout, getUserInfo } = useRole();
  const [showSwitcher, setShowSwitcher] = useState(false);
  
  const roles = [
    { value: 'Root', label: 'Root', description: 'Full system access' },
    { value: 'Bootnode', label: 'Bootnode', description: 'Network bootstrap node' },
    { value: 'Dev', label: 'Developer', description: 'Development access' },
    { value: 'General', label: 'General', description: 'Basic user access' }
  ];

  const networks = [
    { value: 'mainnet', label: 'Mainnet' },
    { value: 'public-testnet', label: 'Public Testnet' },
    { value: 'private-testnet', label: 'Private Testnet' },
    { value: 'demo', label: 'Demo Mode' }
  ];

  const handleRoleSwitch = (newRole, newNetwork = 'demo') => {
    // Clear existing auth
    localStorage.removeItem('knirv_user_role');
    localStorage.removeItem('knirv_network');
    localStorage.removeItem('knirv_auth_token');
    localStorage.removeItem('knirv_demo_mode');
    
    // Set new role and network
    localStorage.setItem('knirv_user_role', newRole);
    localStorage.setItem('knirv_network', newNetwork);
    localStorage.setItem('knirv_auth_token', `demo-token-${newRole.toLowerCase()}`);
    
    // Reload to apply changes
    window.location.reload();
  };

  const handleLogout = () => {
    logout();
  };

  const handleClearAuth = () => {
    localStorage.removeItem('knirv_user_role');
    localStorage.removeItem('knirv_network');
    localStorage.removeItem('knirv_auth_token');
    localStorage.removeItem('knirv_demo_mode');
    window.location.reload();
  };

  const userInfo = getUserInfo();

  return (
    <div className={styles.roleSwitcher}>
      <div className={styles.currentRole} onClick={() => setShowSwitcher(!showSwitcher)}>
        <div className={styles.roleInfo}>
          <span className={styles.roleLabel}>Role: {userInfo.displayName}</span>
          <span className={styles.networkLabel}>Network: {userInfo.networkDisplay}</span>
        </div>
        <span className={styles.toggleIcon}>{showSwitcher ? '▲' : '▼'}</span>
      </div>
      
      {showSwitcher && (
        <div className={styles.switcherPanel}>
          <h3>Authentication Testing</h3>
          
          <div className={styles.section}>
            <h4>Switch Role</h4>
            <div className={styles.roleGrid}>
              {roles.map((roleOption) => (
                <button
                  key={roleOption.value}
                  className={`${styles.roleButton} ${role === roleOption.value ? styles.active : ''}`}
                  onClick={() => handleRoleSwitch(roleOption.value)}
                  title={roleOption.description}
                >
                  <div className={styles.roleButtonContent}>
                    <span className={styles.roleButtonLabel}>{roleOption.label}</span>
                    <span className={styles.roleButtonDesc}>{roleOption.description}</span>
                  </div>
                </button>
              ))}
            </div>
          </div>

          <div className={styles.section}>
            <h4>Network Options</h4>
            <div className={styles.networkGrid}>
              {networks.map((networkOption) => (
                <button
                  key={networkOption.value}
                  className={`${styles.networkButton} ${network === networkOption.value ? styles.active : ''}`}
                  onClick={() => handleRoleSwitch(role, networkOption.value)}
                >
                  {networkOption.label}
                </button>
              ))}
            </div>
          </div>

          <div className={styles.section}>
            <h4>Authentication Actions</h4>
            <div className={styles.actionButtons}>
              <button className={styles.logoutButton} onClick={handleLogout}>
                Logout
              </button>
              <button className={styles.clearButton} onClick={handleClearAuth}>
                Clear Auth & Show Login
              </button>
            </div>
          </div>

          <div className={styles.section}>
            <h4>Current Authentication State</h4>
            <div className={styles.authInfo}>
              <div>Role: <strong>{role}</strong></div>
              <div>Network: <strong>{network}</strong></div>
              <div>Authenticated: <strong>{userInfo.isAuthenticated ? 'Yes' : 'No'}</strong></div>
              <div>Demo Mode: <strong>{localStorage.getItem('knirv_demo_mode') === 'true' ? 'Yes' : 'No'}</strong></div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default RoleSwitcher;
