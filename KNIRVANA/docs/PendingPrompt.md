
In KNIRVCONTROLLER, we need to move some of the buttons and indicators around to fit the game logic and mechanics:

1. Move the theme toggle button into the KnirvShell.tsx topbar header all the way to the right of the screen, to the right of the wallet icon.
2. Remove the "NRN Balance" box on the Game arena UI above the "Tournament" box.
3. Move the "Tournament" box up in the top right of the arena UI. Add a small progress bar near the "Incumbent Score" of this box showing how much time is left in the current Epoch before weights are committed to LoRAX. The "Red Queen" Meter.
4. Move the "Run Epoch" button from the bottom left of the arena screen into the Tournament box on the top right. Make the new "Run Epoch" button indicate its current state with "Epoch Running" status once pressed.
5. When we first click on an agent from the arena, a small "Agent" information box appears but we never see it again. This box is not able to show again, we need to fix that. Reshape and move this box up to fit between the Tournament box and Error Node boxes effectively. Allow the information in the new agent box to change based on agent selection. Add a new "Stage" button at the bottom of each agent card on the arena floor that will remove them from the grid for training in agent management.  
6. 
	A) Refactor the "Error Node" box on the right side of the gaming arena so that it displays more information about the error (once the agent information is migrated up into its own box) just like NRV notiications does currently. ID, status, severity, description, type, and NRN Bounty. 
	B) Repurpose NRV error node notiications (pop ups on the left of the KnirvShell.tsx viewport) to system notifications - Real-time notifications of Adversarial Drift or Backprop Pulses.
	
7. In the "Cognitive Shell", move the memory graph into the "Cortex Builder" modal under the "Data" tab at the bottom. Make the memory button in cognitive shell navigate to the new location in the cortex builder. 
8. On the "Key Agent Status" slide out modal, add a button that opens the "Cortex Builder" modal simultaniously, sliding in from the right side of the screen. This is how we train our primary agent. Increase the height of the robot image and ensure we see the white version of the robot avater as its default state here. Add a button that opens "Agent Management" simultaneously here as well. Opening in the same fashion as the Cortex Builder from here sliding in from the right.
9. In the burger menu remove "Network Status" and change "Network Selection" to simply "Network" for simplicity.
10. In the "Agent Management" modal, move the "Key Agent" box to the left side and move the Training Stats box to the right side of the modal, switching their positions. On the "Key Agent" card, add a train button that opens the cortex builder training tab.
11. In the "Agent Management" modal, on each agent card, add the following buttons to accompany the start traing and deploy agent buttons:
	- Distill (Optimization): Trims the trajectory to lower Inference Latency.
	- Harden (Defense): Runs "Stress Tests" to increase Parity and Stability.
	- Quantize (Precision): A slider to find the lowest bit-rate for a speed boost.
12. Ensure the "Training Stats" box updates data based on the selected sub-agent.
13. In the "Agent Management" modal, when we click on "Deploy Agent" we should see the modal close and an animation of the agent avatar being tossed onto the grid from the camera viewport.
14. Remove the Agent Actions box from the bottom left of the arena view. 
15. Repupose the main button in the bottom right of the KnirvShell.tsx as the new "Sabatoge" button. While an adversary status is "Solving," the Sabotage button should pulse. This allows us to spend Compute to use the 4 nested buttons perform various sabatoge actions:
	- Noise Injection: Dropping "Adversarial Fog" on the map. Obscuring the path for opponent Agents to cause "hallucinations." 
	- Weight Hijacking: If Agent B's trajectory  has a higher reward  than the current Skill Slot holder  (), Agent B "overwrites" the adapter.
	- Context Poisoning: Players can inject "Dead Tokens" into the shared prompt context.
	- - Effect: Increases the opponent's inference latency, lowering their efficiency score.
	- Backprop Pulse: When an opponent fails a Verifier check, you can force a "Negative Reinforcement" on them, pushing their weights away from the Global Minimum.



In KNIRVCONTROLLER, we need to move some of the buttons and indicators around to fit the game logic and mechanics:

1. Replace the "Analyzer" button text in the bottom left view of the gaming arena with "Analyze" that Generates a "Heatmap" overlay of the entire arena grid to find the Global Minimum. Once pressed this button will generate a red fabric across the arena that curves downward and upward from the grid depending on global minimum ranges within the grid around each error node. 
2. The grid around each error node provides access to the metadata that accompanies each error such as logs and traces. This is what is analyzed.
3. In the previous place of the "Run Epoch" button add a new "Sculpt" button that is disabled until the Analyze button is pressed. Once enabled, this is where we write the "Test Cases" or adjust the Final Reward Equation ($R = w_c \cdot C + w_l \cdot L + w_s \cdot S$). This is dependent on the available data within range of the error node. We can choose a noow visible data point within the heat map to place a test case. This allows us to click the 3D grid to place Reward Anchors (checkpoints) to lure our agents away from the "Red Zone" peaks. Once placed, the anchor opens the "Cognitive Shell" modal automatically.
4. In the "Cognitive Shell", allow error node metadata to be automatically loaded into the chat as an attachment with a preloaded prompt that requests a test case for the anchor. llow the user to use the Notes functionality to save the inference enabled response as if "chaining" the test case to the anchor.
5. Once a Reward Anchor has be placed, the architect can click on the anchor icon in the grid and it opens a Verifyer modal. This layout creates a dual-pane view: a "Weight Slider" panel on the left for reward shaping and a "Constraint Editor" on the right for test case editing. This new modal should look like the following:

// VerifierOverlay.html

```HTML
<div id="verifier-overlay" class="cyber-modal">
  <div class="header">
    <h2>VERIFIER_LOGIC_GATE // ARCHITECT_V1</h2>
    <button class="close-btn" onclick="toggleVerifier()">[X]</button>
  </div>

  <div class="content-grid">
    <section class="weight-shaping">
      <h3>REWARD_WEIGHTS</h3>
      <div class="slider-group">
        <label>Correctness (w_c): <span id="val-wc">0.6</span></label>
        <input type="range" min="0" max="1" step="0.05" value="0.6" oninput="updateWeights()">
      </div>
      <div class="slider-group">
        <label>Latency (w_l): <span id="val-wl">0.3</span></label>
        <input type="range" min="0" max="1" step="0.05" value="0.3" oninput="updateWeights()">
      </div>
      <div class="slider-group">
        <label>Simplicity (w_s): <span id="val-ws">0.1</span></label>
        <input type="range" min="0" max="1" step="0.05" value="0.1" oninput="updateWeights()">
      </div>
      <p class="formula-preview">Score = (C × w_c) + (L_norm × w_l) + (S × w_s)</p>
    </section>

    <section class="constraint-editor">
      <h3>UNIT_TEST_INJECTION</h3>
      <div class="editor-container">
        <textarea id="test-editor" spellcheck="false">
// DEFINE ADVERSARIAL CONSTRAINTS
verifier.addConstraint("memory_leak_check", (agent_res) => {
  return agent_res.usage < 128; // Max 128MB
});

verifier.setTargetOutput([104729, 104743]);
        </textarea>
      </div>
      <button class="deploy-logic-btn" onclick="commitLogic()">COMMIT_TO_ARENA</button>
    </section>
  </div>
</div>
```
// VerifyerOverlay.css:

```css
.cyber-modal {
  position: absolute;
  top: 10%;
  left: 10%;
  width: 80%;
  height: 70%;
  background: rgba(4, 12, 24, 0.95);
  border: 1px solid #00f2ff;
  box-shadow: 0 0 20px rgba(0, 242, 255, 0.3);
  color: #e0e0e0;
  font-family: 'Courier New', monospace;
  display: flex;
  flex-direction: column;
  z-index: 1000;
}

.content-grid {
  display: grid;
  grid-template-columns: 1fr 2fr;
  gap: 20px;
  padding: 20px;
  flex-grow: 1;
}

h3 {
  color: #00f2ff;
  border-bottom: 1px dashed #00f2ff;
  padding-bottom: 5px;
}

.editor-container textarea {
  width: 100%;
  height: 300px;
  background: #000;
  color: #00ff41; /* Matrix green for code */
  border: 1px solid #333;
  padding: 10px;
  font-size: 14px;
}

.deploy-logic-btn {
  background: #ff0055;
  color: white;
  border: none;
  padding: 15px;
  width: 100%;
  cursor: pointer;
  margin-top: 10px;
  font-weight: bold;
}

.formula-preview {
  font-size: 11px;
  color: #888;
  margin-top: 20px;
}
```

6. Establish Logical Flow: From UI to LoRAX

Once the human commits this logic, the following happens in our TypeScript engine:
	- Physics Update: The TournamentController receives the new weights ($w_c, w_l, w_s$) and updates the Verifier instance.
	- Edge Case Injection: The raw JavaScript from the text editor is wrapped into the verifier.addConstraint map.
	- Active Evaluation: The next Agent that clicks "Run Epoch" is now running against our new test cases.
	- Reward Recalculation: If an opponent was winning through a "dirty hack" (e.g., high memory usage), our new memory_leak_check constraint will instantly tank their reward score, potentially triggering a Weight Hijacking in our favor.






