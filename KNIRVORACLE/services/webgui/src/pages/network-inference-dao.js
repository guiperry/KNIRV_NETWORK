import React, { useState, useEffect } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './network-inference-dao.module.css';

export default function NetworkInferenceDAO() {
  const { activePage } = useNavigation('network-inference-dao');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedModel, setSelectedModel] = useState(null);
  const [hasVoted, setHasVoted] = useState(false);
  const [votingTimeLeft, setVotingTimeLeft] = useState(604800); // 7 days in seconds
  const [userVotingPower, setUserVotingPower] = useState(150);
  const [activeModel, setActiveModel] = useState('GPT-4 Turbo');
  const [activeModelExpiry, setActiveModelExpiry] = useState(172800); // 2 days remaining

  // Mock model data
  const models = [
    {
      id: 'model-1',
      name: 'GPT-4 Turbo',
      provider: 'OpenAI',
      performance: '98.5%',
      cost: '$0.002/1K tokens',
      latency: '120ms',
      description: 'Latest GPT-4 model with improved reasoning and reduced hallucinations',
      votes: 312,
      totalVotes: 1247,
      features: ['Advanced reasoning', 'Code generation', 'Multilingual support']
    },
    {
      id: 'model-2',
      name: 'Claude 3 Opus',
      provider: 'Anthropic',
      performance: '97.8%',
      cost: '$0.015/1K tokens',
      latency: '150ms',
      description: 'Most capable Claude model with enhanced safety and reasoning',
      votes: 289,
      totalVotes: 1247,
      features: ['Constitutional AI', 'Long context', 'Safety-focused']
    },
    {
      id: 'model-3',
      name: 'Gemini Ultra',
      provider: 'Google',
      performance: '96.9%',
      cost: '$0.001/1K tokens',
      latency: '95ms',
      description: 'Google\'s most advanced multimodal model',
      votes: 267,
      totalVotes: 1247,
      features: ['Multimodal', 'Fast inference', 'Cost-effective']
    },
    {
      id: 'model-4',
      name: 'Llama 3.1 405B',
      provider: 'Meta',
      performance: '95.2%',
      cost: '$0.003/1K tokens',
      latency: '200ms',
      description: 'Open-source model with strong reasoning capabilities',
      votes: 198,
      totalVotes: 1247,
      features: ['Open source', 'Customizable', 'Large context window']
    },
    {
      id: 'model-5',
      name: 'KNIRV-Nexus-1',
      provider: 'KNIRV Network',
      performance: '94.7%',
      cost: '$0.0005/1K tokens',
      latency: '180ms',
      description: 'Proprietary KNIRV model optimized for network tasks',
      votes: 181,
      totalVotes: 1247,
      features: ['Network-optimized', 'Low cost', 'KNIRV ecosystem']
    }
  ];

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  const handleVote = (modelId) => {
    if (hasVoted) return;

    setSelectedModel(modelId);
    setHasVoted(true);
    // In a real app, this would submit the vote to the blockchain
    alert(`Vote submitted for ${models.find(m => m.id === modelId)?.name}`);
  };

  // Countdown timers
  useEffect(() => {
    const votingTimer = setInterval(() => {
      setVotingTimeLeft(prev => {
        if (prev <= 1) {
          clearInterval(votingTimer);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    const expiryTimer = setInterval(() => {
      setActiveModelExpiry(prev => {
        if (prev <= 1) {
          clearInterval(expiryTimer);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => {
      clearInterval(votingTimer);
      clearInterval(expiryTimer);
    };
  }, []);

  const formatTime = (seconds) => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${days}d ${hours.toString().padStart(2, '0')}h ${minutes.toString().padStart(2, '0')}m`;
  };

  const currentWinner = models.reduce((prev, current) =>
    (prev.votes > current.votes) ? prev : current
  );

  return (
    <PageLayout activePage={activePage} pageTitle="Network Inference DAO" onSearch={handleSearch}>
      <PageHeader
        title="Network Inference DAO Governance"
        subtitle="Vote for the best AI model to power the entire KNIRV network"
        titleColor="#8e44ad"
      />

      <div className={styles.daoContainer}>
        {/* Current Active Model */}
        <GlassyCard className={styles.activeModelCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-crown" style={{ marginRight: '10px', color: '#ffd700' }}></i>
            Currently Active Model
          </h3>
          <div className={styles.activeModelInfo}>
            <div className={styles.activeModelName}>{activeModel}</div>
            <div className={styles.activeModelExpiry}>
              Expires in: {formatTime(activeModelExpiry)}
            </div>
            <div className={styles.activeModelStatus}>
              {activeModelExpiry > 86400 ? 'Active' : 'Expiring Soon'}
            </div>
          </div>
        </GlassyCard>

        {/* Voting Status */}
        <GlassyCard className={styles.statusCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-vote-yea" style={{ marginRight: '10px', color: '#8e44ad' }}></i>
            Model Selection Voting
          </h3>
          <div className={styles.statusGrid}>
            <div>
              <div className={styles.statusLabel}>Voting Period</div>
              <div className={styles.statusValue}>{formatTime(votingTimeLeft)}</div>
            </div>
            <div>
              <div className={styles.statusLabel}>Your Voting Power</div>
              <div className={styles.statusValue}>{userVotingPower} NRV</div>
            </div>
            <div>
              <div className={styles.statusLabel}>Total Votes Cast</div>
              <div className={styles.statusValue}>{models.reduce((sum, m) => sum + m.votes, 0)}</div>
            </div>
            <div>
              <div className={styles.statusLabel}>Leading Model</div>
              <div className={styles.statusValue}>{currentWinner.name}</div>
            </div>
          </div>
        </GlassyCard>

        {/* Model Candidates */}
        <div className={styles.candidatesGrid}>
          {models.map(model => (
            <GlassyCard key={model.id} className={styles.candidateCard}>
              <div className={styles.candidateHeader}>
                <div>
                  <h4 className={styles.candidateName}>{model.name}</h4>
                  <div className={styles.providerBadge}>{model.provider}</div>
                </div>
                <div className={styles.modelStats}>
                  <div className={styles.stat}>
                    <span className={styles.statLabel}>Performance</span>
                    <span className={styles.statValue}>{model.performance}</span>
                  </div>
                  <div className={styles.stat}>
                    <span className={styles.statLabel}>Cost</span>
                    <span className={styles.statValue}>{model.cost}</span>
                  </div>
                </div>
              </div>

              <p className={styles.candidateDescription}>{model.description}</p>

              <div className={styles.modelFeatures}>
                <h5>Key Features:</h5>
                <div className={styles.featuresList}>
                  {model.features.map((feature, index) => (
                    <span key={index} className={styles.featureTag}>{feature}</span>
                  ))}
                </div>
              </div>

              <div className={styles.votingSection}>
                <div className={styles.voteProgress}>
                  <div className={styles.progressBar}>
                    <div
                      className={styles.progressFill}
                      style={{ width: `${(model.votes / model.totalVotes) * 100}%` }}
                    ></div>
                  </div>
                  <div className={styles.voteCount}>
                    {model.votes} votes ({((model.votes / model.totalVotes) * 100).toFixed(1)}%)
                  </div>
                </div>

                <button
                  className={`${styles.voteButton} ${selectedModel === model.id ? styles.selected : ''} ${hasVoted ? styles.disabled : ''}`}
                  onClick={() => handleVote(model.id)}
                  disabled={hasVoted}
                >
                  {selectedModel === model.id ? '✓ Voted' :
                   hasVoted ? 'Vote Submitted' : 'Vote for This Model'}
                </button>
              </div>
            </GlassyCard>
          ))}
        </div>

        {/* Voting Rules & Information */}
        <div className={styles.infoGrid}>
          <GlassyCard className={styles.rulesCard}>
            <h3 className={styles.cardTitle}>
              <i className="fas fa-info-circle" style={{ marginRight: '10px', color: '#8e44ad' }}></i>
              Voting Rules
            </h3>
            <div className={styles.rulesList}>
              <div className={styles.rule}>
                <i className="fas fa-check-circle"></i>
                <span>Each NRV token equals 1 voting power</span>
              </div>
              <div className={styles.rule}>
                <i className="fas fa-check-circle"></i>
                <span>Winning model becomes active for 30 days</span>
              </div>
              <div className={styles.rule}>
                <i className="fas fa-check-circle"></i>
                <span>Votes are locked during the voting period</span>
              </div>
              <div className={styles.rule}>
                <i className="fas fa-check-circle"></i>
                <span>New voting periods occur every 25 days</span>
              </div>
            </div>
          </GlassyCard>

          <GlassyCard className={styles.impactCard}>
            <h3 className={styles.cardTitle}>
              <i className="fas fa-chart-line" style={{ marginRight: '10px', color: '#8e44ad' }}></i>
              Network Impact
            </h3>
            <div className={styles.impactStats}>
              <div className={styles.impactStat}>
                <div className={styles.impactLabel}>Avg Response Time</div>
                <div className={styles.impactValue}>~150ms</div>
              </div>
              <div className={styles.impactStat}>
                <div className={styles.impactLabel}>Daily Requests</div>
                <div className={styles.impactValue}>2.4M</div>
              </div>
              <div className={styles.impactStat}>
                <div className={styles.impactLabel}>Cost Savings</div>
                <div className={styles.impactValue}>$1,200/day</div>
              </div>
              <div className={styles.impactStat}>
                <div className={styles.impactLabel}>Success Rate</div>
                <div className={styles.impactValue}>96.8%</div>
              </div>
            </div>
          </GlassyCard>
        </div>
      </div>
    </PageLayout>
  );
}