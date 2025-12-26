// Memory Graph Visualization Component
import React, { useEffect, useRef } from 'react';
import { X, RefreshCw } from 'lucide-react';
import cytoscape from 'cytoscape';
import { useChatBrain } from '../../contexts/ChatBrainContext';

interface MemoryGraphViewProps {
  onClose: () => void;
}

export function MemoryGraphView({ onClose }: MemoryGraphViewProps) {
  const { memoryNodes, memoryEdges, refreshMemoryGraph } = useChatBrain();
  const containerRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<cytoscape.Core | null>(null);

  useEffect(() => {
    refreshMemoryGraph();
  }, [refreshMemoryGraph]);

  useEffect(() => {
    if (!containerRef.current || memoryNodes.length === 0) return;

    // Destroy existing instance
    if (cyRef.current) {
      cyRef.current.destroy();
    }

    // Initialize Cytoscape
    cyRef.current = cytoscape({
      container: containerRef.current,
      elements: [
        // Nodes
        ...memoryNodes.map((node) => ({
          data: {
            id: node.id,
            label: node.label,
            type: node.type,
          },
        })),
        // Edges
        ...memoryEdges.map((edge) => ({
          data: {
            id: edge.id || `${edge.sourceId}-${edge.targetId}`,
            source: edge.sourceId,
            target: edge.targetId,
            label: edge.relation,
          },
        })),
      ],
      style: [
        {
          selector: 'node',
          style: {
            'background-color': '#3B82F6',
            label: 'data(label)',
            color: '#fff',
            'text-valign': 'center',
            'text-halign': 'center',
            'font-size': '12px',
            width: 40,
            height: 40,
          },
        },
        {
          selector: 'node[type="entity"]',
          style: {
            'background-color': '#14B8A6',
          },
        },
        {
          selector: 'node[type="concept"]',
          style: {
            'background-color': '#8B5CF6',
          },
        },
        {
          selector: 'node[type="keyword"]',
          style: {
            'background-color': '#F59E0B',
          },
        },
        {
          selector: 'edge',
          style: {
            width: 2,
            'line-color': '#6B7280',
            'target-arrow-color': '#6B7280',
            'target-arrow-shape': 'triangle',
            'curve-style': 'bezier',
            label: 'data(label)',
            'font-size': '10px',
            color: '#9CA3AF',
            'text-rotation': 'autorotate',
          },
        },
      ],
      layout: {
        name: 'cose',
        idealEdgeLength: 100,
        nodeOverlap: 20,
        refresh: 20,
        fit: true,
        padding: 30,
        randomize: false,
        componentSpacing: 100,
        nodeRepulsion: 400000,
        edgeElasticity: 100,
        nestingFactor: 5,
        gravity: 80,
        numIter: 1000,
        initialTemp: 200,
        coolingFactor: 0.95,
        minTemp: 1.0,
      },
    });

    return () => {
      if (cyRef.current) {
        cyRef.current.destroy();
      }
    };
  }, [memoryNodes, memoryEdges]);

  return (
    <div className="fixed right-0 top-0 bottom-0 w-[600px] bg-gray-800 border-l border-gray-700 z-50 flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-gray-700 bg-gray-800/50">
        <div className="flex items-center space-x-2">
          <h2 className="text-lg font-semibold text-white">Memory Graph</h2>
          <span className="text-sm text-gray-400">
            ({memoryNodes.length} nodes, {memoryEdges.length} edges)
          </span>
        </div>

        <div className="flex items-center space-x-2">
          <button
            onClick={refreshMemoryGraph}
            className="text-gray-400 hover:text-white transition-colors p-2 rounded hover:bg-gray-700"
            title="Refresh graph"
          >
            <RefreshCw size={18} />
          </button>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-white transition-colors p-2 rounded hover:bg-gray-700"
            title="Close"
          >
            <X size={20} />
          </button>
        </div>
      </div>

      {/* Graph Container */}
      {memoryNodes.length === 0 ? (
        <div className="flex-1 flex items-center justify-center text-gray-400">
          <div className="text-center">
            <p className="text-lg mb-2">No memory nodes yet</p>
            <p className="text-sm text-gray-500">
              Start chatting to build your knowledge graph
            </p>
          </div>
        </div>
      ) : (
        <div ref={containerRef} className="flex-1 bg-gray-900" />
      )}

      {/* Legend */}
      <div className="p-4 border-t border-gray-700 bg-gray-800/50">
        <h3 className="text-sm font-semibold text-white mb-2">Legend</h3>
        <div className="grid grid-cols-2 gap-2 text-xs text-gray-400">
          <div className="flex items-center space-x-2">
            <div className="w-3 h-3 rounded-full bg-[#14B8A6]"></div>
            <span>Entity</span>
          </div>
          <div className="flex items-center space-x-2">
            <div className="w-3 h-3 rounded-full bg-[#8B5CF6]"></div>
            <span>Concept</span>
          </div>
          <div className="flex items-center space-x-2">
            <div className="w-3 h-3 rounded-full bg-[#3B82F6]"></div>
            <span>Event</span>
          </div>
          <div className="flex items-center space-x-2">
            <div className="w-3 h-3 rounded-full bg-[#F59E0B]"></div>
            <span>Keyword</span>
          </div>
        </div>
      </div>
    </div>
  );
}
