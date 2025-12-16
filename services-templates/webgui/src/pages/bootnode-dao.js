import React, { useState, useEffect } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './bootnode-dao.module.css';

export default function BootnodeDAO() {
  const { activePage } = useNavigation('bootnode-dao');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedBootnode, setSelectedBootnode] = useState(null);
  const [hasVoted, setHasVoted] = useState(false);
  const [votingTimeLeft, setVotingTimeLeft] = useState(86400); // 24 hours in seconds
  const [userVotingPower, setUserVotingPower] = useState(100);

  // Mock bootnode data
  const bootnodes = [
    {
      id: 'bootnode-1',
      name: 'Primary Bootnode Alpha',
      location: 'North America',
      uptime: '99.9%',
      peers: 1250,
      latency: '12ms',
      description: 'High-performance bootnode with redundant infrastructure',
      votes: 245,
      totalVotes: 892
    },
    {
      id: 'bootnode-2',
      name: 'Secondary Bootnode Beta',
      location: 'Europe',
      uptime: '99.7%',
      peers: 980,
      latency: '18ms',
      description: 'Backup bootnode with geo-redundancy',
      votes: 198,
      totalVotes: 892
    },
    {
      id: 'bootnode-3',
      name: 'Tertiary Bootnode Gamma',
      location: 'Asia Pacific',
      uptime: '99.8%',
      peers: 756,
      latency: '25ms',
      description: 'Regional bootnode optimized for APAC connectivity',
      votes: 156,
      totalVotes: 892
    },
    {
      id: 'bootnode-4',
      name: 'Emergency Bootnode Delta',
      location: 'Global CDN',
      uptime: '99.5%',
      peers: 423,
      latency: '35ms',
      description: 'Distributed bootnode for emergency failover',
      votes: 293,
      totalVotes: 892
    }
  ];

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  const handleVote = (bootnodeId) => {
    if (hasVoted) return;

    setSelectedBootnode(bootnodeId);
    setHasVoted(true);
    // In a real app, this would submit the vote to the blockchain
    alert(`Vote submitted for ${bootnodes.find(b => b.id === bootnodeId)?.name}`);
  };

  // Countdown timer
  useEffect(() => {
    const timer = setInterval(() => {
      setVotingTimeLeft(prev => {
        if (prev <= 1) {
          clearInterval(timer);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(timer);
  }, []);

  const formatTime = (seconds) => {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  };

  const currentWinner = bootnodes.reduce((prev, current) =>
    (prev.votes > current.votes) ? prev : current
  );

  return (
    <PageLayout activePage={activePage} pageTitle="Bootnode DAO" onSearch={handleSearch}>
      <PageHeader
        title="Bootnode DAO Governance"
        subtitle="Vote for the best bootnode for network failover protection"
        titleColor="#4a90e2"
      />

      <div className={styles.daoContainer}>
        {/* Voting Status */}
        <GlassyCard className={styles.statusCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-clock" style={{ marginRight: '10px', color: '#4a90e2' }}></i>
            Voting Period Active
          </h3>
          <div className={styles.statusGrid}>
            <div>
              <div className={styles.statusLabel}>Time Remaining</div>
              <div className={styles.statusValue}>{formatTime(votingTimeLeft)}</div>
            </div>
            <div>
              <div className={styles.statusLabel}>Your Voting Power</div>
              <div className={styles.statusValue}>{userVotingPower} NRV</div>
            </div>
            <div>
              <div className={styles.statusLabel}>Total Votes Cast</div>
              <div className={styles.statusValue}>{bootnodes.reduce((sum, b) => sum + b.votes, 0)}</div>
            </div>
            <div>
              <div className={styles.statusLabel}>Current Leader</div>
              <div className={styles.statusValue}>{currentWinner.name}</div>
            </div>
          </div>
        </GlassyCard>

        {/* Bootnode Candidates */}
        <div className={styles.candidatesGrid}>
          {bootnodes.map(bootnode => (
            <GlassyCard key={bootnode.id} className={styles.candidateCard}>
              <div className={styles.candidateHeader}>
                <h4 className={styles.candidateName}>{bootnode.name}</h4>
                <div className={styles.locationBadge}>{bootnode.location}</div>
              </div>

              <div className={styles.candidateStats}>
                <div className={styles.stat}>
                  <span className={styles.statLabel}>Uptime</span>
                  <span className={styles.statValue}>{bootnode.uptime}</span>
                </div>
                <div className={styles.stat}>
                  <span className={styles.statLabel}>Peers</span>
                  <span className={styles.statValue}>{bootnode.peers.toLocaleString()}</span>
                </div>
                <div className={styles.stat}>
                  <span className={styles.statLabel}>Latency</span>
                  <span className={styles.statValue}>{bootnode.latency}</span>
                </div>
              </div>

              <p className={styles.candidateDescription}>{bootnode.description}</p>

              <div className={styles.votingSection}>
                <div className={styles.voteProgress}>
                  <div className={styles.progressBar}>
                    <div
                      className={styles.progressFill}
                      style={{ width: `${(bootnode.votes / bootnode.totalVotes) * 100}%` }}
                    ></div>
                  </div>
                  <div className={styles.voteCount}>
                    {bootnode.votes} votes ({((bootnode.votes / bootnode.totalVotes) * 100).toFixed(1)}%)
                  </div>
                </div>

                <button
                  className={`${styles.voteButton} ${selectedBootnode === bootnode.id ? styles.selected : ''} ${hasVoted ? styles.disabled : ''}`}
                  onClick={() => handleVote(bootnode.id)}
                  disabled={hasVoted}
                >
                  {selectedBootnode === bootnode.id ? '✓ Voted' :
                   hasVoted ? 'Vote Submitted' : 'Vote for This Bootnode'}
                </button>
              </div>
            </GlassyCard>
          ))}
        </div>

        {/* Voting Rules */}
        <GlassyCard className={styles.rulesCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-gavel" style={{ marginRight: '10px', color: '#4a90e2' }}></i>
            Voting Rules & Information
          </h3>
          <div className={styles.rulesList}>
            <div className={styles.rule}>
              <i className="fas fa-check-circle"></i>
              <span>Each NRV token equals 1 voting power</span>
            </div>
            <div className={styles.rule}>
              <i className="fas fa-check-circle"></i>
              <span>Votes are locked for the duration of the voting period</span>
            </div>
            <div className={styles.rule}>
              <i className="fas fa-check-circle"></i>
              <span>Winning bootnode becomes active for network failover</span>
            </div>
            <div className={styles.rule}>
              <i className="fas fa-check-circle"></i>
              <span>New voting periods occur every 30 days</span>
            </div>
            <div className={styles.rule}>
              <i className="fas fa-check-circle"></i>
              <span>Bootnode performance metrics are updated in real-time</span>
            </div>
          </div>
        </GlassyCard>
      </div>
    </PageLayout>
  );
}