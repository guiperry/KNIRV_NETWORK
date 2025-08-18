import React, { useState } from 'react';
import { 
  Search, 
  Filter, 
  Grid, 
  List, 
  Plus,
  ExternalLink,
  Zap,
  Eye,
  MoreVertical
} from 'lucide-react';

export const NFTManager = () => {
  const [viewMode, setViewMode] = useState('grid');
  const [searchTerm, setSearchTerm] = useState('');

  const nfts = [
    {
      id: 1,
      name: 'CryptoPunk #7804',
      collection: 'CryptoPunks',
      image: 'https://images.pexels.com/photos/5380664/pexels-photo-5380664.jpeg?auto=compress&cs=tinysrgb&w=400',
      lastInference: '2 hours ago',
      inferences: 24,
      capabilities: ['Image Analysis', 'Style Transfer', 'Metadata Extraction'],
      rarity: 'Legendary',
      value: '45.2 ETH'
    },
    {
      id: 2,
      name: 'Bored Ape #3749',
      collection: 'BAYC',
      image: 'https://images.pexels.com/photos/5380617/pexels-photo-5380617.jpeg?auto=compress&cs=tinysrgb&w=400',
      lastInference: '5 minutes ago',
      inferences: 18,
      capabilities: ['Sentiment Analysis', 'Image Classification', 'Color Palette'],
      rarity: 'Rare',
      value: '12.8 ETH'
    },
    {
      id: 3,
      name: 'Art Blocks #182',
      collection: 'Art Blocks Curated',
      image: 'https://images.pexels.com/photos/5380613/pexels-photo-5380613.jpeg?auto=compress&cs=tinysrgb&w=400',
      lastInference: '1 hour ago',
      inferences: 31,
      capabilities: ['Pattern Recognition', 'Generative Analysis', 'Algorithm Detection'],
      rarity: 'Epic',
      value: '8.7 ETH'
    },
    {
      id: 4,
      name: 'Azuki #7894',
      collection: 'Azuki',
      image: 'https://images.pexels.com/photos/5380665/pexels-photo-5380665.jpeg?auto=compress&cs=tinysrgb&w=400',
      lastInference: '30 minutes ago',
      inferences: 12,
      capabilities: ['Anime Style Analysis', 'Character Recognition', 'Art Style'],
      rarity: 'Uncommon',
      value: '3.2 ETH'
    },
    {
      id: 5,
      name: 'Moonbirds #1256',
      collection: 'Moonbirds',
      image: 'https://images.pexels.com/photos/5380668/pexels-photo-5380668.jpeg?auto=compress&cs=tinysrgb&w=400',
      lastInference: '3 hours ago',
      inferences: 7,
      capabilities: ['Trait Analysis', 'Background Removal', 'Style Classification'],
      rarity: 'Common',
      value: '2.1 ETH'
    },
    {
      id: 6,
      name: 'Doodles #4523',
      collection: 'Doodles',
      image: 'https://images.pexels.com/photos/5380671/pexels-photo-5380671.jpeg?auto=compress&cs=tinysrgb&w=400',
      lastInference: '1 day ago',
      inferences: 45,
      capabilities: ['Color Analysis', 'Hand-drawn Detection', 'Whimsical Classification'],
      rarity: 'Rare',
      value: '4.9 ETH'
    }
  ];

  const getRarityColor = (rarity) => {
    switch (rarity) {
      case 'Legendary':
        return 'from-yellow-400 to-orange-500';
      case 'Epic':
        return 'from-purple-400 to-pink-500';
      case 'Rare':
        return 'from-blue-400 to-indigo-500';
      case 'Uncommon':
        return 'from-green-400 to-emerald-500';
      default:
        return 'from-gray-400 to-gray-500';
    }
  };

  const filteredNFTs = nfts.filter(nft =>
    nft.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    nft.collection.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-3xl font-bold text-white mb-2">NFT Asset Manager</h1>
          <p className="text-slate-400">Manage and analyze your NFT collection with AI-powered insights.</p>
        </div>
        <div className="mt-4 lg:mt-0">
          <button className="bg-gradient-to-r from-purple-500 to-blue-500 text-white px-6 py-3 rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 flex items-center space-x-2">
            <Plus className="w-5 h-5" />
            <span>Import NFTs</span>
          </button>
        </div>
      </div>

      {/* Controls */}
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between space-y-4 lg:space-y-0">
        <div className="flex items-center space-x-4">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400 w-5 h-5" />
            <input
              type="text"
              placeholder="Search NFTs..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="bg-slate-800/50 border border-slate-700/50 rounded-lg pl-10 pr-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent w-64"
            />
          </div>
          <button className="bg-slate-800/50 border border-slate-700/50 rounded-lg px-4 py-2 text-slate-300 hover:text-white hover:border-slate-600/50 transition-all duration-200 flex items-center space-x-2">
            <Filter className="w-5 h-5" />
            <span>Filter</span>
          </button>
        </div>
        
        <div className="flex items-center space-x-2">
          <button
            onClick={() => setViewMode('grid')}
            className={`p-2 rounded-lg transition-all duration-200 ${
              viewMode === 'grid' 
                ? 'bg-purple-500/20 text-purple-400 border border-purple-500/30' 
                : 'bg-slate-800/50 text-slate-400 border border-slate-700/50 hover:text-white'
            }`}
          >
            <Grid className="w-5 h-5" />
          </button>
          <button
            onClick={() => setViewMode('list')}
            className={`p-2 rounded-lg transition-all duration-200 ${
              viewMode === 'list' 
                ? 'bg-purple-500/20 text-purple-400 border border-purple-500/30' 
                : 'bg-slate-800/50 text-slate-400 border border-slate-700/50 hover:text-white'
            }`}
          >
            <List className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* NFT Grid */}
      {viewMode === 'grid' ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredNFTs.map((nft) => (
            <div key={nft.id} className="bg-slate-800/50 backdrop-blur-sm rounded-xl border border-slate-700/50 hover:border-slate-600/50 transition-all duration-300 group overflow-hidden">
              <div className="relative overflow-hidden">
                <img 
                  src={nft.image} 
                  alt={nft.name}
                  className="w-full h-48 object-cover group-hover:scale-105 transition-transform duration-300"
                />
                <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                <div className="absolute top-3 right-3">
                  <span className={`px-2 py-1 rounded-full text-xs font-medium bg-gradient-to-r ${getRarityColor(nft.rarity)} text-white`}>
                    {nft.rarity}
                  </span>
                </div>
                <div className="absolute bottom-3 right-3 flex space-x-2 opacity-0 group-hover:opacity-100 transition-opacity duration-300">
                  <button className="p-2 bg-black/50 backdrop-blur-sm rounded-lg text-white hover:bg-black/70 transition-colors duration-200">
                    <Eye className="w-4 h-4" />
                  </button>
                  <button className="p-2 bg-black/50 backdrop-blur-sm rounded-lg text-white hover:bg-black/70 transition-colors duration-200">
                    <ExternalLink className="w-4 h-4" />
                  </button>
                </div>
              </div>
              
              <div className="p-5 space-y-4">
                <div>
                  <h3 className="text-lg font-bold text-white mb-1">{nft.name}</h3>
                  <p className="text-slate-400 text-sm">{nft.collection}</p>
                </div>
                
                <div className="flex items-center justify-between text-sm">
                  <span className="text-slate-400">Value:</span>
                  <span className="text-white font-medium">{nft.value}</span>
                </div>
                
                <div className="flex items-center justify-between text-sm">
                  <span className="text-slate-400">Inferences:</span>
                  <span className="text-white font-medium">{nft.inferences}</span>
                </div>
                
                <div className="space-y-2">
                  <p className="text-slate-400 text-sm">Capabilities:</p>
                  <div className="flex flex-wrap gap-1">
                    {nft.capabilities.slice(0, 2).map((capability, index) => (
                      <span key={index} className="px-2 py-1 bg-slate-700/50 text-slate-300 text-xs rounded-full">
                        {capability}
                      </span>
                    ))}
                    {nft.capabilities.length > 2 && (
                      <span className="px-2 py-1 bg-slate-700/50 text-slate-300 text-xs rounded-full">
                        +{nft.capabilities.length - 2} more
                      </span>
                    )}
                  </div>
                </div>
                
                <button className="w-full bg-gradient-to-r from-purple-500/20 to-blue-500/20 hover:from-purple-500/30 hover:to-blue-500/30 border border-purple-500/30 text-white py-2 px-4 rounded-lg transition-all duration-200 flex items-center justify-center space-x-2">
                  <Zap className="w-4 h-4" />
                  <span>Run Inference</span>
                </button>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl border border-slate-700/50 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-slate-700/30 border-b border-slate-600/50">
                <tr>
                  <th className="text-left py-4 px-6 text-slate-300 font-medium">NFT</th>
                  <th className="text-left py-4 px-6 text-slate-300 font-medium">Collection</th>
                  <th className="text-left py-4 px-6 text-slate-300 font-medium">Value</th>
                  <th className="text-left py-4 px-6 text-slate-300 font-medium">Inferences</th>
                  <th className="text-left py-4 px-6 text-slate-300 font-medium">Last Inference</th>
                  <th className="text-left py-4 px-6 text-slate-300 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700/50">
                {filteredNFTs.map((nft) => (
                  <tr key={nft.id} className="hover:bg-slate-700/20 transition-colors duration-200">
                    <td className="py-4 px-6">
                      <div className="flex items-center space-x-3">
                        <img src={nft.image} alt={nft.name} className="w-12 h-12 rounded-lg object-cover" />
                        <div>
                          <p className="text-white font-medium">{nft.name}</p>
                          <span className={`px-2 py-1 rounded-full text-xs font-medium bg-gradient-to-r ${getRarityColor(nft.rarity)} text-white`}>
                            {nft.rarity}
                          </span>
                        </div>
                      </div>
                    </td>
                    <td className="py-4 px-6 text-slate-300">{nft.collection}</td>
                    <td className="py-4 px-6 text-white font-medium">{nft.value}</td>
                    <td className="py-4 px-6 text-slate-300">{nft.inferences}</td>
                    <td className="py-4 px-6 text-slate-300">{nft.lastInference}</td>
                    <td className="py-4 px-6">
                      <div className="flex items-center space-x-2">
                        <button className="p-2 text-slate-400 hover:text-white transition-colors duration-200">
                          <Zap className="w-4 h-4" />
                        </button>
                        <button className="p-2 text-slate-400 hover:text-white transition-colors duration-200">
                          <Eye className="w-4 h-4" />
                        </button>
                        <button className="p-2 text-slate-400 hover:text-white transition-colors duration-200">
                          <MoreVertical className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
};