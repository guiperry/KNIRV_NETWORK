import React, { useState } from 'react';

export default function AdvancedDashboard() {
  const darkBlue = '#000c2e';
  const brightBlue = '#007bff';
  
  const [activeTab, setActiveTab] = useState('dashboard');
  
  const glassyStyle = {
    backgroundColor: 'rgba(255, 255, 255, 0.05)',
    backdropFilter: 'blur(8px)',
    borderRadius: '15px',
    border: '1px solid rgba(255, 255, 255, 0.1)',
    boxShadow: '0 8px 32px 0 rgba(0, 0, 0, 0.1)',
  };
  
  return (
    <div style={{ 
      backgroundColor: darkBlue, 
      color: 'white', 
      minHeight: '100vh',
      display: 'flex'
    }}>
      {/* Sidebar */}
      <div style={{ 
        width: '250px', 
        backgroundColor: 'rgba(0, 0, 0, 0.6)', 
        padding: '20px',
        display: 'flex',
        flexDirection: 'column'
      }}>
        <h2 style={{ color: brightBlue, marginBottom: '30px' }}>Blockchain</h2>
        
        <div 
          style={{ 
            ...glassyStyle, 
            backgroundColor: activeTab === 'dashboard' ? brightBlue : 'rgba(255, 255, 255, 0.05)', 
            padding: '10px 15px', 
            marginBottom: '10px',
            display: 'flex',
            alignItems: 'center',
            cursor: 'pointer'
          }}
          onClick={() => setActiveTab('dashboard')}
        >
          <span style={{ marginRight: '10px' }}>☰</span>
          <span>Dashboard</span>
        </div>
        
        <div 
          style={{ 
            ...glassyStyle, 
            backgroundColor: activeTab === 'wallet' ? brightBlue : 'rgba(255, 255, 255, 0.05)', 
            padding: '10px 15px', 
            marginBottom: '10px',
            display: 'flex',
            alignItems: 'center',
            cursor: 'pointer'
          }}
          onClick={() => setActiveTab('wallet')}
        >
          <span style={{ marginRight: '10px' }}>💼</span>
          <span>Wallet</span>
        </div>
        
        <div 
          style={{ 
            ...glassyStyle, 
            backgroundColor: activeTab === 'networks' ? brightBlue : 'rgba(255, 255, 255, 0.05)', 
            padding: '10px 15px', 
            marginBottom: '10px',
            display: 'flex',
            alignItems: 'center',
            cursor: 'pointer'
          }}
          onClick={() => setActiveTab('networks')}
        >
          <span style={{ marginRight: '10px' }}>🔗</span>
          <span>Networks</span>
        </div>
        
        <div 
          style={{ 
            ...glassyStyle, 
            backgroundColor: activeTab === 'donations' ? brightBlue : 'rgba(255, 255, 255, 0.05)', 
            padding: '10px 15px', 
            marginBottom: '10px',
            display: 'flex',
            alignItems: 'center',
            cursor: 'pointer'
          }}
          onClick={() => setActiveTab('donations')}
        >
          <span style={{ marginRight: '10px' }}>❤️</span>
          <span>Donations</span>
        </div>
        
        <div 
          style={{ 
            ...glassyStyle, 
            backgroundColor: activeTab === 'prediction' ? brightBlue : 'rgba(255, 255, 255, 0.05)', 
            padding: '10px 15px', 
            marginBottom: '10px',
            display: 'flex',
            alignItems: 'center',
            cursor: 'pointer'
          }}
          onClick={() => setActiveTab('prediction')}
        >
          <span style={{ marginRight: '10px' }}>📊</span>
          <span>Prediction</span>
        </div>
      </div>
      
      {/* Main Content */}
      <div style={{ flex: 1, padding: '20px' }}>
        {/* Top Navigation */}
        <div style={{ 
          ...glassyStyle, 
          padding: '15px 20px', 
          marginBottom: '20px',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center'
        }}>
          <h3 style={{ color: brightBlue, margin: 0 }}>Blockchain</h3>
          
          <div style={{ 
            ...glassyStyle,
            padding: '8px 15px',
            display: 'flex',
            alignItems: 'center',
            width: '300px'
          }}>
            <span style={{ marginRight: '10px' }}>🔍</span>
            <input 
              type="text" 
              placeholder="Search blockchain" 
              style={{
                backgroundColor: 'transparent',
                border: 'none',
                color: 'white',
                outline: 'none',
                width: '100%'
              }}
            />
          </div>
          
          <div style={{ display: 'flex', gap: '15px' }}>
            <div style={{ ...glassyStyle, padding: '8px', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <span>🔍</span>
            </div>
            <div style={{ ...glassyStyle, padding: '8px', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <span>🔔</span>
            </div>
            <div style={{ ...glassyStyle, padding: '8px', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <span>⚙️</span>
            </div>
            <div style={{ ...glassyStyle, padding: '8px', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <span>👤</span>
            </div>
          </div>
        </div>
        
        {activeTab === 'dashboard' && (
          <>
            <h2 style={{ color: brightBlue }}>Dashboard</h2>
            <p style={{ color: 'rgba(255, 255, 255, 0.7)' }}>Overview of your blockchain assets</p>
            
            {/* Cards Row */}
            <div style={{ 
              display: 'flex', 
              flexWrap: 'wrap', 
              gap: '20px', 
              marginTop: '20px',
              marginBottom: '20px'
            }}>
              <div style={{ 
                ...glassyStyle,
                backgroundColor: 'rgba(255, 255, 255, 0.08)',
                padding: '20px', 
                flex: '1 1 20%',
                textAlign: 'center'
              }}>
                <h3 style={{ color: brightBlue }}>Refu</h3>
                <h2>15.556</h2>
                <span style={{ color: '#28a745' }}>+2.3%</span>
              </div>
              
              <div style={{ 
                ...glassyStyle,
                backgroundColor: 'rgba(255, 255, 255, 0.08)',
                padding: '20px', 
                flex: '1 1 20%',
                textAlign: 'center'
              }}>
                <h3 style={{ color: brightBlue }}>Dratic</h3>
                <h2>255.0%</h2>
                <span style={{ color: '#dc3545' }}>-1.1%</span>
              </div>
              
              <div style={{ 
                ...glassyStyle,
                backgroundColor: 'rgba(255, 255, 255, 0.08)',
                padding: '20px', 
                flex: '1 1 20%',
                textAlign: 'center'
              }}>
                <h3 style={{ color: brightBlue }}>Hapler</h3>
                <h2>997</h2>
                <span style={{ color: '#28a745' }}>+0.8%</span>
              </div>
              
              <div style={{ 
                ...glassyStyle,
                backgroundColor: 'rgba(255, 255, 255, 0.08)',
                padding: '20px', 
                flex: '1 1 20%',
                textAlign: 'center'
              }}>
                <h3 style={{ color: brightBlue }}>Coetic</h3>
                <h2>461.53</h2>
                <span style={{ color: '#28a745' }}>+0.5%</span>
              </div>
            </div>
            
            {/* Analytics Card */}
            <div style={{ 
              ...glassyStyle,
              backgroundColor: 'rgba(255, 255, 255, 0.08)',
              padding: '20px'
            }}>
              <h3 style={{ color: brightBlue }}>Analytics Overview</h3>
              
              <div style={{ 
                height: '200px', 
                backgroundColor: 'rgba(224, 247, 250, 0.1)', 
                border: '1px dashed rgba(178, 235, 242, 0.3)', 
                borderRadius: '10px', 
                display: 'flex', 
                justifyContent: 'center', 
                alignItems: 'center', 
                color: brightBlue,
                marginBottom: '20px'
              }}>
                <span>Chart Area Placeholder</span>
              </div>
              
              <div style={{ 
                display: 'flex', 
                gap: '20px'
              }}>
                <div style={{ 
                  ...glassyStyle,
                  backgroundColor: 'rgba(255, 255, 255, 0.08)',
                  padding: '20px',
                  flex: '0 0 45%'
                }}>
                  <h3 style={{ color: brightBlue }}>Navigation</h3>
                  <h2>15,0.5K</h2>
                </div>
              </div>
            </div>
          </>
        )}
        
        {activeTab === 'wallet' && (
          <>
            <h2 style={{ color: brightBlue }}>Wallet</h2>
            <p style={{ color: 'rgba(255, 255, 255, 0.7)' }}>Manage your blockchain wallet</p>
            
            <div style={{ 
              ...glassyStyle,
              backgroundColor: 'rgba(255, 255, 255, 0.08)',
              padding: '20px',
              marginTop: '20px'
            }}>
              <h3 style={{ color: brightBlue }}>Wallet Balance</h3>
              <h1>$12,345.67</h1>
              
              <div style={{ 
                display: 'flex', 
                gap: '20px',
                marginTop: '20px'
              }}>
                <button style={{ 
                  backgroundColor: brightBlue,
                  color: 'white',
                  border: 'none',
                  borderRadius: '5px',
                  padding: '10px 20px',
                  cursor: 'pointer'
                }}>
                  Send
                </button>
                
                <button style={{ 
                  backgroundColor: 'transparent',
                  color: 'white',
                  border: `1px solid ${brightBlue}`,
                  borderRadius: '5px',
                  padding: '10px 20px',
                  cursor: 'pointer'
                }}>
                  Receive
                </button>
              </div>
            </div>
            
            <div style={{ 
              ...glassyStyle,
              backgroundColor: 'rgba(255, 255, 255, 0.08)',
              padding: '20px',
              marginTop: '20px'
            }}>
              <h3 style={{ color: brightBlue }}>Recent Transactions</h3>
              
              <div style={{ 
                ...glassyStyle,
                padding: '15px',
                marginTop: '15px',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center'
              }}>
                <div>
                  <div style={{ color: brightBlue }}>Sent Bitcoin</div>
                  <div style={{ color: 'rgba(255, 255, 255, 0.7)', fontSize: '0.9em' }}>June 8, 2023</div>
                </div>
                <div style={{ color: '#dc3545' }}>-0.05 BTC</div>
              </div>
              
              <div style={{ 
                ...glassyStyle,
                padding: '15px',
                marginTop: '15px',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center'
              }}>
                <div>
                  <div style={{ color: brightBlue }}>Received Ethereum</div>
                  <div style={{ color: 'rgba(255, 255, 255, 0.7)', fontSize: '0.9em' }}>June 7, 2023</div>
                </div>
                <div style={{ color: '#28a745' }}>+2.5 ETH</div>
              </div>
            </div>
          </>
        )}
        
        {activeTab === 'networks' && (
          <>
            <h2 style={{ color: brightBlue }}>Networks</h2>
            <p style={{ color: 'rgba(255, 255, 255, 0.7)' }}>Manage blockchain networks</p>
            
            <div style={{ 
              ...glassyStyle,
              backgroundColor: 'rgba(255, 255, 255, 0.08)',
              padding: '20px',
              marginTop: '20px'
            }}>
              <h3 style={{ color: brightBlue }}>Available Networks</h3>
              
              <div style={{ 
                ...glassyStyle,
                padding: '15px',
                marginTop: '15px',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center'
              }}>
                <div style={{ display: 'flex', alignItems: 'center' }}>
                  <div style={{ 
                    width: '30px', 
                    height: '30px', 
                    borderRadius: '50%', 
                    backgroundColor: '#f2a900',
                    marginRight: '15px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center'
                  }}>₿</div>
                  <div>Bitcoin</div>
                </div>
                <div style={{ color: '#28a745' }}>Connected</div>
              </div>
              
              <div style={{ 
                ...glassyStyle,
                padding: '15px',
                marginTop: '15px',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center'
              }}>
                <div style={{ display: 'flex', alignItems: 'center' }}>
                  <div style={{ 
                    width: '30px', 
                    height: '30px', 
                    borderRadius: '50%', 
                    backgroundColor: '#627eea',
                    marginRight: '15px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center'
                  }}>Ξ</div>
                  <div>Ethereum</div>
                </div>
                <div style={{ color: '#28a745' }}>Connected</div>
              </div>
              
              <div style={{ 
                ...glassyStyle,
                padding: '15px',
                marginTop: '15px',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center'
              }}>
                <div style={{ display: 'flex', alignItems: 'center' }}>
                  <div style={{ 
                    width: '30px', 
                    height: '30px', 
                    borderRadius: '50%', 
                    backgroundColor: '#345d9d',
                    marginRight: '15px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center'
                  }}>L</div>
                  <div>Litecoin</div>
                </div>
                <button style={{ 
                  backgroundColor: brightBlue,
                  color: 'white',
                  border: 'none',
                  borderRadius: '5px',
                  padding: '5px 10px',
                  cursor: 'pointer'
                }}>
                  Connect
                </button>
              </div>
            </div>
          </>
        )}
        
        {activeTab === 'donations' && (
          <>
            <h2 style={{ color: brightBlue }}>Donations</h2>
            <p style={{ color: 'rgba(255, 255, 255, 0.7)' }}>Support blockchain projects</p>
            
            <div style={{ 
              ...glassyStyle,
              backgroundColor: 'rgba(255, 255, 255, 0.08)',
              padding: '20px',
              marginTop: '20px'
            }}>
              <h3 style={{ color: brightBlue }}>Featured Projects</h3>
              
              <div style={{ 
                display: 'flex', 
                flexWrap: 'wrap', 
                gap: '20px', 
                marginTop: '20px'
              }}>
                <div style={{ 
                  ...glassyStyle,
                  backgroundColor: 'rgba(255, 255, 255, 0.08)',
                  padding: '20px', 
                  flex: '1 1 45%'
                }}>
                  <h3 style={{ color: brightBlue }}>Blockchain Education</h3>
                  <p>Supporting educational resources for blockchain technology.</p>
                  <div style={{ 
                    width: '100%', 
                    height: '10px', 
                    backgroundColor: 'rgba(255, 255, 255, 0.1)',
                    borderRadius: '5px',
                    marginBottom: '10px'
                  }}>
                    <div style={{ 
                      width: '65%', 
                      height: '100%', 
                      backgroundColor: brightBlue,
                      borderRadius: '5px'
                    }}></div>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                    <span>$6,500 raised</span>
                    <span>Goal: $10,000</span>
                  </div>
                  <button style={{ 
                    backgroundColor: brightBlue,
                    color: 'white',
                    border: 'none',
                    borderRadius: '5px',
                    padding: '10px 20px',
                    cursor: 'pointer',
                    marginTop: '15px',
                    width: '100%'
                  }}>
                    Donate
                  </button>
                </div>
                
                <div style={{ 
                  ...glassyStyle,
                  backgroundColor: 'rgba(255, 255, 255, 0.08)',
                  padding: '20px', 
                  flex: '1 1 45%'
                }}>
                  <h3 style={{ color: brightBlue }}>Open Source Development</h3>
                  <p>Supporting open source blockchain projects.</p>
                  <div style={{ 
                    width: '100%', 
                    height: '10px', 
                    backgroundColor: 'rgba(255, 255, 255, 0.1)',
                    borderRadius: '5px',
                    marginBottom: '10px'
                  }}>
                    <div style={{ 
                      width: '40%', 
                      height: '100%', 
                      backgroundColor: brightBlue,
                      borderRadius: '5px'
                    }}></div>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                    <span>$8,000 raised</span>
                    <span>Goal: $20,000</span>
                  </div>
                  <button style={{ 
                    backgroundColor: brightBlue,
                    color: 'white',
                    border: 'none',
                    borderRadius: '5px',
                    padding: '10px 20px',
                    cursor: 'pointer',
                    marginTop: '15px',
                    width: '100%'
                  }}>
                    Donate
                  </button>
                </div>
              </div>
            </div>
          </>
        )}
        
        {activeTab === 'prediction' && (
          <>
            <h2 style={{ color: brightBlue }}>Prediction</h2>
            <p style={{ color: 'rgba(255, 255, 255, 0.7)' }}>Cryptocurrency price predictions</p>
            
            <div style={{ 
              ...glassyStyle,
              backgroundColor: 'rgba(255, 255, 255, 0.08)',
              padding: '20px',
              marginTop: '20px'
            }}>
              <h3 style={{ color: brightBlue }}>Price Predictions</h3>
              
              <div style={{ 
                height: '300px', 
                backgroundColor: 'rgba(224, 247, 250, 0.1)', 
                border: '1px dashed rgba(178, 235, 242, 0.3)', 
                borderRadius: '10px', 
                display: 'flex', 
                justifyContent: 'center', 
                alignItems: 'center', 
                color: brightBlue,
                marginBottom: '20px',
                marginTop: '20px'
              }}>
                <span>Price Prediction Chart Placeholder</span>
              </div>
              
              <div style={{ 
                display: 'flex', 
                flexWrap: 'wrap', 
                gap: '20px', 
                marginTop: '20px'
              }}>
                <div style={{ 
                  ...glassyStyle,
                  backgroundColor: 'rgba(255, 255, 255, 0.08)',
                  padding: '20px', 
                  flex: '1 1 30%',
                  textAlign: 'center'
                }}>
                  <h3 style={{ color: brightBlue }}>Bitcoin</h3>
                  <h2>$45,678</h2>
                  <span style={{ color: '#28a745' }}>Predicted: $50,000</span>
                </div>
                
                <div style={{ 
                  ...glassyStyle,
                  backgroundColor: 'rgba(255, 255, 255, 0.08)',
                  padding: '20px', 
                  flex: '1 1 30%',
                  textAlign: 'center'
                }}>
                  <h3 style={{ color: brightBlue }}>Ethereum</h3>
                  <h2>$3,456</h2>
                  <span style={{ color: '#28a745' }}>Predicted: $4,000</span>
                </div>
                
                <div style={{ 
                  ...glassyStyle,
                  backgroundColor: 'rgba(255, 255, 255, 0.08)',
                  padding: '20px', 
                  flex: '1 1 30%',
                  textAlign: 'center'
                }}>
                  <h3 style={{ color: brightBlue }}>Litecoin</h3>
                  <h2>$180</h2>
                  <span style={{ color: '#dc3545' }}>Predicted: $150</span>
                </div>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  );
}