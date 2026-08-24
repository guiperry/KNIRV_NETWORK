import React, { useState, useEffect } from 'react';
import { Vote, Users, Clock, CheckCircle, XCircle, TrendingUp, Award } from 'lucide-react';
import { useAuth } from '../AuthContext';

interface Proposal {
  id: string;
  title: string;
  description: string;
  type: 'model_addition' | 'model_removal' | 'parameter_change' | 'governance';
  status: 'active' | 'passed' | 'rejected' | 'pending';
  votesFor: number;
  votesAgainst: number;
  totalVotes: number;
  quorum: number;
  endTime: Date;
  proposer: string;
  createdAt: Date;
}

interface VotingStats {
  totalProposals: number;
  activeProposals: number;
  userVotingPower: number;
  participationRate: number;
}

const DAOVoting: React.FC = () => {
  const { user } = useAuth();
  const [proposals, setProposals] = useState<Proposal[]>([]);
  const [stats, setStats] = useState<VotingStats>({
    totalProposals: 0,
    activeProposals: 0,
    userVotingPower: 0,
    participationRate: 0
  });
  const [selectedTab, setSelectedTab] = useState<'active' | 'completed'>('active');
  const [userVotes, setUserVotes] = useState<Record<string, 'for' | 'against'>>({});

  useEffect(() => {
    // Load proposals and voting data
    const mockProposals: Proposal[] = [
      {
        id: 'prop-1',
        title: 'Add GPT-4 Turbo to Shared Model Pool',
        description: 'Proposal to add OpenAI GPT-4 Turbo model to the shared KNIRVCORTEX model pool for community use.',
        type: 'model_addition',
        status: 'active',
        votesFor: 847,
        votesAgainst: 123,
        totalVotes: 970,
        quorum: 1000,
        endTime: new Date(Date.now() + 86400000 * 2), // 2 days from now
        proposer: 'alice.knirv',
        createdAt: new Date(Date.now() - 86400000 * 3)
      },
      {
        id: 'prop-2',
        title: 'Increase Model Access Rate Limits',
        description: 'Proposal to increase the rate limits for shared model access from 100 to 200 requests per hour.',
        type: 'parameter_change',
        status: 'active',
        votesFor: 234,
        votesAgainst: 567,
        totalVotes: 801,
        quorum: 1000,
        endTime: new Date(Date.now() + 86400000 * 5), // 5 days from now
        proposer: 'bob.knirv',
        createdAt: new Date(Date.now() - 86400000 * 1)
      },
      {
        id: 'prop-3',
        title: 'Remove Deprecated Model v1.0',
        description: 'Proposal to remove the deprecated legacy model v1.0 from the shared pool to free up resources.',
        type: 'model_removal',
        status: 'passed',
        votesFor: 1234,
        votesAgainst: 89,
        totalVotes: 1323,
        quorum: 1000,
        endTime: new Date(Date.now() - 86400000 * 1), // 1 day ago
        proposer: 'charlie.knirv',
        createdAt: new Date(Date.now() - 86400000 * 7)
      },
      {
        id: 'prop-4',
        title: 'Update Governance Voting Period',
        description: 'Proposal to extend the voting period for governance proposals from 7 days to 14 days.',
        type: 'governance',
        status: 'rejected',
        votesFor: 456,
        votesAgainst: 1098,
        totalVotes: 1554,
        quorum: 1000,
        endTime: new Date(Date.now() - 86400000 * 3), // 3 days ago
        proposer: 'diana.knirv',
        createdAt: new Date(Date.now() - 86400000 * 10)
      }
    ];

    setProposals(mockProposals);
    
    const activeProposals = mockProposals.filter(p => p.status === 'active').length;
    
    setStats({
      totalProposals: mockProposals.length,
      activeProposals,
      userVotingPower: 125, // User's voting power based on stake
      participationRate: 78.5
    });
  }, []);

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active':
        return <Clock className="w-4 h-4 text-yellow-500" />;
      case 'passed':
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'rejected':
        return <XCircle className="w-4 h-4 text-red-500" />;
      default:
        return <Clock className="w-4 h-4 text-slate-500" />;
    }
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'model_addition':
        return 'text-green-400 bg-green-500/20';
      case 'model_removal':
        return 'text-red-400 bg-red-500/20';
      case 'parameter_change':
        return 'text-blue-400 bg-blue-500/20';
      case 'governance':
        return 'text-purple-400 bg-purple-500/20';
      default:
        return 'text-slate-400 bg-slate-500/20';
    }
  };

  const handleVote = (proposalId: string, vote: 'for' | 'against') => {
    setUserVotes(prev => ({ ...prev, [proposalId]: vote }));
    
    // Update proposal vote counts
    setProposals(prev => prev.map(p => {
      if (p.id === proposalId) {
        const currentUserVote = userVotes[proposalId];
        let newVotesFor = p.votesFor;
        let newVotesAgainst = p.votesAgainst;
        
        // Remove previous vote if exists
        if (currentUserVote === 'for') newVotesFor -= stats.userVotingPower;
        if (currentUserVote === 'against') newVotesAgainst -= stats.userVotingPower;
        
        // Add new vote
        if (vote === 'for') newVotesFor += stats.userVotingPower;
        if (vote === 'against') newVotesAgainst += stats.userVotingPower;
        
        return {
          ...p,
          votesFor: newVotesFor,
          votesAgainst: newVotesAgainst,
          totalVotes: newVotesFor + newVotesAgainst
        };
      }
      return p;
    }));
  };

  const getVotePercentage = (votes: number, total: number) => {
    return total > 0 ? (votes / total) * 100 : 0;
  };

  const filteredProposals = proposals.filter(p => 
    selectedTab === 'active' ? p.status === 'active' : p.status !== 'active'
  );

  const formatTimeRemaining = (endTime: Date) => {
    const now = new Date();
    const diff = endTime.getTime() - now.getTime();
    
    if (diff <= 0) return 'Ended';
    
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    
    if (days > 0) return `${days}d ${hours}h remaining`;
    return `${hours}h remaining`;
  };

  return (
    <div className="h-full bg-slate-900 p-6">
      {/* Header */}
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-purple-500/20 rounded-lg">
          <Vote className="w-6 h-6 text-purple-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">DAO KNIRVCORTEX Voting</h1>
          <p className="text-slate-400">Participate in shared model governance and decision making</p>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center space-x-2">
            <Vote className="w-5 h-5 text-purple-400" />
            <span className="text-sm text-slate-400">Total Proposals</span>
          </div>
          <div className="text-2xl font-bold text-white mt-2">{stats.totalProposals}</div>
        </div>
        
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center space-x-2">
            <Clock className="w-5 h-5 text-yellow-400" />
            <span className="text-sm text-slate-400">Active</span>
          </div>
          <div className="text-2xl font-bold text-white mt-2">{stats.activeProposals}</div>
        </div>
        
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center space-x-2">
            <Award className="w-5 h-5 text-blue-400" />
            <span className="text-sm text-slate-400">Voting Power</span>
          </div>
          <div className="text-2xl font-bold text-white mt-2">{stats.userVotingPower}</div>
        </div>
        
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center space-x-2">
            <TrendingUp className="w-5 h-5 text-green-400" />
            <span className="text-sm text-slate-400">Participation</span>
          </div>
          <div className="text-2xl font-bold text-white mt-2">{stats.participationRate}%</div>
        </div>
      </div>

      {/* Tab Navigation */}
      <div className="flex space-x-2 mb-6">
        <button
          onClick={() => setSelectedTab('active')}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
            selectedTab === 'active'
              ? 'bg-purple-600/30 text-purple-300 border border-purple-500/50'
              : 'bg-slate-700/30 text-slate-400 hover:bg-slate-600/30 hover:text-slate-300'
          }`}
        >
          Active Proposals
        </button>
        <button
          onClick={() => setSelectedTab('completed')}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
            selectedTab === 'completed'
              ? 'bg-purple-600/30 text-purple-300 border border-purple-500/50'
              : 'bg-slate-700/30 text-slate-400 hover:bg-slate-600/30 hover:text-slate-300'
          }`}
        >
          Completed Proposals
        </button>
      </div>

      {/* Proposals List */}
      <div className="space-y-4">
        {filteredProposals.map((proposal) => (
          <div key={proposal.id} className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
            <div className="flex items-start justify-between mb-4">
              <div className="flex-1">
                <div className="flex items-center space-x-3 mb-2">
                  <h3 className="text-lg font-semibold text-white">{proposal.title}</h3>
                  <span className={`px-2 py-1 rounded text-xs font-medium ${getTypeColor(proposal.type)}`}>
                    {proposal.type.replace('_', ' ')}
                  </span>
                  <div className="flex items-center space-x-1">
                    {getStatusIcon(proposal.status)}
                    <span className="text-sm text-slate-300 capitalize">{proposal.status}</span>
                  </div>
                </div>
                <p className="text-slate-400 mb-3">{proposal.description}</p>
                <div className="text-xs text-slate-500">
                  Proposed by {proposal.proposer} • {proposal.createdAt.toLocaleDateString()}
                </div>
              </div>
              
              {proposal.status === 'active' && (
                <div className="text-right">
                  <div className="text-sm text-slate-400 mb-2">
                    {formatTimeRemaining(proposal.endTime)}
                  </div>
                  <div className="flex space-x-2">
                    <button
                      onClick={() => handleVote(proposal.id, 'for')}
                      className={`px-3 py-1 rounded text-sm transition-colors ${
                        userVotes[proposal.id] === 'for'
                          ? 'bg-green-600 text-white'
                          : 'bg-slate-600 text-slate-300 hover:bg-green-600/50'
                      }`}
                    >
                      For
                    </button>
                    <button
                      onClick={() => handleVote(proposal.id, 'against')}
                      className={`px-3 py-1 rounded text-sm transition-colors ${
                        userVotes[proposal.id] === 'against'
                          ? 'bg-red-600 text-white'
                          : 'bg-slate-600 text-slate-300 hover:bg-red-600/50'
                      }`}
                    >
                      Against
                    </button>
                  </div>
                </div>
              )}
            </div>
            
            {/* Vote Progress */}
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-slate-400">
                  For: {proposal.votesFor.toLocaleString()} ({getVotePercentage(proposal.votesFor, proposal.totalVotes).toFixed(1)}%)
                </span>
                <span className="text-slate-400">
                  Against: {proposal.votesAgainst.toLocaleString()} ({getVotePercentage(proposal.votesAgainst, proposal.totalVotes).toFixed(1)}%)
                </span>
              </div>
              
              <div className="w-full bg-slate-700 rounded-full h-2">
                <div className="flex h-2 rounded-full overflow-hidden">
                  <div 
                    className="bg-green-500"
                    style={{ width: `${getVotePercentage(proposal.votesFor, proposal.totalVotes)}%` }}
                  ></div>
                  <div 
                    className="bg-red-500"
                    style={{ width: `${getVotePercentage(proposal.votesAgainst, proposal.totalVotes)}%` }}
                  ></div>
                </div>
              </div>
              
              <div className="flex justify-between text-xs text-slate-500">
                <span>Total votes: {proposal.totalVotes.toLocaleString()}</span>
                <span>Quorum: {proposal.quorum.toLocaleString()}</span>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default DAOVoting;
