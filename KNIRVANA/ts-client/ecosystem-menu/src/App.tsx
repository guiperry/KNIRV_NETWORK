import React, { useMemo, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Bot, Wallet, Trophy, Star, ChevronLeft, ChevronRight, Hammer, Zap, Shield, Brain, Map as MapIcon, Award, Plus, Minus } from "lucide-react";
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts";

// ----------------------- Utility: Seeded RNG -----------------------
function makeRNG(seed = 1) {
  let s = seed % 2147483647;
  if (s <= 0) s += 2147483646;
  return function () {
    // Park–Miller RNG
    s = (s * 16807) % 2147483647;
    return (s - 1) / 2147483646;
  };
}

// ----------------------- Catalogs & Constants -----------------------
const SKILL_CATALOG = [
  {
    id: "debugging",
    name: "Debugging",
    icon: Hammer,
    domain: "Core",
    desc: "Syntax repair, trace analysis, unit patching.",
    baseCost: 30,
  },
  {
    id: "ml",
    name: "Machine Learning",
    icon: Brain,
    domain: "Intelligence",
    desc: "Pattern detection, anomaly prediction.",
    baseCost: 40,
  },
  {
    id: "security",
    name: "Security",
    icon: Shield,
    domain: "Defense",
    desc: "Exploit patching, encryption, access control.",
    baseCost: 35,
  },
  {
    id: "optimization",
    name: "Optimization",
    icon: Zap,
    domain: "Performance",
    desc: "Runtime tuning, memory & IO efficiency.",
    baseCost: 25,
  },
];

const ATTRIBUTE_KEYS = [
  { id: "strength", label: "Strength" },
  { id: "logic", label: "Logic" },
  { id: "agility", label: "Agility" },
  { id: "charisma", label: "Charisma" },
];

// ----------------------- Helpers -----------------------
function severityToColor(sev: number) {
  if (sev >= 3) return "from-rose-500 to-red-600";
  if (sev === 2) return "from-amber-400 to-orange-600";
  return "from-yellow-300 to-amber-500";
}

function classNames(...a: (string | boolean | undefined)[]) {
  return a.filter(Boolean).join(" ");
}

// ----------------------- Data Generators -----------------------
function generateClusters(count = 16, seed = 42) {
  const rng = makeRNG(seed);
  return Array.from({ length: count }).map((_, i) => {
    const sev = Math.ceil(rng() * 3);
    const state = rng() > 0.85 ? "critical" : "active"; // a few critical
    const col = i % 4;
    const row = Math.floor(i / 4);
    return {
      id: `cluster-${i + 1}`,
      name: `Cluster ${i + 1}`,
      severity: sev,
      state,
      grid: { row, col },
    };
  });
}

function generateErrorsForCluster(clusterId: string, mode = "minimal") {
  const seedNum = parseInt(clusterId.split("-")[1], 10) || 1;
  const rng = makeRNG(seedNum * (mode === "full" ? 17 : 7));
  const errorCount = mode === "full" ? Math.ceil(rng() * 4) + 2 : Math.ceil(rng() * 2) + 1; // 2-3 minimal, 3-6 full

  const errors = [];
  for (let i = 0; i < errorCount; i++) {
    const requiredIdxA = Math.floor(rng() * SKILL_CATALOG.length);
    const requiredIdxB = mode === "full" && rng() > 0.6 ? Math.floor(rng() * SKILL_CATALOG.length) : null;
    const required = [SKILL_CATALOG[requiredIdxA].id];
    if (requiredIdxB != null && requiredIdxB !== requiredIdxA) required.push(SKILL_CATALOG[requiredIdxB].id);
    const complexity = mode === "full" ? (rng() > 0.7 ? 2 : 1) + (rng() > 0.9 ? 1 : 0) : 1; // up to 3 in full

    errors.push({
      id: `${clusterId}-err-${i + 1}`,
      title: `Error ${i + 1}`,
      required, // array of skill ids
      complexity, // min level needed across required skills
      status: "active",
    });
  }
  return errors;
}

// ----------------------- Types -----------------------
interface Cluster {
  id: string;
  name: string;
  severity: number;
  state: string;
  grid: { row: number; col: number };
}

interface Error {
  id: string;
  title: string;
  required: string[];
  complexity: number;
  status: string;
}

interface Avatar {
  strength: number;
  logic: number;
  agility: number;
  charisma: number;
  pool: number;
}

interface HistoryEntry {
  knerv: number;
  clusters: number;
}

// ----------------------- Root Component -----------------------
export default function UntitledEcosystemGame() {
  const [mode, setMode] = useState("");
  const [knerv, setKnerv] = useState(200);
  const [botSkills, setBotSkills] = useState<Record<string, number>>({ debugging: 1 }); // start minimal: debugging L1
  const [clusters] = useState(() => generateClusters(16, 101));
  const [activeClusterId, setActiveClusterId] = useState<string | null>(null);
  const [clusterErrors, setClusterErrors] = useState<Error[]>([]);
  const [avatar, setAvatar] = useState<Avatar>({
    strength: 0,
    logic: 0,
    agility: 0,
    charisma: 0,
    pool: 0
  });
  const [history, setHistory] = useState<HistoryEntry[]>([]); // for chart

  const effectiveMode = mode || "minimal"; // default to minimal until user switches

  const activeCluster = useMemo(() => clusters.find((c) => c.id === activeClusterId) || null, [clusters, activeClusterId]);

  const progressData = useMemo(() => {
    return history.map((h, idx) => ({
      step: idx + 1,
      knerv: h.knerv,
      clusters: h.clusters
    }));
  }, [history]);

  function openCluster(c: Cluster) {
    setActiveClusterId(c.id);
    const errs = generateErrorsForCluster(c.id, effectiveMode === "full" ? "full" : "minimal");
    setClusterErrors(errs);
  }

  function buySkill(skillId: string) {
    const cat = SKILL_CATALOG.find((s) => s.id === skillId);
    if (!cat) return;
    const current = botSkills[skillId] || 0;
    const cost = Math.round(cat.baseCost * Math.pow(1.6, current));
    if (knerv < cost) return;
    setKnerv((k) => k - cost);
    setBotSkills((bs) => ({ ...bs, [skillId]: current + 1 }));
    noteHistory();
  }

  function noteHistory() {
    setHistory((h) => [
      ...h,
      {
        knerv: knerv,
        clusters: clusters.filter((c) => c.state === "resolved").length,
      },
    ]);
  }

  function canFix(error: Error) {
    // need all required skills at or above complexity
    return error.required.every((rid) => (botSkills[rid] || 0) >= error.complexity);
  }

  function attemptFix(errorId: string) {
    setClusterErrors((errs) => {
      const idx = errs.findIndex((e) => e.id === errorId);
      if (idx === -1) return errs;
      const e = errs[idx];
      if (e.status !== "active" || !canFix(e)) return errs;

      const newErrs = [...errs];
      newErrs[idx] = { ...e, status: "resolved" };

      // Reward Knerv per fix
      const reward = 10 + (e.complexity - 1) * 5;
      setKnerv((k) => k + reward);

      // In full mode: chance to spawn cascade error
      if (effectiveMode === "full") {
        const rng = makeRNG(Date.now() % 100000);
        if (rng() > 0.7) {
          const reqA = SKILL_CATALOG[Math.floor(rng() * SKILL_CATALOG.length)].id;
          const reqB = rng() > 0.6 ? SKILL_CATALOG[Math.floor(rng() * SKILL_CATALOG.length)].id : null;
          const required = reqB && reqB !== reqA ? [reqA, reqB] : [reqA];
          const complexity = rng() > 0.75 ? 2 : 1;
          newErrs.push({
            id: `${activeClusterId}-err-${newErrs.length + 1}`,
            title: `Cascaded Error ${newErrs.length + 1}`,
            required,
            complexity,
            status: "active",
          });
        }
      }

      // When all resolved, finish cluster
      const allResolved = newErrs.every((x) => x.status === "resolved");
      if (allResolved) completeCluster();
      return newErrs;
    });
  }

  function completeCluster() {
    // Mark cluster as resolved & award attribute points
    const bonus = activeCluster?.severity === 3 ? 5 : activeCluster?.severity === 2 ? 3 : 2;
    setAvatar((a) => ({ ...a, pool: a.pool + bonus }));
    setKnerv((k) => k + 20 + (activeCluster?.severity || 1) * 5);

    // mutate clusters array immutably
    const idx = clusters.findIndex((c) => c.id === activeClusterId);
    if (idx !== -1) {
      clusters[idx] = { ...clusters[idx], state: "resolved" };
    }
    setActiveClusterId(null);
    noteHistory();
  }

  function spendAttribute(attrId: string, delta: number) {
    setAvatar((a) => {
      if (delta > 0 && a.pool <= 0) return a;
      const next = { ...a };
      if (delta > 0) {
        next[attrId as keyof Avatar] = ((next[attrId as keyof Avatar] as number) || 0) + 1;
        next.pool -= 1;
      } else if (delta < 0 && (next[attrId as keyof Avatar] as number) > 0) {
        next[attrId as keyof Avatar] = (next[attrId as keyof Avatar] as number) - 1;
        next.pool += 1;
      }
      return next;
    });
  }

  return (
    <div className="min-h-screen w-full bg-slate-950 text-slate-100">
      <Header mode={effectiveMode} onModeChange={setMode} />
      <div className="mx-auto max-w-7xl px-4 pb-10">
        <div className="grid grid-cols-12 gap-4">
          {/* Left: Bot & Skills */}
          <div className="col-span-12 md:col-span-3">
            <BotPanel knerv={knerv} botSkills={botSkills} onBuy={buySkill} mode={effectiveMode} />
            {effectiveMode === "full" && <AvatarPanel avatar={avatar} onSpend={spendAttribute} />}
          </div>

          {/* Center: Map or Cluster */}
          <div className="col-span-12 md:col-span-6">
            {!activeCluster && (
              <EcosystemMap
                clusters={clusters}
                onOpenCluster={openCluster}
                mode={effectiveMode}
              />
            )}
            {activeCluster && (
              <ClusterView
                cluster={activeCluster}
                errors={clusterErrors}
                canFix={canFix}
                onFix={attemptFix}
                onBack={() => setActiveClusterId(null)}
                mode={effectiveMode}
              />
            )}
          </div>

          {/* Right: Leaderboard & Charts */}
          <div className="col-span-12 md:col-span-3 space-y-4">
            <LeaderboardPanel clusters={clusters} />
            {effectiveMode === "full" && (
              <ProgressChart data={progressData} />
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

// ----------------------- Header -----------------------
interface HeaderProps {
  mode: string;
  onModeChange: (mode: string) => void;
}

function Header({ mode, onModeChange }: HeaderProps) {
  return (
    <div className="sticky top-0 z-30 backdrop-blur supports-[backdrop-filter]:bg-slate-950/70">
      <div className="mx-auto max-w-7xl px-4 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <motion.div
            initial={{ scale: 0.9, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            className="p-2 rounded-2xl bg-gradient-to-br from-indigo-500 to-purple-600 shadow-lg shadow-indigo-800/30"
          >
            <MapIcon className="w-6 h-6" />
          </motion.div>
          <div>
            <h1 className="text-xl md:text-2xl font-semibold tracking-tight">Untitled Ecosystem</h1>
            <p className="text-xs text-slate-400">Compete. Evolve. Debug the Chaos.</p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <span className="text-xs text-slate-400 hidden sm:block">Mode</span>
          <div className="inline-flex rounded-xl p-1 bg-slate-800/60 border border-slate-700/50">
            <button
              onClick={() => onModeChange("minimal")}
              className={classNames(
                "px-3 py-1.5 rounded-lg text-sm transition",
                mode === "minimal"
                  ? "bg-slate-700 text-white shadow"
                  : "text-slate-300 hover:text-white"
              )}
            >
              Step 1: Minimal
            </button>
            <button
              onClick={() => onModeChange("full")}
              className={classNames(
                "px-3 py-1.5 rounded-lg text-sm transition",
                mode === "full"
                  ? "bg-slate-700 text-white shadow"
                  : "text-slate-300 hover:text-white"
              )}
            >
              Step 2: Full
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

// ----------------------- Panels -----------------------
interface PanelProps {
  title: string;
  icon?: React.ComponentType<{ className?: string }>;
  children: React.ReactNode;
  className?: string;
}

function Panel({ title, icon: Icon, children, className = "" }: PanelProps) {
  return (
    <div className={classNames("rounded-2xl border border-slate-800 bg-slate-900/60 shadow-xl shadow-black/20", className)}>
      <div className="flex items-center gap-2 px-4 py-3 border-b border-slate-800/70">
        {Icon && <Icon className="w-4 h-4 text-slate-300" />}
        <h2 className="text-sm font-semibold tracking-wide text-slate-200">{title}</h2>
      </div>
      <div className="p-4">{children}</div>
    </div>
  );
}

interface BotPanelProps {
  knerv: number;
  botSkills: Record<string, number>;
  onBuy: (skillId: string) => void;
  mode: string;
}

function BotPanel({ knerv, botSkills, onBuy, mode }: BotPanelProps) {
  return (
    <Panel title="Bot Control & Skills" icon={Bot}>
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-sky-500 to-cyan-600 flex items-center justify-center shadow">
            <Bot className="w-5 h-5" />
          </div>
          <div>
            <div className="text-sm">TopBot-Δ</div>
            <div className="text-xs text-slate-400">v0.{mode === "full" ? "2" : "1"}</div>
          </div>
        </div>
        <div className="flex items-center gap-2 text-emerald-300">
          <Wallet className="w-4 h-4" />
          <span className="text-sm font-medium">{knerv} Knerv</span>
        </div>
      </div>

      <div className="space-y-3">
        {SKILL_CATALOG.map((s) => (
          <div key={s.id} className="flex items-center justify-between rounded-xl border border-slate-800 p-3 bg-slate-900/60">
            <div className="flex items-center gap-3">
              <div className="w-9 h-9 rounded-lg bg-slate-800/80 flex items-center justify-center">
                <s.icon className="w-4 h-4" />
              </div>
              <div>
                <div className="text-sm font-medium">{s.name}</div>
                <div className="text-[10px] text-slate-400">{s.desc}</div>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <div className="text-xs text-slate-300">Lv. {botSkills[s.id] || 0}</div>
              <button
                onClick={() => onBuy(s.id)}
                className="text-xs px-2 py-1 rounded-lg bg-emerald-600/80 hover:bg-emerald-600 transition border border-emerald-500/40"
              >
                Buy
              </button>
            </div>
          </div>
        ))}
      </div>
    </Panel>
  );
}

interface AvatarPanelProps {
  avatar: Avatar;
  onSpend: (attrId: string, delta: number) => void;
}

function AvatarPanel({ avatar, onSpend }: AvatarPanelProps) {
  return (
    <Panel title="Avatar Builder" icon={Star}>
      <div className="mb-2 text-xs text-slate-400">
        Attribute points available: <span className="text-emerald-300 font-semibold">{avatar.pool}</span>
      </div>
      <div className="space-y-3">
        {ATTRIBUTE_KEYS.map((a) => (
          <div key={a.id} className="flex items-center justify-between rounded-xl border border-slate-800 p-3 bg-slate-900/60">
            <div className="text-sm">{a.label}</div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => onSpend(a.id, -1)}
                className="p-1 rounded-lg border border-slate-700 hover:bg-slate-800"
              >
                <Minus className="w-4 h-4" />
              </button>
              <div className="w-8 text-center text-sm">{avatar[a.id as keyof Avatar] as number}</div>
              <button
                onClick={() => onSpend(a.id, 1)}
                disabled={avatar.pool <= 0}
                className="p-1 rounded-lg border border-slate-700 hover:bg-slate-800 disabled:opacity-40"
              >
                <Plus className="w-4 h-4" />
              </button>
            </div>
          </div>
        ))}
      </div>
    </Panel>
  );
}

interface LeaderboardPanelProps {
  clusters: Cluster[];
}

function LeaderboardPanel({ clusters }: LeaderboardPanelProps) {
  const resolvedCount = clusters.filter((c) => c.state === "resolved").length;
  const myScore = resolvedCount * 100 + clusters.reduce((acc, c) => acc + (c.state === "resolved" ? c.severity * 25 : 0), 0);
  const others = [
    { name: "ByteBandit", score: 920 },
    { name: "NullPointer", score: 860 },
    { name: "PatchPriest", score: 810 },
  ];
  const board = [{ name: "You", score: myScore, you: true }, ...others].sort((a, b) => b.score - a.score);

  return (
    <Panel title="Top Bots Leaderboard" icon={Trophy}>
      <ul className="space-y-2">
        {board.map((p, idx) => (
          <li
            key={p.name}
            className={classNames(
              "flex items-center justify-between rounded-xl border p-2 px-3",
              p.you ? "border-emerald-600/40 bg-emerald-600/10" : "border-slate-800 bg-slate-900/60"
            )}
          >
            <div className="flex items-center gap-2">
              <div
                className={classNames(
                  "w-6 h-6 rounded-lg text-[11px] flex items-center justify-center",
                  idx === 0 ? "bg-gradient-to-br from-yellow-400 to-amber-500 text-black" : "bg-slate-800"
                )}
              >
                {idx + 1}
              </div>
              <span className={p.you ? "text-emerald-300" : "text-slate-200"}>{p.name}</span>
            </div>
            <div className="text-sm">{p.score}</div>
          </li>
        ))}
      </ul>
    </Panel>
  );
}

interface ProgressChartProps {
  data: { step: number; knerv: number; clusters: number }[];
}

function ProgressChart({ data }: ProgressChartProps) {
  return (
    <Panel title="Ecosystem Progress" icon={Award}>
      {data.length === 0 ? (
        <div className="text-xs text-slate-400">Make a move to start tracking progress.</div>
      ) : (
        <div className="h-36">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={data}>
              <XAxis dataKey="step" tick={{ fontSize: 10, fill: "#94a3b8" }} />
              <YAxis tick={{ fontSize: 10, fill: "#94a3b8" }} />
              <Tooltip
                contentStyle={{
                  background: "#0f172a",
                  border: "1px solid #1e293b",
                  borderRadius: 12,
                  color: "#e2e8f0"
                }}
              />
              <Line type="monotone" dataKey="knerv" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </Panel>
  );
}

// ----------------------- Ecosystem Map -----------------------
interface EcosystemMapProps {
  clusters: Cluster[];
  onOpenCluster: (cluster: Cluster) => void;
  mode: string;
}

function EcosystemMap({ clusters, onOpenCluster, mode }: EcosystemMapProps) {
  return (
    <Panel title="Ecosystem Map" icon={MapIcon}>
      <div className="mb-3 text-xs text-slate-400">
        {mode === "minimal" ? (
          <span>Step 1: Click a node to open its cluster. Fix errors using your bot's skills.</span>
        ) : (
          <span>Step 2: Critical nodes may cascade new errors. Build skills and stabilize clusters.</span>
        )}
      </div>
      <div className="grid grid-cols-4 gap-3">
        {clusters.map((c) => (
          <button
            key={c.id}
            onClick={() => onOpenCluster(c)}
            className={classNames(
              "group relative h-24 rounded-2xl border p-3 text-left overflow-hidden transition focus:outline-none",
              c.state === "resolved"
                ? "border-emerald-600/40 bg-emerald-950/20"
                : "border-slate-800 bg-slate-900/60 hover:border-slate-700"
            )}
          >
            <div className="flex items-start justify-between">
              <div>
                <div className="text-[10px] text-slate-400">{c.id}</div>
                <div className="text-sm font-medium">{c.name}</div>
              </div>
              <span
                className={classNames(
                  "inline-flex items-center gap-1 text-[10px] px-2 py-0.5 rounded-full border",
                  c.state === "critical"
                    ? "border-rose-500/40 text-rose-300"
                    : c.state === "resolved"
                    ? "border-emerald-500/40 text-emerald-300"
                    : "border-amber-500/40 text-amber-300"
                )}
              >
                {c.state}
              </span>
            </div>

            <div
              className={classNames(
                "absolute -bottom-6 -right-6 w-24 h-24 rounded-full blur-xl opacity-60 bg-gradient-to-br",
                severityToColor(c.severity)
              )}
            />
          </button>
        ))}
      </div>
    </Panel>
  );
}

// ----------------------- Cluster View -----------------------
interface ClusterViewProps {
  cluster: Cluster;
  errors: Error[];
  canFix: (error: Error) => boolean;
  onFix: (errorId: string) => void;
  onBack: () => void;
  mode: string;
}

function ClusterView({ cluster, errors, canFix, onFix, onBack, mode }: ClusterViewProps) {
  const unresolved = errors.filter((e) => e.status === "active").length;
  return (
    <Panel title={`${cluster.name} – Node Cluster`} icon={MapIcon}>
      <div className="flex items-center justify-between mb-3">
        <button
          onClick={onBack}
          className="inline-flex items-center gap-1 text-xs px-2 py-1 rounded-lg border border-slate-700 hover:bg-slate-800"
        >
          <ChevronLeft className="w-4 h-4" />
          Back
        </button>
        <div className="flex items-center gap-2 text-xs">
          <span className="text-slate-400">Severity:</span>
          <span
            className={classNames(
              "px-2 py-0.5 rounded-full border",
              cluster.severity === 3
                ? "border-rose-500/40 text-rose-300"
                : cluster.severity === 2
                ? "border-amber-500/40 text-amber-300"
                : "border-yellow-400/40 text-yellow-300"
            )}
          >
            {cluster.severity}
          </span>
          <span className="text-slate-500">•</span>
          <span className="text-slate-400">Unresolved:</span>
          <span className="text-slate-200">{unresolved}</span>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-3">
        {errors.map((e) => (
          <motion.div
            key={e.id}
            layout
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            className={classNames(
              "rounded-xl border p-3",
              e.status === "resolved"
                ? "border-emerald-600/40 bg-emerald-950/10"
                : "border-slate-800 bg-slate-900/60"
            )}
          >
            <div className="flex items-center justify-between">
              <div className="text-sm font-medium">{e.title}</div>
              <span
                className={classNames(
                  "text-[10px] px-2 py-0.5 rounded-full border",
                  e.status === "resolved"
                    ? "border-emerald-500/40 text-emerald-300"
                    : "border-slate-700 text-slate-300"
                )}
              >
                {e.status}
              </span>
            </div>
            <div className="mt-2 text-xs text-slate-300 flex flex-wrap items-center gap-2">
              <span className="text-slate-400">Requires:</span>
              {e.required.map((rid) => {
                const s = SKILL_CATALOG.find((x) => x.id === rid);
                return (
                  <span key={rid} className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full border border-slate-700">
                    {s && <s.icon className="w-3 h-3" />} {s?.name}
                  </span>
                );
              })}
              <span className="text-slate-500">•</span>
              <span className="text-slate-400">Complexity</span>
              <span className="px-2 py-0.5 rounded border border-slate-700">Lv.{e.complexity}</span>
            </div>
            {e.status === "active" && (
              <div className="mt-3 flex items-center justify-between">
                <div className="text-[11px] text-slate-400">
                  {canFix(e) ? "Ready to fix." : "Insufficient skills – level up required skills."}
                </div>
                <button
                  onClick={() => onFix(e.id)}
                  disabled={!canFix(e)}
                  className="text-xs px-3 py-1.5 rounded-lg border border-emerald-600/50 bg-emerald-600/20 hover:bg-emerald-600/30 disabled:opacity-40"
                >
                  Attempt Fix
                </button>
              </div>
            )}
          </motion.div>
        ))}
      </div>

      {mode === "full" && (
        <div className="mt-4 text-[11px] text-slate-400">
          Cascades: Fixes may trigger new errors. Stabilize all nodes to complete the cluster.
        </div>
      )}
    </Panel>
  );
}
