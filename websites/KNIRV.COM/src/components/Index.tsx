'use client';

import React, { useState, useEffect, useRef } from 'react';
import Link from 'next/link';
import { Button } from "@/components/ui/button";
import {
  Shield, Cpu, Database, Network, Terminal, Zap, Layers,
  Key, Globe, BarChart3, Award, FileKey, Workflow, Coins,
  Bot, Eye, ChevronRight, Lock, Activity, Blocks,
  Server, GitBranch, Radio, Scan, FlaskConical, CheckCircle
} from 'lucide-react';

// Intersection-observer hook for scroll reveal
function useReveal(threshold = 0.12) {
  const ref = useRef<HTMLDivElement>(null);
  const [visible, setVisible] = useState(false);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const obs = new IntersectionObserver(
      ([entry]) => { if (entry.isIntersecting) { setVisible(true); obs.disconnect(); } },
      { threshold }
    );
    obs.observe(el);
    return () => obs.disconnect();
  }, [threshold]);
  return { ref, visible };
}

// Animated counter
function Counter({ target, suffix = '', duration = 2000 }: { target: number; suffix?: string; duration?: number }) {
  const [value, setValue] = useState(0);
  const { ref, visible } = useReveal(0.5);
  useEffect(() => {
    if (!visible) return;
    const start = Date.now();
    const tick = () => {
      const elapsed = Date.now() - start;
      const progress = Math.min(elapsed / duration, 1);
      const ease = 1 - Math.pow(1 - progress, 3);
      setValue(Math.round(target * ease));
      if (progress < 1) requestAnimationFrame(tick);
    };
    requestAnimationFrame(tick);
  }, [visible, target, duration]);
  return <span ref={ref}>{value}{suffix}</span>;
}

// ─── Data ───────────────────────────────────────────────────────────────────

const services = [
  {
    icon: Bot,
    color: 'blue',
    label: 'Multi-Provider AI Orchestration',
    desc: 'Route inference across Anthropic, Cerebras, Gemini, and DeepSeek with automatic failover, Mixture-of-Agents ensemble reasoning, and per-task provider routing.',
    points: ['Primary → fallback chain with zero downtime', 'MOA: planner / executor / verifier roles', 'Token-efficient context chunking'],
  },
  {
    icon: Shield,
    color: 'green',
    label: 'Enterprise AI Governance',
    desc: 'Auto-generate 50+ guardrail rules from your organizational ontology. Every LLM request checked against corporate policy before execution.',
    points: ['Ontology-driven rule generation', 'Real-time violation alerts via WebSocket', 'Policy hashes anchored to ValidationChain'],
  },
  {
    icon: Server,
    color: 'cyan',
    label: 'DVE Compute Marketplace',
    desc: 'One-click provisioning of Distributed Virtual Execution nodes. P2P discovery via libp2p DHT. NAT traversal via KNIRVGATEWAY TURN/STUN.',
    points: ['Docker → Podman → native fallback', 'SSH access from the browser (xterm.js)', 'Reputation-based load balancing'],
  },
  {
    icon: Database,
    color: 'purple',
    label: 'Sovereign Knowledge Base',
    desc: 'GraphRAG-rs (Rust FFI) builds entity-relation graphs from your document corpus. FAISS + BM25 hybrid search with graph context.',
    points: ['Upload PDFs, markdown, text documents', 'Deploy indexed knowledge for live inference', 'PQC-encrypted at rest'],
  },
  {
    icon: FileKey,
    color: 'amber',
    label: 'Cryptographic Compliance',
    desc: 'Post-quantum cryptographic signatures on every policy, validation result, and evidence pack. Immutable audit trail anchored on-chain.',
    points: ['PQC-signed evidence packs', 'ValidationChain anchoring with tx_hash', 'Regulator-ready compliance dashboards'],
  },
  {
    icon: Coins,
    color: 'emerald',
    label: 'Error Mining & NRN Rewards',
    desc: 'AI failures become monetizable knowledge. ErrorNode → SkillNode transitions mine skill.md files. Contributors earn NRN token rewards.',
    points: ['KNIRVGRAPH P2P error propagation', 'HERO Model automated resolution', 'Compute rewards proportional to contribution'],
  },
  {
    icon: Scan,
    color: 'red',
    label: 'eBPF Kernel Security',
    desc: 'Real-time kernel-level monitoring of every DVE workload. LSM policy events, syscall counts, and XDP packet filtering at the OS level.',
    points: ['Captures LLM prompts before encryption', 'Intent-action drift detection kill switch', 'Predictive anomaly detection'],
  },
  {
    icon: Award,
    color: 'indigo',
    label: 'Badge Governance Lab',
    desc: 'Mint capability tokens encoding ontology tags, value signals, and scoped auth credentials to KNIRVCHAIN. Attach badges to DVE nodes for inherited governance.',
    points: ['SVG badge minted as chain-native token', 'AND-evaluated stacking across DVE nodes', 'Audit trail stamped with badge_id'],
  },
];

const useCases = [
  {
    icon: Globe,
    title: 'Enterprise AI Governance',
    body: 'Deploy KNIRVSERVER as your internal AI governance layer. All LLM requests routed through policy-checked guardrails with cryptographic audit trails.',
    tag: 'Zero-Trust AI',
  },
  {
    icon: BarChart3,
    title: 'Regulated Financial Services',
    body: 'Fintech plugin loads KYC, AML, Basel, and SEC ontologies. FintechValidator runs compliance checks; ReplayEngine produces transaction audit trails.',
    tag: 'Fintech / Compliance',
  },
  {
    icon: FlaskConical,
    title: 'Sovereign Research Knowledge',
    body: 'Upload proprietary documents, build GraphRAG knowledge bases, and query with semantic graph retrieval — all encrypted at rest with PQC.',
    tag: 'Research / Knowledge',
  },
  {
    icon: Radio,
    title: 'Decentralized Error Mining',
    body: 'Operate a KNIRVGRAPH node that mines AI error knowledge, accumulates skill.md files, and earns NRN rewards for successful resolutions.',
    tag: 'Node Operator',
  },
];

const phases = [
  { n: '01', label: 'Onboard Organization', desc: 'ValueSystem + Ontology → auto-generated guardrail rules' },
  { n: '02', label: 'Provision DVE Nodes', desc: 'One-click container deployment, P2P network registration' },
  { n: '03', label: 'Ingest Knowledge', desc: 'GraphRAG-rs indexes documents; plugins registered' },
  { n: '04', label: 'Delegate Credentials', desc: 'Provider chain configured; MOA roles assigned' },
  { n: '05', label: 'Run Inference', desc: 'Tasks routed, context chunked, MOA ensemble fires' },
  { n: '06', label: 'Enforce Guardrails', desc: 'Every request checked; violations logged and anchored' },
  { n: '07', label: 'Mine & Anchor', desc: 'Errors become skills; evidence packs anchored on-chain' },
];

const colorMap: Record<string, { bg: string; border: string; text: string; glow: string }> = {
  blue:    { bg: 'bg-blue-500/10',    border: 'border-blue-500/30',    text: 'text-blue-400',    glow: 'group-hover:shadow-blue-500/20' },
  green:   { bg: 'bg-green-500/10',   border: 'border-green-500/30',   text: 'text-green-400',   glow: 'group-hover:shadow-green-500/20' },
  cyan:    { bg: 'bg-cyan-500/10',    border: 'border-cyan-500/30',    text: 'text-cyan-400',    glow: 'group-hover:shadow-cyan-500/20' },
  purple:  { bg: 'bg-purple-500/10',  border: 'border-purple-500/30',  text: 'text-purple-400',  glow: 'group-hover:shadow-purple-500/20' },
  amber:   { bg: 'bg-amber-500/10',   border: 'border-amber-500/30',   text: 'text-amber-400',   glow: 'group-hover:shadow-amber-500/20' },
  emerald: { bg: 'bg-emerald-500/10', border: 'border-emerald-500/30', text: 'text-emerald-400', glow: 'group-hover:shadow-emerald-500/20' },
  red:     { bg: 'bg-red-500/10',     border: 'border-red-500/30',     text: 'text-red-400',     glow: 'group-hover:shadow-red-500/20' },
  indigo:  { bg: 'bg-indigo-500/10',  border: 'border-indigo-500/30',  text: 'text-indigo-400',  glow: 'group-hover:shadow-indigo-500/20' },
};

// ─── Main Component ──────────────────────────────────────────────────────────

const Index = () => {
  const servicesReveal  = useReveal();
  const useCasesReveal  = useReveal();
  const phasesReveal    = useReveal();
  const ctaReveal       = useReveal();
  const statsReveal     = useReveal(0.3);

  return (
    <div className="dve-page text-white overflow-x-hidden">

      {/* ─── Inline keyframes ─── */}
      <style>{`
        @keyframes float-slow  { 0%,100%{transform:translateY(0)}    50%{transform:translateY(-24px)} }
        @keyframes float-med   { 0%,100%{transform:translateY(0)}    50%{transform:translateY(-14px)} }
        @keyframes pulse-ring  { 0%{transform:scale(1);opacity:.6}  100%{transform:scale(1.6);opacity:0} }
        @keyframes grid-drift  { 0%{background-position:0 0}        100%{background-position:60px 60px} }
        @keyframes fade-up     { from{opacity:0;transform:translateY(32px)} to{opacity:1;transform:translateY(0)} }
        @keyframes fade-left   { from{opacity:0;transform:translateX(-24px)} to{opacity:1;transform:translateX(0)} }
        @keyframes slide-right { from{width:0} to{width:100%} }
        .float-slow  { animation: float-slow  7s ease-in-out infinite; }
        .float-med   { animation: float-med   5s ease-in-out infinite; }
        .pulse-ring  { animation: pulse-ring  2s ease-out infinite; }
        .grid-drift  { animation: grid-drift 12s linear infinite; }
        .fade-up     { animation: fade-up     0.7s ease forwards; }
        .fade-left   { animation: fade-left   0.7s ease forwards; }
        .stagger-1   { animation-delay: 0.1s; opacity:0; }
        .stagger-2   { animation-delay: 0.25s; opacity:0; }
        .stagger-3   { animation-delay: 0.4s; opacity:0; }
        .stagger-4   { animation-delay: 0.55s; opacity:0; }
      `}</style>

      {/* ════════════════════════════════════════════════════════
          NAV
      ════════════════════════════════════════════════════════ */}
      <nav className="fixed top-0 left-0 right-0 z-50 dve-nav">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="relative">
              <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-500 to-cyan-400 flex items-center justify-center">
                <Network className="w-4 h-4 text-white" />
              </div>
              <div className="absolute inset-0 rounded-lg bg-blue-500/40 blur-sm -z-10" />
            </div>
            <span className="text-xl font-black tracking-tight text-white">KNIRV</span>
            <span className="hidden sm:block text-xs font-semibold text-white/30 uppercase tracking-widest ml-1 mt-px">SERVER</span>
          </div>
          <div className="flex items-center gap-3">
            <Link href="/documentation">
              <Button variant="ghost" size="sm" className="text-white/60 hover:text-white text-sm">Docs</Button>
            </Link>
            <Link href="/features">
              <Button variant="ghost" size="sm" className="text-white/60 hover:text-white text-sm">Features</Button>
            </Link>
            <Link href="/pricing">
              <Button size="sm" className="bg-blue-600 hover:bg-blue-500 text-white font-bold px-4">
                Get Started
              </Button>
            </Link>
          </div>
        </div>
      </nav>

      {/* ════════════════════════════════════════════════════════
          HERO
      ════════════════════════════════════════════════════════ */}
      <section className="relative min-h-screen flex flex-col items-center justify-center pt-16 overflow-hidden">

        {/* Animated grid background */}
        <div
          className="absolute inset-0 opacity-[0.04] grid-drift"
          style={{ backgroundImage: 'linear-gradient(rgba(0,192,250,1) 1px,transparent 1px),linear-gradient(90deg,rgba(0,192,250,1) 1px,transparent 1px)', backgroundSize: '60px 60px' }}
        />

        {/* Ambient glows */}
        <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-blue-600/20 rounded-full blur-[120px] float-slow pointer-events-none" />
        <div className="absolute bottom-1/4 right-1/4 w-80 h-80 bg-indigo-600/15 rounded-full blur-[100px] float-med pointer-events-none" />
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] bg-cyan-500/5 rounded-full blur-[140px] pointer-events-none" />

        {/* Pulsing rings */}
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 pointer-events-none">
          <div className="w-[700px] h-[700px] rounded-full border border-blue-500/10 pulse-ring" style={{ animationDelay: '0s' }} />
          <div className="absolute inset-0 w-[700px] h-[700px] rounded-full border border-cyan-500/8 pulse-ring" style={{ animationDelay: '0.7s' }} />
          <div className="absolute inset-0 w-[700px] h-[700px] rounded-full border border-indigo-500/6 pulse-ring" style={{ animationDelay: '1.4s' }} />
        </div>

        {/* Content */}
        <div className="relative z-10 max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 text-center space-y-8">

          <div className="fade-up stagger-1 inline-flex items-center gap-2 px-4 py-1.5 rounded-full border border-blue-500/30 bg-blue-500/10 text-blue-400 text-xs font-bold uppercase tracking-widest">
            <span className="w-2 h-2 rounded-full bg-blue-400 animate-pulse" />
            Enterprise AI Governance Platform
          </div>

          <h1 className="fade-up stagger-2 text-5xl sm:text-6xl lg:text-7xl font-black leading-[1.05] tracking-tight">
            Govern Every{' '}
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-blue-400 via-cyan-400 to-blue-500">
              AI Decision
            </span>
            <br />
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 to-indigo-400">
              At the Source
            </span>
          </h1>

          <p className="fade-up stagger-3 max-w-2xl mx-auto text-lg sm:text-xl text-white/60 leading-relaxed">
            KNIRVSERVER is your organization&apos;s AI sovereignty layer — orchestrating multi-provider LLMs, enforcing ontology-driven guardrails,
            anchoring compliance evidence on-chain, and turning AI failures into collective, monetizable knowledge.
          </p>

          <div className="fade-up stagger-4 flex flex-col sm:flex-row gap-4 justify-center">
            <Button asChild size="lg" className="bg-gradient-to-r from-blue-600 to-cyan-600 hover:from-blue-500 hover:to-cyan-500 text-white font-bold px-8 py-6 text-base h-auto shadow-lg shadow-blue-500/25">
              <Link href="/documentation">
                <Zap className="w-5 h-5 mr-2" />
                Deploy KNIRVSERVER
              </Link>
            </Button>
            <Button asChild size="lg" variant="outline" className="border-white/15 text-white/70 hover:bg-white/8 hover:text-white px-8 py-6 text-base h-auto">
              <Link href="/features">
                <Terminal className="w-5 h-5 mr-2" />
                Explore Architecture
              </Link>
            </Button>
          </div>

          {/* Trust signals */}
          <div className="fade-up flex flex-wrap items-center justify-center gap-x-8 gap-y-3 pt-4 text-white/30 text-xs font-semibold uppercase tracking-widest" style={{ animationDelay: '0.7s', opacity: 0 }}>
            <span className="flex items-center gap-2"><CheckCircle className="w-3.5 h-3.5 text-green-500" /> Post-Quantum Cryptography</span>
            <span className="flex items-center gap-2"><CheckCircle className="w-3.5 h-3.5 text-green-500" /> On-Chain Audit Anchoring</span>
            <span className="flex items-center gap-2"><CheckCircle className="w-3.5 h-3.5 text-green-500" /> eBPF Kernel Security</span>
            <span className="flex items-center gap-2"><CheckCircle className="w-3.5 h-3.5 text-green-500" /> Multi-Provider Failover</span>
          </div>
        </div>

        {/* Scroll indicator */}
        <div className="absolute bottom-10 left-1/2 -translate-x-1/2 flex flex-col items-center gap-2 animate-bounce text-white/20">
          <div className="w-px h-10 bg-gradient-to-b from-transparent to-white/20" />
          <ChevronRight className="w-4 h-4 rotate-90" />
        </div>
      </section>

      {/* ════════════════════════════════════════════════════════
          STATS STRIP
      ════════════════════════════════════════════════════════ */}
      <section className="border-y border-white/5 bg-white/[0.02]" ref={statsReveal.ref}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
          <div className={`grid grid-cols-2 md:grid-cols-4 gap-8 transition-all duration-700 ${statsReveal.visible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-8'}`}>
            {[
              { value: 50, suffix: '+', label: 'Auto-Generated Guardrail Rules', icon: Shield },
              { value: 3,  suffix: ' LLM Providers', label: 'In Automatic Failover Chain', icon: Bot },
              { value: 100, suffix: '%', label: 'PQC-Signed Evidence On-Chain', icon: FileKey },
              { value: 4,  suffix: ' Node Types', label: 'DVE, Skill, Error, Capability', icon: Layers },
            ].map((stat, i) => (
              <div key={i} className="text-center space-y-2">
                <div className="text-3xl sm:text-4xl font-black text-transparent bg-clip-text bg-gradient-to-r from-blue-400 to-cyan-400">
                  {statsReveal.visible ? <Counter target={stat.value} suffix={stat.suffix} /> : '0'}
                </div>
                <div className="text-xs text-white/40 uppercase tracking-wider leading-tight">{stat.label}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ════════════════════════════════════════════════════════
          SERVICES GRID
      ════════════════════════════════════════════════════════ */}
      <section className="py-28 px-4 sm:px-6 lg:px-8 max-w-7xl mx-auto">
        <div ref={servicesReveal.ref}>
          <div className={`text-center mb-16 transition-all duration-700 ${servicesReveal.visible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-8'}`}>
            <div className="inline-block text-xs font-bold uppercase tracking-widest text-blue-400 mb-4 px-3 py-1 border border-blue-500/20 rounded-full bg-blue-500/5">
              Platform Services
            </div>
            <h2 className="text-4xl sm:text-5xl font-black leading-tight">
              Everything your enterprise AI stack needs,<br />
              <span className="text-transparent bg-clip-text bg-gradient-to-r from-blue-400 to-cyan-400">sovereign from day one</span>
            </h2>
            <p className="mt-4 text-white/50 text-lg max-w-2xl mx-auto">
              Eight integrated services that compose into a complete AI governance, compute, and compliance platform.
            </p>
          </div>

          <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-5">
            {services.map((svc, i) => {
              const c = colorMap[svc.color];
              const Icon = svc.icon;
              return (
                <div
                  key={i}
                  className={`group relative p-6 rounded-2xl border bg-white/[0.02] hover:bg-white/[0.05] transition-all duration-500 cursor-default ${c.border} hover:shadow-xl ${c.glow} ${servicesReveal.visible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-10'}`}
                  style={{ transitionDelay: servicesReveal.visible ? `${i * 60}ms` : '0ms' }}
                >
                  <div className={`w-11 h-11 rounded-xl ${c.bg} flex items-center justify-center mb-4 transition-transform duration-300 group-hover:scale-110`}>
                    <Icon className={`w-5 h-5 ${c.text}`} />
                  </div>
                  <h3 className="font-bold text-white text-sm mb-2 leading-tight">{svc.label}</h3>
                  <p className="text-white/40 text-xs leading-relaxed mb-4">{svc.desc}</p>
                  <ul className="space-y-1.5">
                    {svc.points.map((pt, j) => (
                      <li key={j} className="flex items-start gap-2 text-xs text-white/30">
                        <ChevronRight className={`w-3 h-3 mt-px flex-shrink-0 ${c.text} opacity-70`} />
                        {pt}
                      </li>
                    ))}
                  </ul>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      {/* ════════════════════════════════════════════════════════
          PLATFORM LIFECYCLE
      ════════════════════════════════════════════════════════ */}
      <section className="py-24 bg-gradient-to-b from-transparent via-blue-950/10 to-transparent">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8" ref={phasesReveal.ref}>
          <div className={`text-center mb-16 transition-all duration-700 ${phasesReveal.visible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-8'}`}>
            <div className="inline-block text-xs font-bold uppercase tracking-widest text-cyan-400 mb-4 px-3 py-1 border border-cyan-500/20 rounded-full bg-cyan-500/5">
              Platform Lifecycle
            </div>
            <h2 className="text-4xl font-black">Seven phases. One coherent platform.</h2>
            <p className="mt-3 text-white/40 max-w-xl mx-auto">
              From initial onboarding to continuous knowledge mining, every stage is governed, auditable, and composable.
            </p>
          </div>

          <div className="relative">
            {/* Connector line */}
            <div className="hidden lg:block absolute top-8 left-0 right-0 h-px bg-gradient-to-r from-transparent via-blue-500/30 to-transparent" />

            <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-7 gap-4">
              {phases.map((ph, i) => (
                <div
                  key={i}
                  className={`relative flex flex-col items-center text-center gap-3 transition-all duration-500 ${phasesReveal.visible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-6'}`}
                  style={{ transitionDelay: phasesReveal.visible ? `${i * 80}ms` : '0ms' }}
                >
                  <div className="relative z-10 w-16 h-16 rounded-2xl bg-gradient-to-br from-blue-600/30 to-cyan-600/20 border border-blue-500/30 flex flex-col items-center justify-center">
                    <span className="text-[10px] font-black text-blue-400 uppercase tracking-wider">{ph.n}</span>
                  </div>
                  <div>
                    <div className="text-xs font-bold text-white leading-tight">{ph.label}</div>
                    <div className="text-[10px] text-white/30 mt-1 leading-tight">{ph.desc}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* ════════════════════════════════════════════════════════
          USE CASES
      ════════════════════════════════════════════════════════ */}
      <section className="py-28 px-4 sm:px-6 lg:px-8 max-w-7xl mx-auto">
        <div ref={useCasesReveal.ref}>
          <div className={`text-center mb-16 transition-all duration-700 ${useCasesReveal.visible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-8'}`}>
            <div className="inline-block text-xs font-bold uppercase tracking-widest text-indigo-400 mb-4 px-3 py-1 border border-indigo-500/20 rounded-full bg-indigo-500/5">
              Use Cases
            </div>
            <h2 className="text-4xl font-black">Built for organizations that take AI seriously</h2>
          </div>

          <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6">
            {useCases.map((uc, i) => {
              const Icon = uc.icon;
              return (
                <div
                  key={i}
                  className={`group relative p-7 rounded-2xl border border-white/8 bg-white/[0.02] hover:bg-white/[0.05] hover:border-white/15 transition-all duration-400 ${useCasesReveal.visible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-10'}`}
                  style={{ transitionDelay: useCasesReveal.visible ? `${i * 80}ms` : '0ms' }}
                >
                  <div className="inline-block text-[9px] font-black uppercase tracking-widest text-blue-400 border border-blue-500/25 bg-blue-500/8 px-2 py-0.5 rounded-full mb-4">
                    {uc.tag}
                  </div>
                  <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-blue-500/20 to-indigo-500/10 flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
                    <Icon className="w-5 h-5 text-blue-400" />
                  </div>
                  <h3 className="font-bold text-white mb-3">{uc.title}</h3>
                  <p className="text-sm text-white/40 leading-relaxed">{uc.body}</p>
                  <div className="mt-5 flex items-center text-xs text-blue-400 font-semibold opacity-0 group-hover:opacity-100 transition-opacity">
                    Learn more <ChevronRight className="w-3.5 h-3.5 ml-1" />
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      {/* ════════════════════════════════════════════════════════
          SECURITY SPOTLIGHT
      ════════════════════════════════════════════════════════ */}
      <section className="py-24 bg-gradient-to-br from-red-950/15 via-transparent to-transparent">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid lg:grid-cols-2 gap-12 items-center">
            <div className="space-y-6">
              <div className="inline-block text-xs font-bold uppercase tracking-widest text-red-400 px-3 py-1 border border-red-500/20 rounded-full bg-red-500/5">
                Security Architecture
              </div>
              <h2 className="text-4xl font-black leading-tight">
                Kernel-level oversight.<br />
                <span className="text-transparent bg-clip-text bg-gradient-to-r from-red-400 to-orange-400">Not just a firewall.</span>
              </h2>
              <p className="text-white/50 text-lg leading-relaxed">
                GuardrailEngine runs below your application stack. eBPF probes inspect every DVE workload at the OS level —
                capturing LLM prompts before encryption, correlating intended tool calls with actual syscalls, and triggering
                session revocation the instant agent behavior drifts from its declared intent.
              </p>
              <ul className="space-y-4">
                {[
                  { icon: Eye, text: 'Intercepts plain-text prompts before they reach the LLM provider', color: 'text-red-400' },
                  { icon: Scan, text: 'Maps predicted tool calls vs. actual kernel syscalls in real time', color: 'text-orange-400' },
                  { icon: Lock, text: 'Revokes sessions and blocks calls on intent-action drift detection', color: 'text-amber-400' },
                  { icon: Activity, text: 'Predictive anomaly surface via DistributedScaler + ProactiveDetector', color: 'text-yellow-400' },
                ].map((item, i) => {
                  const Icon = item.icon;
                  return (
                    <li key={i} className="flex items-start gap-3">
                      <div className="w-8 h-8 rounded-lg bg-white/5 flex items-center justify-center flex-shrink-0 mt-0.5">
                        <Icon className={`w-4 h-4 ${item.color}`} />
                      </div>
                      <span className="text-white/60 text-sm leading-relaxed">{item.text}</span>
                    </li>
                  );
                })}
              </ul>
            </div>

            {/* Visual panel */}
            <div className="relative">
              <div className="rounded-2xl border border-red-500/20 bg-black/40 p-6 font-mono text-xs space-y-2 backdrop-blur-sm overflow-hidden">
                <div className="flex items-center gap-2 mb-4">
                  <div className="w-2 h-2 rounded-full bg-red-500 animate-pulse" />
                  <span className="text-red-400 font-bold text-[10px] uppercase tracking-widest">eBPF Monitor — Live</span>
                </div>
                {[
                  { time: '14:02:31.004', event: 'SYSCALL_INTERCEPT', detail: 'pid=4821 comm=knirvdve syscall=openat', color: 'text-white/50' },
                  { time: '14:02:31.021', event: 'LLM_PROMPT_CAPTURE', detail: 'tokens=312 provider=anthropic intent=search', color: 'text-cyan-400' },
                  { time: '14:02:31.044', event: 'TOOL_CALL_CHECK', detail: 'predicted=search actual=socket — MATCH', color: 'text-green-400' },
                  { time: '14:02:31.109', event: 'SYSCALL_INTERCEPT', detail: 'pid=4821 syscall=connect dst=10.0.0.1:443', color: 'text-white/50' },
                  { time: '14:02:31.115', event: 'INTENT_DRIFT_ALERT', detail: 'predicted=search actual=shell — MISMATCH', color: 'text-red-400 font-bold' },
                  { time: '14:02:31.116', event: 'SESSION_REVOKED', detail: 'agent=dve-a1b2 reason=action_drift', color: 'text-orange-400 font-bold' },
                  { time: '14:02:31.118', event: 'GUARDRAIL_VIOLATION', detail: 'severity=CRITICAL node=dve-a1b2', color: 'text-red-500 font-bold' },
                ].map((row, i) => (
                  <div key={i} className="grid grid-cols-[85px_1fr] gap-3">
                    <span className="text-white/20">{row.time}</span>
                    <div>
                      <span className={`${row.color}`}>[{row.event}] </span>
                      <span className="text-white/30">{row.detail}</span>
                    </div>
                  </div>
                ))}
              </div>
              {/* Glow behind panel */}
              <div className="absolute inset-0 bg-red-500/5 rounded-2xl blur-xl -z-10" />
            </div>
          </div>
        </div>
      </section>

      {/* ════════════════════════════════════════════════════════
          CTA
      ════════════════════════════════════════════════════════ */}
      <section className="py-32 px-4 sm:px-6 lg:px-8" ref={ctaReveal.ref}>
        <div className={`max-w-4xl mx-auto text-center space-y-8 transition-all duration-700 ${ctaReveal.visible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-10'}`}>
          <div className="inline-block text-xs font-bold uppercase tracking-widest text-blue-400 px-3 py-1 border border-blue-500/20 rounded-full bg-blue-500/5">
            Deploy Today
          </div>
          <h2 className="text-5xl sm:text-6xl font-black leading-tight">
            Your organization's AI.<br />
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-blue-400 to-cyan-400">
              Governed. Auditable. Sovereign.
            </span>
          </h2>
          <p className="text-white/40 text-lg max-w-2xl mx-auto">
            Deploy KNIRVSERVER on your infrastructure or use our managed cloud hosting. Both options include
            the full governance stack, DVE provisioning, and on-chain compliance anchoring.
          </p>

          <div className="grid sm:grid-cols-2 gap-5 max-w-2xl mx-auto">
            {/* Self-hosted */}
            <div className="group p-6 rounded-2xl border border-white/10 bg-white/[0.03] hover:border-blue-500/40 hover:bg-white/[0.06] transition-all text-left space-y-4">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-blue-500/15 flex items-center justify-center">
                  <Server className="w-5 h-5 text-blue-400" />
                </div>
                <div>
                  <div className="font-bold text-white text-sm">Sovereign Edition</div>
                  <div className="text-xs text-white/30">Self-hosted on your infrastructure</div>
                </div>
              </div>
              <ul className="space-y-2 text-xs text-white/40">
                {['Full data sovereignty', 'No vendor data access', 'systemd / Docker deployment', 'Community support'].map(f => (
                  <li key={f} className="flex items-center gap-2">
                    <CheckCircle className="w-3.5 h-3.5 text-green-500 flex-shrink-0" /> {f}
                  </li>
                ))}
              </ul>
              <Button asChild className="w-full bg-blue-600 hover:bg-blue-500 text-white font-bold" size="sm">
                <Link href="/documentation">Download & Deploy</Link>
              </Button>
            </div>

            {/* Cloud */}
            <div className="group p-6 rounded-2xl border border-cyan-500/25 bg-cyan-500/5 hover:border-cyan-400/50 hover:bg-cyan-500/10 transition-all text-left space-y-4">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-cyan-500/15 flex items-center justify-center">
                  <Globe className="w-5 h-5 text-cyan-400" />
                </div>
                <div>
                  <div className="font-bold text-white text-sm">Managed Cloud</div>
                  <div className="text-xs text-white/30">Hosted & managed by KNIRV</div>
                </div>
              </div>
              <ul className="space-y-2 text-xs text-white/40">
                {['Instant provisioning', 'Auto-scaling DVE nodes', 'SLA-backed uptime', 'Enterprise support'].map(f => (
                  <li key={f} className="flex items-center gap-2">
                    <CheckCircle className="w-3.5 h-3.5 text-cyan-400 flex-shrink-0" /> {f}
                  </li>
                ))}
              </ul>
              <Button asChild className="w-full bg-gradient-to-r from-cyan-600 to-blue-600 hover:from-cyan-500 hover:to-blue-500 text-white font-bold" size="sm">
                <Link href="/pricing">View Pricing</Link>
              </Button>
            </div>
          </div>

          <p className="text-white/20 text-xs">
            Questions? Read the <Link href="/documentation" className="text-blue-400 hover:text-blue-300 underline underline-offset-2">documentation</Link> or{' '}
            <Link href="/contact" className="text-blue-400 hover:text-blue-300 underline underline-offset-2">contact us</Link>.
          </p>
        </div>
      </section>

    </div>
  );
};

export default Index;
