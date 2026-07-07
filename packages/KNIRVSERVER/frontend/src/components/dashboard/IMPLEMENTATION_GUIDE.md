# DVE Dashboard Integration Guide

## Architecture Overview

### Component Tree
```
DVENodesPanel (dve-nodes-panel.tsx)
├── DVE List Cards Grid
│   ├── Start Button
│   ├── Details Button
│   ├── Access Button (conditional)
│   └── DVE Solver Button
├── DVECardModal
├── CDEPanel
│   ├── ConsolePanel (slides from top-left)
│   ├── PolicyEditor (slides below console)
│   ├── MonitorPanel (slides from bottom)
│   └── ConnectionsPanel (left sidebar)
└── DVESolverModal
```

## User Workflows

### Workflow 1: Viewing Node Details
```
1. User sees DVE card with [Start] [Details] buttons
2. Click [Details]
3. DVECardModal opens with:
   - Full node information
   - Performance metrics
   - TEE details
   - Reputation score
   - Location info
4. Click close or click outside to dismiss
```

### Workflow 2: Accessing Cloud Development Environment
```
1. User sees DVE card with [Start] [Details] buttons
2. Click [Start] button
   → Button turns green
   → "Access" button appears
3. Click [Access] button
4. CDEPanel opens with:
   - Header showing node info
   - Tools section with 4 buttons
5. Click individual tool buttons to reveal panels:
   - Console → Real-time failure feed (top-left)
   - Policy → Security editor (middle-left)
   - Monitor → Resolution tracker (bottom)
   - Connections → NRV list (left sidebar)
6. Multiple panels can be open simultaneously
7. Click [X] on any panel to close it
8. Click main close button to close CDE
```

### Workflow 3: Using DVE Solver
```
1. User clicks [DVE Solver] button
   (Available regardless of node status)
2. DVESolverModal opens with:
   - Problem panel (left) - NRV selection
   - Validation panel (right) - test execution
   - Submission panel (bottom-right) - result submission
3. Select an NRV from problem list
4. Click [Run Validation] button
5. Watch log stream update in real-time
6. Review validation report card
7. Click [Submit to Consensus] when ready
8. Close modal to dismiss
```

## Component Communication

### DVENodesPanel → DVECardModal
```typescript
Props:
- isOpen: boolean
- onClose: () => void
- node: DVENode

Data Flow:
DVENodesPanel stores selected node in state
→ Passes to DVECardModal
→ Modal displays complete node information
```

### DVENodesPanel → CDEPanel
```typescript
Props:
- isOpen: boolean
- onClose: () => void
- nodeId?: string
- nodeName?: string

Data Flow:
DVENodesPanel stores selected node
→ Passes ID and name to CDE
→ CDE displays in header for context
→ CDE can use nodeId for data filtering
```

### CDEPanel → Sub-panels
```typescript
CDEPanel manages state for:
- showConsole: boolean
- showPolicy: boolean
- showMonitor: boolean
- showConnections: boolean

Each sub-panel receives:
- isOpen: boolean (derived from showXXX state)
- onClose: () => void (handler to toggle state)

Sub-panels are positioned absolutely
and controlled via show/hide state
```

## CSS Layout Logic

### CDE Panel Content Area Adjustments
```typescript
const marginLeft = showConnections ? '280px' : '0'
const marginBottom = showMonitor ? 'calc(33vh + 1rem)' : '0'

This creates space for:
- Left sidebar (Connections panel)
- Bottom slide-up (Monitor panel)
- Left slide-outs don't push content (absolute positioning)
```

### Panel Z-Index Stacking
```
z-50:  Main modals (CDEPanel, DVESolverModal)
z-30:  Sub-panels (Console, Policy, Monitor)
z-20:  Connections sidebar
z-50:  Backdrop (black/50 backdrop-blur)
```

## Black & Blue Theme Implementation

### Color Palette
```javascript
// Primary Blues
blue-600: '#2563eb'   // Interactive elements
blue-950: '#172554'   // Dark backgrounds

// Slate Grays
slate-900: '#0f172a'  // Darkest backgrounds
slate-950: '#03050a'  // Ultra-dark backgrounds
slate-800: '#1e293b'  // Card backgrounds
slate-700: '#334155'  // Hover states

// Accents
green-600: '#16a34a'   // Online/success
red-600: '#dc2626'     // Errors/offline
yellow-400: '#facc15'  // Warnings
purple-600: '#9333ea'  // Access/premium

// Text
blue-300: '#93c5fd'    // Primary headings
slate-300: '#cbd5e1'   // Body text
slate-400: '#94a3b8'   // Muted text
```

### Usage Examples
```jsx
// Gradient backgrounds
className="bg-gradient-to-br from-slate-900 via-blue-950 to-slate-950"

// Blue borders with transparency
className="border border-blue-600/30"
className="border-2 border-blue-600/50"

// Text colors
className="text-blue-300"  // Headings
className="text-slate-300" // Body
className="text-slate-400" // Muted

// Interactive states
className="hover:border-blue-600"
className="focus:border-blue-500"
className="active:bg-blue-700"
```

## State Management Best Practices

### Local State vs Lifted State

**Local (Component Level)**
```typescript
// In ConsolePanel, PolicyEditor, etc.
const [selectedNRV, setSelectedNRV] = useState(null)
const [isValidating, setIsValidating] = useState(false)
```

**Lifted (Parent Level)**
```typescript
// In CDEPanel
const [showConsole, setShowConsole] = useState(false)
const [showPolicy, setShowPolicy] = useState(false)
// Controls visibility of child panels
```

**Top Level (Root)**
```typescript
// In DVENodesPanel
const [selectedNode, setSelectedNode] = useState<DVENode | null>(null)
const [cdeOpen, setCDEOpen] = useState(false)
// Controls which node and which modals are open
```

## Performance Considerations

### Modal Rendering
- Modals are conditionally rendered: `{isOpen && <Component />}`
- Prevents DOM overhead when not visible

### List Rendering
```typescript
// Use keys for NRV lists
{mockNRVs.map(nrv => (
  <div key={nrv.id}>  // ← Important!
    ...
  </div>
))}
```

### Memoization (Future Enhancement)
```typescript
const ConsolePanel = React.memo(({ isOpen, onClose }) => {
  // Component only re-renders if props change
})
```

## Responsive Design Notes

### Current Implementation
- Fixed positioning for modals (viewport-based)
- Absolute positioning for sub-panels
- May need adjustments for mobile/tablet

### Recommended Improvements
- Add media queries for smaller screens
- Stack panels vertically on mobile
- Consider full-screen mode on tablets
- Add touch-friendly button sizes

## Testing Scenarios

### Unit Tests (Per Component)
- ConsolePanel: Renders when isOpen=true
- PolicyEditor: Saves policy changes
- MonitorPanel: Displays logs correctly
- ConnectionsPanel: Filters NRVs on search

### Integration Tests
- CDE opens with correct node info
- Opening one panel doesn't interfere with others
- Panel state persists during CDE session
- Closing CDE resets all panel states

### End-to-End Tests
- Full workflow: Start → Access → Open panels → Close
- Solver workflow: Click button → Select NRV → Validate → Submit
- Card modal: Open → View details → Close
- Multiple nodes: Switch between nodes without closing CDE

## Troubleshooting

### Issue: Panels not appearing
- Check z-index values
- Verify isOpen prop is true
- Check CSS overflow properties

### Issue: Layout breaking with multiple panels
- Verify margin/padding calculations
- Check viewport height calculations
- Test with browser zoom

### Issue: Buttons not responding
- Check onClick handlers are bound correctly
- Verify state updates are working
- Check for event propagation issues

## Migration Notes

### From Old Architecture
- Old `DVEDualDashboard` with tabs is now split
- Enterprise RTR functionality moved to CDE sub-panels
- Solver interface extracted to separate modal
- Card details extracted to separate modal

### Backwards Compatibility
- `dve_dual_dashboard.tsx` still exists for reference
- Can be removed after full migration
- No breaking changes to existing DVE data structure

## Future Roadmap

### Phase 2: Data Integration
- Connect panels to real API endpoints
- Implement WebSocket for real-time updates
- Add data persistence to local storage

### Phase 3: Advanced Features
- Drag-to-resize panels
- Custom panel layouts per user
- Export logs and reports
- Real-time collaboration

### Phase 4: Analytics
- Track panel usage patterns
- Monitor performance metrics
- User behavior analytics

### Phase 5: Mobile Support
- Responsive design overhaul
- Touch-optimized interactions
- Mobile-specific layouts