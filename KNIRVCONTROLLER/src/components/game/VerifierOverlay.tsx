import React, { useState } from 'react';
import './VerifierOverlay.css';

interface WindowWithTournamentController extends Window {
  tournamentController?: {
    updateRewardWeights: (weights: { w_c: number; w_l: number; w_s: number }) => void;
    addConstraint: (constraints: string) => void;
  };
}

interface RewardAnchor {
  id: number;
  x: number;
  y: number;
  weights: {
    w_c: number;
    w_l: number;
    w_s: number;
  };
  constraints: string;
  metadata?: any;
}

interface VerifierOverlayProps {
  onClose: () => void;
  rewardAnchors: RewardAnchor[];
  setRewardAnchors: (anchors: RewardAnchor[]) => void;
  initialAnchor?: RewardAnchor | null;
}

export default function VerifierOverlay({ onClose, rewardAnchors, setRewardAnchors, initialAnchor }: VerifierOverlayProps) {
  const [selectedAnchor, setSelectedAnchor] = useState<RewardAnchor | null>(
    initialAnchor || (rewardAnchors.length > 0 ? rewardAnchors[rewardAnchors.length - 1] : null)
  );
  
  const [weights, setWeights] = useState({
    w_c: selectedAnchor?.weights.w_c || 0.6,
    w_l: selectedAnchor?.weights.w_l || 0.3,
    w_s: selectedAnchor?.weights.w_s || 0.1
  });
  
  const [constraints, setConstraints] = useState(
    selectedAnchor?.constraints || `// DEFINE ADVERSARIAL CONSTRAINTS
verifier.addConstraint("memory_leak_check", (agent_res) => {
  return agent_res.usage < 128; // Max 128MB
});

verifier.setTargetOutput([104729, 104743]);`
  );

  const updateWeights = (weightType: 'w_c' | 'w_l' | 'w_s', value: number) => {
    const newWeights = { ...weights, [weightType]: value };
    setWeights(newWeights);
  };

  const commitLogic = () => {
    if (!selectedAnchor) return;

    const updatedAnchors = rewardAnchors.map(anchor =>
      anchor.id === selectedAnchor.id
        ? { ...anchor, weights, constraints }
        : anchor
    );
    
    setRewardAnchors(updatedAnchors);
    
    // Send logic to TournamentController
    console.log('Committing logic to TournamentController:', { weights, constraints });
    
    // Integrate with TournamentController/LoRAX system
    const windowWithTC = window as WindowWithTournamentController;
    if (typeof window !== 'undefined' && windowWithTC.tournamentController) {
      windowWithTC.tournamentController.updateRewardWeights(weights);
      windowWithTC.tournamentController.addConstraint(constraints);
      console.log('Logic committed to TournamentController/LoRAX system');
    }
    
    onClose();
  };

  return (
    <div id="verifier-overlay" className="cyber-modal">
      <div className="header">
        <h2>VERIFIER_LOGIC_GATE // ARCHITECT_V1</h2>
        <button className="close-btn" onClick={onClose}>[X]</button>
      </div>

      <div className="content-grid">
        <section className="weight-shaping">
          <h3>REWARD_WEIGHTS</h3>
          <div className="slider-group">
            <label>Correctness (w_c): <span id="val-wc">{weights.w_c.toFixed(2)}</span></label>
            <input 
              type="range" 
              min="0" 
              max="1" 
              step="0.05" 
              value={weights.w_c} 
              onInput={(e: React.ChangeEvent<HTMLInputElement>) => updateWeights('w_c', parseFloat(e.target.value))}
            />
          </div>
          <div className="slider-group">
            <label>Latency (w_l): <span id="val-wl">{weights.w_l.toFixed(2)}</span></label>
            <input 
              type="range" 
              min="0" 
              max="1" 
              step="0.05" 
              value={weights.w_l} 
              onInput={(e: React.ChangeEvent<HTMLInputElement>) => updateWeights('w_l', parseFloat(e.target.value))}
            />
          </div>
          <div className="slider-group">
            <label>Simplicity (w_s): <span id="val-ws">{weights.w_s.toFixed(2)}</span></label>
            <input 
              type="range" 
              min="0" 
              max="1" 
              step="0.05" 
              value={weights.w_s} 
              onInput={(e: React.ChangeEvent<HTMLInputElement>) => updateWeights('w_s', parseFloat(e.target.value))}
            />
          </div>
          <p className="formula-preview">Score = (C × w_c) + (L_norm × w_l) + (S × w_s)</p>
        </section>

        <section className="constraint-editor">
          <h3>UNIT_TEST_INJECTION</h3>
          <div className="editor-container">
            <textarea 
              id="test-editor" 
              spellCheck={false}
              value={constraints}
              onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setConstraints(e.target.value)}
            />
          </div>
          <button className="deploy-logic-btn" onClick={commitLogic}>COMMIT_TO_ARENA</button>
        </section>
      </div>
    </div>
  );
}