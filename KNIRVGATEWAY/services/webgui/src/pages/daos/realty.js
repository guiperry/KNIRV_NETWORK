import React, { useState } from 'react';
import { useNavigation } from '../../hooks/useNavigation';
import styles from './realty.module.css';

// Import property images (you'll need to add these to your public folder)
// For now, we'll use placeholder URLs
const property1 = '/images/property1.jpg';
const property2 = '/images/property2.jpg';
const property3 = '/images/property3.jpg';

export default function RealtyDAO() {
  const { activePage, handleNavigation } = useNavigation('daos/realty');

  // Sample property proposals with votes and unique IDs
  const [proposals, setProposals] = useState([
    {
      id: 1,
      title: 'Invest in Downtown Loft Conversion',
      description:
        'Proposal to invest in converting an underutilized downtown commercial space into modern lofts.',
      image: property1,
      location: 'Downtown, City Center',
      status: 'Voting',
      details: 'This is a 3 story building that we are looking to convert to housing.',
      votes: { yes: 0, no: 0 },
    },
    {
      id: 2,
      title: 'Acquire Suburban Family Homes',
      description:
        'Proposal to acquire a portfolio of single-family homes in the growing suburban area.',
      image: property2,
      location: 'Suburbs, Green Meadows',
      status: 'Accepted',
      details: 'This is a nice quite neighborhood that would make for good housing.',
      votes: { yes: 0, no: 0 },
    },
    {
      id: 3,
      title: 'Develop Eco-Friendly Apartments',
      description:
        'Proposal to develop a new eco-friendly apartment complex with sustainable features.',
      image: property3,
      location: 'Uptown, Eco District',
      status: 'Rejected',
      details: 'This building will include solar, and water re-use.',
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
      image: property1, //default image
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
          <h3 className={styles.pageTitle}>Realty DAO</h3>
          <div className={styles.userControls}>
            <span className={styles.controlIcon}>🔍</span>
            <span className={styles.controlIcon}>🔔</span>
            <span className={styles.controlIcon}>⚙️</span>
            <span className={styles.controlIcon}>👤</span>
          </div>
        </div>

        <div className={`${styles.daoHeader} ${styles.glassyContainer}`}>
          <div className={styles.daoIconLarge}>🏢</div>
          <div className={styles.daoHeaderContent}>
            <h2 className={styles.daoTitle}>Realty DAO</h2>
            <p className={styles.daoDescription}>
              Welcome to the Realty DAO! Here you can review and vote on new
              proposals to add to our real estate portfolio.
            </p>
            <div className={styles.daoStats}>
              <div className={styles.statItem}>
                <span className={styles.statLabel}>Members</span>
                <span className={styles.statValue}>1,245</span>
              </div>
              <div className={styles.statItem}>
                <span className={styles.statLabel}>Properties</span>
                <span className={styles.statValue}>37</span>
              </div>
              <div className={styles.statItem}>
                <span className={styles.statLabel}>Total Value</span>
                <span className={styles.statValue}>$24.5M</span>
              </div>
              <div className={styles.statItem}>
                <span className={styles.statLabel}>Active Proposals</span>
                <span className={styles.statValue}>3</span>
              </div>
            </div>
          </div>
        </div>

        <h3 className={styles.sectionTitle}>Property Proposals</h3>
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