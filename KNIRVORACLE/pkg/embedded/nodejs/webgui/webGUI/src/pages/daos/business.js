import React, { useState } from 'react';
import { useNavigation } from '../../hooks/useNavigation';
import styles from './business.module.css';

// Import business images (you'll need to add these to your public folder)
// For now, we'll use placeholder URLs
const business1 = '/images/business1.jpg';
const business2 = '/images/business2.jpg';
const business3 = '/images/business3.jpg';

export default function BusinessDAO() {
  const { activePage, handleNavigation } = useNavigation('daos/business');

  // Sample business proposals with votes and unique IDs
  const [proposals, setProposals] = useState([
    {
      id: 1,
      title: 'Tech Startup Investment Opportunity',
      description:
        'Proposal to invest in a promising AI-driven analytics startup with strong growth potential.',
      image: business1,
      location: 'San Francisco, CA',
      status: 'Voting',
      details: 'The startup has a proven MVP and early customer traction with Fortune 500 companies.',
      votes: { yes: 0, no: 0 },
    },
    {
      id: 2,
      title: 'Sustainable Food Production Chain',
      description:
        'Proposal to acquire a controlling stake in a vertical farming operation with established distribution.',
      image: business2,
      location: 'Chicago, IL',
      status: 'Accepted',
      details: 'The business has been profitable for 3 years and is looking to expand operations.',
      votes: { yes: 0, no: 0 },
    },
    {
      id: 3,
      title: 'Renewable Energy Infrastructure',
      description:
        'Proposal to fund a solar panel installation business with government contracts.',
      image: business3,
      location: 'Austin, TX',
      status: 'Rejected',
      details: 'The company has secured tax incentives and has a 5-year growth plan.',
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
      image: business1, //default image
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
          <h3 className={styles.pageTitle}>Business DAO</h3>
          <div className={styles.userControls}>
            <span className={styles.controlIcon}>🔍</span>
            <span className={styles.controlIcon}>🔔</span>
            <span className={styles.controlIcon}>⚙️</span>
            <span className={styles.controlIcon}>👤</span>
          </div>
        </div>

        <div className={`${styles.daoHeader} ${styles.glassyContainer}`}>
          <div className={styles.daoIconLarge}>💼</div>
          <div className={styles.daoHeaderContent}>
            <h2 className={styles.daoTitle}>Business DAO</h2>
            <p className={styles.daoDescription}>
              Welcome to the Business DAO! Here you can review and vote on new
              proposals for business investments and acquisitions.
            </p>
            <div className={styles.daoStats}>
              <div className={styles.statItem}>
                <span className={styles.statLabel}>Members</span>
                <span className={styles.statValue}>1,532</span>
              </div>
              <div className={styles.statItem}>
                <span className={styles.statLabel}>Businesses</span>
                <span className={styles.statValue}>21</span>
              </div>
              <div className={styles.statItem}>
                <span className={styles.statLabel}>Total Value</span>
                <span className={styles.statValue}>$32.8M</span>
              </div>
              <div className={styles.statItem}>
                <span className={styles.statLabel}>Active Proposals</span>
                <span className={styles.statValue}>3</span>
              </div>
            </div>
          </div>
        </div>

        <h3 className={styles.sectionTitle}>Business Proposals</h3>
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