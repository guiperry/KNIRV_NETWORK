import React, { useState } from 'react';
import { useNavigation } from '../../hooks/useNavigation';
import styles from './minerals.module.css';

// Import mineral images (you'll need to add these to your public folder)
// For now, we'll use placeholder URLs
const mineral1 = '/images/mineral1.jpg';
const mineral2 = '/images/mineral2.jpg';
const mineral3 = '/images/mineral3.jpg';

export default function MineralsDAO() {
  const { activePage, handleNavigation } = useNavigation('daos/minerals');

  // Sample mineral proposals with votes and unique IDs
  const [proposals, setProposals] = useState([
    {
      id: 1,
      title: 'Gold Mining Operation in Nevada',
      description:
        'Proposal to invest in a gold mining operation in Nevada with proven reserves.',
      image: mineral1,
      location: 'Nevada, USA',
      status: 'Voting',
      details: 'This operation has an estimated yield of 50,000 ounces over 5 years.',
      votes: { yes: 0, no: 0 },
    },
    {
      id: 2,
      title: 'Lithium Extraction Project',
      description:
        'Proposal to acquire rights to a lithium extraction project in South America.',
      image: mineral2,
      location: 'Chile, South America',
      status: 'Accepted',
      details: 'This project will tap into one of the largest lithium reserves in the world.',
      votes: { yes: 0, no: 0 },
    },
    {
      id: 3,
      title: 'Rare Earth Minerals Exploration',
      description:
        'Proposal to fund exploration for rare earth minerals in Southeast Asia.',
      image: mineral3,
      location: 'Myanmar, Southeast Asia',
      status: 'Rejected',
      details: 'The exploration will focus on neodymium and dysprosium deposits.',
      votes: { yes: 0, no: 0 },
    },
  ]);

  // State for the proposal submission form
  const [newProposal, setNewProposal] = useState({
    title: '',
    description: '',
    location: '',
    details: '',
  });

  // Function to handle voting
  const handleVote = (proposalId, voteType) => {
    setProposals((prevProposals) =>
      prevProposals.map((proposal) => {
        if (proposal.id === proposalId) {
          return {
            ...proposal,
            votes: {
              ...proposal.votes,
              [voteType]: proposal.votes[voteType] + 1,
            },
          };
        }
        return proposal;
      })
    );
  };

  // Function to handle form input changes
  const handleInputChange = (event) => {
    const { name, value } = event.target;
    setNewProposal((prevProposal) => ({
      ...prevProposal,
      [name]: value,
    }));
  };

  // Function to handle form submission
  const handleSubmit = (event) => {
    event.preventDefault();

    // Create a new proposal object
    const newProposalObject = {
      id: proposals.length + 1,
      title: newProposal.title,
      description: newProposal.description,
      image: mineral1, //default image
      location: newProposal.location,
      status: 'Voting', // Default status
      details: newProposal.details,
      votes: { yes: 0, no: 0 },
    };

    // Add the new proposal to the proposals array
    setProposals((prevProposals) => [...prevProposals, newProposalObject]);

    // Clear the form
    setNewProposal({
      title: '',
      description: '',
      location: '',
      details: '',
    });
  };

  return (
    <div className={styles.dashboardContainer}>
      {/* Sidebar */}
      <div className={styles.sidebar}>
        <h2 className={styles.dashboardTitle}>Blockchain Dashboard</h2>

        <div
          onClick={() => handleNavigation('inventory')}
          className={`${styles.navItem} ${styles.glassyContainer} ${activePage === 'inventory' ? styles.active : styles.inactive}`}
        >
          <span className={styles.navIcon}>📦</span>
          <span>Inventory</span>
        </div>

        <div
          onClick={() => handleNavigation('vault')}
          className={`${styles.navItem} ${styles.glassyContainer} ${activePage === 'vault' ? styles.active : styles.inactive}`}
        >
          <span className={styles.navIcon}>🔒</span>
          <span>Vault</span>
        </div>

        <div
          onClick={() => handleNavigation('blockchain')}
          className={`${styles.navItem} ${styles.glassyContainer} ${activePage === 'blockchain' ? styles.active : styles.inactive}`}
        >
          <span className={styles.navIcon}>⛓️</span>
          <span>Blockchain</span>
        </div>

        <div
          className={`${styles.navItem} ${styles.glassyContainer} ${styles.inactive}`}
          style={{ opacity: 0.5, cursor: 'not-allowed' }}
          title="DEX functionality has been moved to the Signifier app"
        >
          <span className={styles.navIcon}>💱</span>
          <span>DEX (Moved)</span>
        </div>

        <div
          onClick={() => handleNavigation('daos')}
          className={`${styles.navItem} ${styles.glassyContainer} ${activePage === 'daos' ? styles.active : styles.inactive}`}
        >
          <span className={styles.navIcon}>🏛️</span>
          <span>DAOs</span>
        </div>

        <div
          onClick={() => handleNavigation('settlement')}
          className={`${styles.navItem} ${styles.glassyContainer} ${activePage === 'settlement' ? styles.active : styles.inactive}`}
        >
          <span className={styles.navIcon}>📝</span>
          <span>Settlement</span>
        </div>
      </div>
      
      {/* Main Content */}
      <div className={styles.mainContent}>
        {/* Top Navigation */}
        <div className={`${styles.topNav} ${styles.glassyContainer}`}>
          <h3 className={styles.pageTitle}>Minerals DAO</h3>
          <div className={styles.userControls}>
            <span className={styles.controlIcon}>🔍</span>
            <span className={styles.controlIcon}>🔔</span>
            <span className={styles.controlIcon}>⚙️</span>
            <span className={styles.controlIcon}>👤</span>
          </div>
        </div>

        <div className={`${styles.daoHeader} ${styles.glassyContainer}`}>
          <div className={styles.daoIconLarge}>💎</div>
          <div className={styles.daoHeaderContent}>
            <h2 className={styles.daoTitle}>Minerals DAO</h2>
            <p className={styles.daoDescription}>
              Welcome to the Minerals DAO! Here you can review and vote on new
              proposals for mineral rights acquisition and mining operations.
            </p>
            <div className={styles.daoStats}>
              <div className={styles.statItem}>
                <span className={styles.statLabel}>Members</span>
                <span className={styles.statValue}>876</span>
              </div>
              <div className={styles.statItem}>
                <span className={styles.statLabel}>Operations</span>
                <span className={styles.statValue}>12</span>
              </div>
              <div className={styles.statItem}>
                <span className={styles.statLabel}>Total Value</span>
                <span className={styles.statValue}>$18.7M</span>
              </div>
              <div className={styles.statItem}>
                <span className={styles.statLabel}>Active Proposals</span>
                <span className={styles.statValue}>3</span>
              </div>
            </div>
          </div>
        </div>

        <h3 className={styles.sectionTitle}>Mineral Proposals</h3>
        <div className={styles.proposalsGrid}>
          {proposals.map((proposal) => (
            <div key={proposal.id} className={`${styles.proposalCard} ${styles.glassyContainer}`}>
              <div 
                className={styles.proposalImage} 
                style={{ backgroundImage: `url(${proposal.image})` }}
              >
                <div className={`${styles.proposalStatus} ${styles[proposal.status.toLowerCase()]}`}>
                  {proposal.status}
                </div>
              </div>
              <div className={styles.proposalInfo}>
                <h3 className={styles.proposalTitle}>{proposal.title}</h3>
                <p className={styles.proposalLocation}>{proposal.location}</p>
                <p className={styles.proposalDescription}>{proposal.description}</p>
                <p className={styles.proposalDetails}>{proposal.details}</p>
                <div className={styles.proposalVotes}>
                  <button 
                    className={`${styles.voteButton} ${styles.voteYes}`}
                    onClick={() => handleVote(proposal.id, 'yes')}
                  >
                    Yes ({proposal.votes.yes})
                  </button>
                  <button 
                    className={`${styles.voteButton} ${styles.voteNo}`}
                    onClick={() => handleVote(proposal.id, 'no')}
                  >
                    No ({proposal.votes.no})
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>

        <h3 className={styles.sectionTitle}>Submit a New Proposal</h3>
        <div className={`${styles.proposalForm} ${styles.glassyContainer}`}>
          <form onSubmit={handleSubmit}>
            <div className={styles.formGroup}>
              <label htmlFor="title">Title:</label>
              <input
                type="text"
                id="title"
                name="title"
                value={newProposal.title}
                onChange={handleInputChange}
                required
                className={styles.formInput}
              />
            </div>

            <div className={styles.formGroup}>
              <label htmlFor="description">Description:</label>
              <textarea
                id="description"
                name="description"
                value={newProposal.description}
                onChange={handleInputChange}
                required
                className={styles.formTextarea}
              />
            </div>

            <div className={styles.formGroup}>
              <label htmlFor="location">Location:</label>
              <input
                type="text"
                id="location"
                name="location"
                value={newProposal.location}
                onChange={handleInputChange}
                required
                className={styles.formInput}
              />
            </div>

            <div className={styles.formGroup}>
              <label htmlFor="details">Details:</label>
              <textarea
                id="details"
                name="details"
                value={newProposal.details}
                onChange={handleInputChange}
                required
                className={styles.formTextarea}
              />
            </div>

            <button type="submit" className={styles.submitButton}>Submit Proposal</button>
          </form>
        </div>

        <div className={styles.backLink}>
          <button 
            onClick={() => handleNavigation('daos')}
            className={styles.backButton}
          >
            ← Back to DAOs
          </button>
        </div>
      </div>
    </div>
  );
}