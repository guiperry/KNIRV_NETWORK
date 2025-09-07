/**
 * Keyboard Shortcuts Help Component
 * Displays available keyboard shortcuts and navigation help
 */

import React, { useState } from 'react';
import { X, Keyboard, Search, Command } from 'lucide-react';
import { KeyboardShortcut, formatShortcut, groupShortcutsByCategory } from '../hooks/useKeyboardNavigation';

interface KeyboardShortcutsHelpProps {
  isOpen: boolean;
  onClose: () => void;
  shortcuts: KeyboardShortcut[];
  className?: string;
}

const KeyboardShortcutsHelp: React.FC<KeyboardShortcutsHelpProps> = ({
  isOpen,
  onClose,
  shortcuts,
  className = ''
}) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string>('All');

  if (!isOpen) return null;

  // Filter shortcuts based on search term
  const filteredShortcuts = shortcuts.filter(shortcut =>
    shortcut.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
    shortcut.key.toLowerCase().includes(searchTerm.toLowerCase()) ||
    (shortcut.category || 'General').toLowerCase().includes(searchTerm.toLowerCase())
  );

  // Group shortcuts by category
  const groupedShortcuts = groupShortcutsByCategory(filteredShortcuts);
  
  // Get all categories
  const categories = ['All', ...Object.keys(groupedShortcuts).sort()];
  
  // Filter by selected category
  const displayShortcuts = selectedCategory === 'All' 
    ? groupedShortcuts 
    : { [selectedCategory]: groupedShortcuts[selectedCategory] || [] };

  return (
    <div className={`fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4 ${className}`}>
      <div className="bg-gray-900 border border-gray-700 rounded-xl w-full max-w-4xl max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-700">
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 bg-blue-500/20 rounded-lg flex items-center justify-center border border-blue-500/20">
              <Keyboard className="w-5 h-5 text-blue-400" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-white">Keyboard Shortcuts</h2>
              <p className="text-sm text-gray-400">Navigate KNIRV Controller with keyboard</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-2 hover:bg-gray-700 rounded-lg text-gray-400 hover:text-white transition-all"
            aria-label="Close shortcuts help"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Search and Filter */}
        <div className="p-6 border-b border-gray-700 space-y-4">
          {/* Search */}
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input
              type="text"
              placeholder="Search shortcuts..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          {/* Category Filter */}
          <div className="flex flex-wrap gap-2">
            {categories.map(category => (
              <button
                key={category}
                onClick={() => setSelectedCategory(category)}
                className={`px-3 py-1 rounded-full text-sm font-medium transition-all ${
                  selectedCategory === category
                    ? 'bg-blue-500 text-white'
                    : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                }`}
              >
                {category}
              </button>
            ))}
          </div>
        </div>

        {/* Shortcuts List */}
        <div className="flex-1 overflow-y-auto p-6">
          {Object.keys(displayShortcuts).length === 0 ? (
            <div className="text-center py-12">
              <Keyboard className="w-12 h-12 text-gray-600 mx-auto mb-4" />
              <p className="text-gray-400">No shortcuts found matching your search.</p>
            </div>
          ) : (
            <div className="space-y-6">
              {Object.entries(displayShortcuts).map(([category, categoryShortcuts]) => (
                <div key={category}>
                  <h3 className="text-lg font-semibold text-white mb-4 flex items-center space-x-2">
                    <Command className="w-5 h-5 text-blue-400" />
                    <span>{category}</span>
                    <span className="text-sm text-gray-400 font-normal">
                      ({categoryShortcuts.length} shortcuts)
                    </span>
                  </h3>
                  
                  <div className="grid gap-3">
                    {categoryShortcuts.map((shortcut, index) => (
                      <div
                        key={`${category}-${index}`}
                        className="flex items-center justify-between p-3 bg-gray-800/50 border border-gray-700 rounded-lg hover:bg-gray-800/70 transition-all"
                      >
                        <div className="flex-1">
                          <p className="text-white font-medium">{shortcut.description}</p>
                        </div>
                        <div className="flex items-center space-x-1">
                          {formatShortcut(shortcut).split(' + ').map((key, keyIndex) => (
                            <React.Fragment key={keyIndex}>
                              {keyIndex > 0 && (
                                <span className="text-gray-400 text-sm">+</span>
                              )}
                              <kbd className="px-2 py-1 bg-gray-700 border border-gray-600 rounded text-sm font-mono text-gray-300">
                                {key}
                              </kbd>
                            </React.Fragment>
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-6 border-t border-gray-700 bg-gray-800/50">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm text-gray-400">
            <div>
              <h4 className="font-semibold text-white mb-2">Navigation</h4>
              <ul className="space-y-1">
                <li>• Use Tab/Shift+Tab to navigate</li>
                <li>• Arrow keys for directional movement</li>
                <li>• Enter/Space to activate</li>
              </ul>
            </div>
            <div>
              <h4 className="font-semibold text-white mb-2">Accessibility</h4>
              <ul className="space-y-1">
                <li>• Screen reader compatible</li>
                <li>• High contrast support</li>
                <li>• Focus indicators visible</li>
              </ul>
            </div>
            <div>
              <h4 className="font-semibold text-white mb-2">Tips</h4>
              <ul className="space-y-1">
                <li>• Press ? to open this help</li>
                <li>• Escape to close dialogs</li>
                <li>• Home/End for first/last item</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default KeyboardShortcutsHelp;
