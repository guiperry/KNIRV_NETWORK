// Chat-Brain Page - Main Orchestrator Component
import React, { useState } from 'react';
import { Network, FileText, Trash2 } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { ChatBrainProvider } from '../contexts/ChatBrainContext';
import { ChatInterface } from '../components/chat-brain/ChatInterface';
import { MemoryGraphView } from '../components/chat-brain/MemoryGraphView';
import { NotesPanel } from '../components/chat-brain/NotesPanel';
import { LLMSelector } from '../components/chat-brain/LLMSelector';
import { useChatBrain } from '../contexts/ChatBrainContext';

// Burger Menu Component
interface BurgerMenuProps {
  isOpen: boolean;
  onToggle: () => void;
  children: React.ReactNode;
}

const BurgerMenu: React.FC<BurgerMenuProps> = ({ isOpen, onToggle, children }) => {
  return (
    <div className="relative">
      {/* Burger Button */}
      <button
        onClick={onToggle}
        className="bg-gray-800/80 hover:bg-gray-700/80 text-white p-3 rounded-lg shadow-lg transition-all duration-200 border border-gray-600/50 backdrop-blur-sm"
        aria-label="Navigation menu"
      >
        <div className="w-5 h-5 flex flex-col justify-center items-center">
          <div className={`w-5 h-0.5 bg-white transition-all duration-300 ${isOpen ? 'rotate-45 translate-y-1' : ''}`}></div>
          <div className={`w-5 h-0.5 bg-white transition-all duration-300 mt-1 ${isOpen ? 'opacity-0' : ''}`}></div>
          <div className={`w-5 h-0.5 bg-white transition-all duration-300 mt-1 ${isOpen ? '-rotate-45 -translate-y-1' : ''}`}></div>
        </div>
      </button>

      {/* Menu Dropdown */}
      {isOpen && (
        <div className="absolute top-full right-0 mt-2 bg-gray-800/90 backdrop-blur-sm border border-gray-600/50 rounded-lg shadow-xl min-w-48 z-50">
          <div className="p-2 space-y-1">
            {children}
          </div>
        </div>
      )}
    </div>
  );
};

// Menu Item Component
interface MenuItemProps {
  onClick: () => void;
  children: React.ReactNode;
  icon: string;
}

const MenuItem: React.FC<MenuItemProps> = ({ onClick, children, icon }) => {
  return (
    <button
      onClick={onClick}
      className="w-full text-left px-3 py-2 rounded-md text-white hover:bg-gray-700/80 transition-all duration-200 flex items-center space-x-2"
    >
      <span className="text-lg">{icon}</span>
      <span className="font-medium">{children}</span>
    </button>
  );
};

function ChatBrainContent() {
  const navigate = useNavigate();
  const { clearChat } = useChatBrain();
  const [showMemoryGraph, setShowMemoryGraph] = useState(false);
  const [showNotes, setShowNotes] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);

  const handleClearChat = () => {
    if (confirm('Are you sure you want to clear the chat history?')) {
      clearChat();
    }
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white flex flex-col relative">
      {/* Burger Menu Navigation */}
      <div className="absolute top-4 right-4 z-50">
        <BurgerMenu isOpen={menuOpen} onToggle={() => setMenuOpen(!menuOpen)}>
          <MenuItem onClick={() => { navigate('/manager/skills'); setMenuOpen(false); }} icon="⚡">
            Skills
          </MenuItem>
          <MenuItem onClick={() => { navigate('/manager/udc'); setMenuOpen(false); }} icon="🔐">
            UDC
          </MenuItem>
          <MenuItem onClick={() => { navigate('/manager/wallet'); setMenuOpen(false); }} icon="💰">
            Wallet
          </MenuItem>
          <MenuItem onClick={() => { navigate('/'); setMenuOpen(false); }} icon="🏠">
            Input Interface
          </MenuItem>
        </BurgerMenu>
      </div>

      {/* Header */}
      <div className="bg-gray-800/80 backdrop-blur-sm border-b border-gray-700/50 p-4">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          {/* Left: Title and LLM Selector */}
          <div className="flex items-center space-x-4">
            <h1 className="text-2xl font-bold text-white flex items-center space-x-2">
              <span>🧠</span>
              <span>Chat-Brain</span>
            </h1>
            <LLMSelector />
          </div>

          {/* Right: Action Buttons */}
          <div className="flex items-center space-x-2 mr-16">
            <button
              onClick={() => setShowMemoryGraph(!showMemoryGraph)}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors flex items-center space-x-2 ${
                showMemoryGraph
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-700 hover:bg-gray-600 text-gray-300'
              }`}
            >
              <Network size={16} />
              <span>Memory Graph</span>
            </button>

            <button
              onClick={() => setShowNotes(!showNotes)}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors flex items-center space-x-2 ${
                showNotes
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-700 hover:bg-gray-600 text-gray-300'
              }`}
            >
              <FileText size={16} />
              <span>Notes</span>
            </button>

            <button
              onClick={handleClearChat}
              className="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-gray-300 rounded-lg text-sm font-medium transition-colors flex items-center space-x-2"
            >
              <Trash2 size={16} />
              <span>Clear Chat</span>
            </button>
          </div>
        </div>
      </div>

      {/* Main Chat Interface */}
      <ChatInterface />

      {/* Side Panels */}
      {showMemoryGraph && <MemoryGraphView onClose={() => setShowMemoryGraph(false)} />}
      {showNotes && <NotesPanel onClose={() => setShowNotes(false)} />}
    </div>
  );
}

export default function ChatBrain() {
  return (
    <ChatBrainProvider>
      <ChatBrainContent />
    </ChatBrainProvider>
  );
}
