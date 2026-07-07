# DVE Dashboard - Visual Component Guide

## DVE List Card - Initial State

```
┌─────────────────────────────────────────────────┐
│ Node Name                          [●]  OFFLINE  │
├─────────────────────────────────────────────────┤
│                                                 │
│  CPU:     ████████░░░░░░░░░░  45%              │
│  Memory:  ███████░░░░░░░░░░░░  32%              │
│                                                 │
│  TEE:         SGX                               │
│  Stake:       1,000 NRN                         │
│  Rep Score:   92/100                            │
│  Location:    San Francisco                     │
│  Last HB:     5m ago                            │
│                                                 │
│  ┌──────────────┬──────────────┐               │
│  │ [▶ Start]    │ [Details]    │               │
│  └──────────────┴──────────────┘               │
│  ┌────────────────────────────────┐            │
│  │ [⚡ DVE Solver]                │            │
│  └────────────────────────────────┘            │
└─────────────────────────────────────────────────┘
```

## DVE List Card - After Clicking Start

```
┌─────────────────────────────────────────────────┐
│ Node Name                          [●]  ONLINE ◄── CHANGED!
├─────────────────────────────────────────────────┤
│ ...node details...                              │
│                                                 │
│  ┌──────────┬──────────┬──────────┐            │
│  │ [✓ Start]│[Details] │[Access]  │◄── NEW!
│  │ GREEN    │ BLUE     │ PURPLE   │
│  └──────────┴──────────┴──────────┘            │
│  ┌────────────────────────────────┐            │
│  │ [⚡ DVE Solver]                │            │
│  └────────────────────────────────┘            │
└─────────────────────────────────────────────────┘
```

## DVE Card Modal - Details View

```
╔════════════════════════════════════════════════════════╗
║  Node Name (ID: node-12345)                      [×]  ║
╠════════════════════════════════════════════════════════╣
║                                                        ║
║  Status: [● ONLINE]  Last heartbeat: 5m ago           ║
║                                                        ║
║  ┌──────────────────────────────────────────────────┐ ║
║  │ PERFORMANCE METRICS                              │ ║
║  ├──────────────────────────────────────────────────┤ ║
║  │ CPU Usage:      ████████░░░░░░░░  45%           │ ║
║  │ Memory Usage:   ███████░░░░░░░░░░  32%           │ ║
║  └──────────────────────────────────────────────────┘ ║
║                                                        ║
║  ┌────────────────┬────────────────┬─────────────┐  ║
║  │ TEE: SGX       │ Stake: 1K NRN │ Rep: 92/100 │  ║
║  └────────────────┴────────────────┴─────────────┘  ║
║                                                        ║
║  ┌──────────────────────────────────────────────────┐ ║
║  │ ADDITIONAL INFORMATION                           │ ║
║  │ TEE Version: v2.1.0                              │ ║
║  │ Last Update: 2 hours ago                         │ ║
║  │ Uptime: 45 days                                  │ ║
║  └──────────────────────────────────────────────────┘ ║
║                                                        ║
║                                      [Close]          ║
╚════════════════════════════════════════════════════════╝
```

## Cloud Development Environment (CDE) Panel

```
╔════════════════════════════════════════════════════════════╗
║ Cloud Development Environment                        [×]   ║
║ Node: node-name (node-12345)                              ║
╠════════════════════════════════════════════════════════════╣
║                                                            ║
║  ┌────────────────────────────────────────────────────┐  ║
║  │ TOOLS                                              │  ║
║  ├────────────────────────────────────────────────────┤  ║
║  │ ┌──────────────┬──────────────┐                    │  ║
║  │ │ 🖥 Console   │ 🛡️ Policy    │                    │  ║
║  │ │ (Red)        │ (Amber)      │                    │  ║
║  │ └──────────────┴──────────────┘                    │  ║
║  │ ┌──────────────┬──────────────┐                    │  ║
║  │ │ 📊 Monitor   │ 🌐 Connections                   │  ║
║  │ │ (Green)      │ (Purple)     │                    │  ║
║  │ └──────────────┴──────────────┘                    │  ║
║  │                                                    │  ║
║  │ Status: ● Online  Active Panels: 0                │  ║
║  └────────────────────────────────────────────────────┘  ║
║                                                            ║
╚════════════════════════════════════════════════════════════╝
```

## CDE with All Panels Open (Layout)

```
┌─────────────────────────────────────────────────────────────┐
│ CDE HEADER                                            [×]   │
├──────────┬────────────────────────────────────────────────┤
│CONN  │ CONSOLE PANEL (top-left)                           │
│CONN  │ [X] Real-Time Failure Feed                        │
│CONN  │ ────────────────────────────────────────────────  │
│CONN  │ [10:45] FAILURE DETECTED - cortex-8472            │
│CONN  │ [10:44] FAILURE DETECTED - cortex-8471            │
│CONN  ├────────────────────────────────────────────────────┤
│CONN  │ POLICY PANEL (middle-left)                        │
│CONN  │ [X] Security Policy Editor                        │
│CONN  │ ────────────────────────────────────────────────  │
│      │ Network Whitelist │ Validation Sensitivity        │
│      │ [Save Policy]                                     │
│      ├────────────────────────────────────────────────────┤
│      │ MAIN CONTENT - TOOLS VIEW                         │
│      │ [Console] [Policy] [Monitor] [Connections]        │
│      │                                                    │
├──────┴────────────────────────────────────────────────────┤
│MONITOR PANEL (bottom - 33vh) - locks to viewport           │
│Timestamp │ Failure ID │ Node │ Strategy │ Validation      │
│10:45:23  │ FAIL-8472  │ C-01 │ Constraint │ PASS          │
│10:44:56  │ FAIL-8471  │ C-02 │ Forensic   │ FAIL          │
└──────────────────────────────────────────────────────────┘

Legend:
CONN = Connections Sidebar (280px vertical, left edge)
```

## DVE Solver Modal - 3-Column Layout

```
╔═══════════════════════════════════════════════════════════════════╗
║ 🔧 DVE Solver Interface                                     [×]   ║
╠═══════════════════════════════════════════════════════════════════╣
║                                                                   ║
║  ┌──────────────────┬─────────────────────────────────────────┐ ║
║  │ PROBLEM PANEL    │ VALIDATION PANEL                        │ ║
║  ├──────────────────┼─────────────────────────────────────────┤ ║
║  │ [Search NRVs]    │ [▶ Run Validation]                      │ ║
║  │                  │                                         │ ║
║  │ ┌──────────────┐ │ ┌─────────────────────────────────────┐│ ║
║  │ │ NRV-001      │ │ │ [strace] Intercepting...           ││ ║
║  │ │ Hallucination│ │ │ [ltrace] Monitoring...             ││ ║
║  │ │ HIGH ▲       │ │ │ [sandbox] Executing...             ││ ║
║  │ ├──────────────┤ │ │ [ghidra] Running analysis...        ││ ║
║  │ │ NRV-002      │ │ │                                     ││ ║
║  │ │ Security     │ │ └─────────────────────────────────────┘│ ║
║  │ │ Flaw         │ │                                         │ ║
║  │ │ HIGH ▲       │ │ ✓ Static Analysis: PASS               │ ║
║  │ │              │ │ ✓ Dynamic Analysis: PASS              │ ║
║  │ └──────────────┘ │ ✓ Forensics: PASS                     │ ║
║  │                  │ ✓ Resolution: PASS                    │ ║
║  │ Failure Context: │                                         │ ║
║  │ Prompt: ...      │ ┌─────────────────────────────────────┐│ ║
║  │ Bad Response: .. │ │ SUBMISSION PANEL                   ││ ║
║  │ Issue: Halluc... │ ├─────────────────────────────────────┤│ ║
║  │                  │ │ NRN Balance: 1500                  ││ ║
║  │                  │ │ Required Bond: 100                 ││ ║
║  │                  │ │                                    ││ ║
║  │                  │ │ [🔐 Submit to Consensus]           ││ ║
║  │                  │ └─────────────────────────────────────┘│ ║
║  └──────────────────┴─────────────────────────────────────────┘ ║
║                                                    [Close]        ║
╚═══════════════════════════════════════════════════════════════════╝
```

## Console Panel - Slide-out Detail

```
BEFORE:                    AFTER clicking Console button:
                          
                          ┌──────────────────────┐
                          │ [X] Console          │
                          │ Real-Time Feed       │
                          │ ─────────────────── │
                          │ [10:45] FAILURE     │
                          │ [10:44] FAILURE     │
                          │ [10:43] RESOLVED    │
                          │ [10:42] FAILURE     │
                          │ [10:41] FAILURE     │
                          │                    │
                          │ (500px × 200px)     │
                          │ (Positioned top-left)
                          └──────────────────────┘
```

## Policy Editor - Slide-out Detail

```
BEFORE:                    AFTER clicking Policy button:
                          
┌──────────────────┐      ┌──────────────────┐
│ Console Panel    │      │ Console Panel    │
│ (if open)        │      │ (if open)        │
│                  │      │                  │
└──────────────────┘      └──────────────────┘
                          ┌──────────────────┐
                          │ [X] Policy       │
                          │ Security Editor  │
                          │ ──────────────  │
                          │ Network Whitelist
                          │ [textarea]       │
                          │                  │
                          │ Validation       │
                          │ [dropdown]       │
                          │ [✓] Block file I/O
                          │ [□] Allow R/O    │
                          │ [✓] Forensics    │
                          │ [Save Policy]    │
                          │ (500px × 280px)  │
                          │ (Below Console)   │
                          └──────────────────┘
```

## Monitor Panel - Viewport-Locked Bottom

```
┌─────────────────────────────────────────────────────────┐
│ Main CDE Content (67% height)                           │
│                                                         │
│                                                         │
└─────────────────────────────────────────────────────────┘
┌──────────────────────────────────────────────────────────┐
│ [X] Monitor - Resolution Tracking (33% height)          │
├──────────────────────────────────────────────────────────┤
│ Time │ Failure ID │ Node │ Strategy │ Validation │      │
├──────────────────────────────────────────────────────────┤
│ 10:45│ FAIL-8472  │ C-01 │ Constraint │ ✓ PASS  │      │
│ 10:44│ FAIL-8471  │ C-02 │ Forensic   │ ✗ FAIL  │      │
│ 10:43│ FAIL-8470  │ C-01 │ Self-Corr  │ ✓ PASS  │      │
│                                                          │
│ ↓ SCROLLABLE ↓                                          │
│                                                          │
│ (Stays at bottom when scrolling)                        │
└──────────────────────────────────────────────────────────┘
```

## Connections Sidebar - Independent Left Edge

```
┌──────┬───────────────────────────────────────────────────┐
│ CONN │ MAIN CDE CONTENT                                 │
│ CONN │                                                   │
│ [X]  │ Console Panel (if open)                          │
│CONN  │ Policy Panel (if open)                           │
│ NRV  │                                                   │
│LIST  │ Main view area                                   │
│      │                                                   │
│ [Search] │                                               │
│ ──────── │                                               │
│ NRV-001  │                                               │
│ Halluc.. │ Monitor Panel (if open)                       │
│ HIGH     │                                               │
│ ──────── │                                               │
│ NRV-002  │                                               │
│ Security │                                               │
│ HIGH     │                                               │
│ ──────── │                                               │
│ NRV-003  │                                               │
│ Math     │                                               │
│ MEDIUM   │                                               │
│          │                                               │
│ (280px   │                                               │
│ width,   │                                               │
│ full     │                                               │
│ height)  │                                               │
└──────────┴───────────────────────────────────────────────┘
```

## Color Coding System

### Buttons
- **Blue** (#2563eb): Primary actions (Start initially, Details)
- **Green** (#16a34a): Active/Success state (Start when online, validation pass)
- **Purple** (#9333ea): Premium/Access features (Access button)
- **Amber/Orange** (#f59e0b): Secondary tools (DVE Solver)
- **Red** (#dc2626): Errors/Alerts (Console failures)

### Panels
- **Console**: Red accents (Alert theme)
- **Policy**: Amber accents (Security theme)
- **Monitor**: Green accents (Success tracking)
- **Connections**: Purple accents (Network theme)

### Backgrounds
- **Main**: `from-slate-900 via-blue-950 to-slate-950` (gradient)
- **Cards**: `from-slate-800 to-blue-950` (gradient)
- **Borders**: `border-blue-600/30` (transparent blue)

## Interactive States

### Button States
```
Default:  bg-slate-800 border-slate-700 text-slate-300
Hover:    bg-slate-700 border-blue-600 (brightens)
Active:   bg-blue-600 shadow-lg shadow-blue-500/30
Disabled: bg-slate-600 cursor-not-allowed
```

### Panel States
```
Open:     Slides in with transition
Closed:   Slides out, display: none equivalent
Hover:    border-blue-600 (full opacity)
```

## Responsive Behavior

### Current (Desktop)
- Full layout as designed
- Fixed pixel dimensions
- Absolute positioning

### Future (Tablet)
- Stack panels vertically
- Reduce panel widths
- Touch-friendly sizing

### Future (Mobile)
- Full-screen panels
- Stacked layout
- Touch gestures