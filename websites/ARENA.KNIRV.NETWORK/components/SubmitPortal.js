function SubmitPortal() {
    try {
        const [target, setTarget] = React.useState('');
        const [vulnClass, setVulnClass] = React.useState('');
        const [score, setScore] = React.useState(0);
        const [estimatedPayout, setEstimatedPayout] = React.useState([0, 0]);

        // Mock calculation based on selections
        React.useEffect(() => {
            if (target && vulnClass) {
                let baseScore = 50;
                let baseMin = 1000;
                let baseMax = 5000;

                // Adjust based on Target
                if (target === 'rust-hypervisor') { baseScore += 20; baseMin += 15000; baseMax += 80000; }
                if (target === 'graphql-core') { baseScore += 15; baseMin += 8000; baseMax += 40000; }
                if (target === 'jvm-runtime') { baseScore += 10; baseMin += 5000; baseMax += 25000; }

                // Adjust based on Vulnerability Class
                if (vulnClass === 'memory-corruption') { baseScore += 24; baseMin += 10000; baseMax += 100000; }
                if (vulnClass === 'auth-bypass') { baseScore += 18; baseMin += 5000; baseMax += 50000; }
                if (vulnClass === 'rce') { baseScore += 25; baseMin += 20000; baseMax += 150000; }

                setScore(Math.min(baseScore, 99.9).toFixed(1));
                setEstimatedPayout([baseMin, baseMax]);
            } else {
                setScore(0);
                setEstimatedPayout([0, 0]);
            }
        }, [target, vulnClass]);

        return (
            <section className="py-12 bg-[var(--bg-black)] relative" data-name="submit-portal" data-file="components/SubmitPortal.js">
                <div className="container mx-auto px-6 max-w-6xl">
                    <div className="mb-10 border-b border-gray-800 pb-6">
                        <h1 className="text-4xl font-bold font-mono tracking-tight mb-2">Vulnerability Submission Protocol</h1>
                        <p className="text-[var(--text-gray)]">Execute smart contract for automated actuarial pricing and settlement.</p>
                    </div>

                    <div className="grid md:grid-cols-3 gap-8">
                        {/* Submission Form */}
                        <div className="md:col-span-2 space-y-6">
                            <div className="glass-panel p-8">
                                <h3 className="font-mono text-xl font-bold text-[var(--accent-blue)] mb-6 flex items-center gap-2">
                                    <div className="icon-file-code"></div> 
                                    Payload Parameters
                                </h3>
                                
                                <form className="space-y-6" onSubmit={(e) => e.preventDefault()}>
                                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                                        <div>
                                            <label className="block text-sm font-mono text-gray-400 mb-2">Target Environment</label>
                                            <select 
                                                className="w-full bg-black border border-gray-800 rounded p-3 text-white focus:border-[var(--accent-blue)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-blue)] font-mono text-sm"
                                                value={target}
                                                onChange={(e) => setTarget(e.target.value)}
                                            >
                                                <option value="">Select Target...</option>
                                                <option value="rust-hypervisor">Rust Hypervisor</option>
                                                <option value="graphql-core">GraphQL API Core</option>
                                                <option value="jvm-runtime">JVM Runtime</option>
                                                <option value="cosmos-sdk">Cosmos SDK Node</option>
                                            </select>
                                        </div>
                                        <div>
                                            <label className="block text-sm font-mono text-gray-400 mb-2">Vulnerability Class</label>
                                            <select 
                                                className="w-full bg-black border border-gray-800 rounded p-3 text-white focus:border-[var(--accent-blue)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-blue)] font-mono text-sm"
                                                value={vulnClass}
                                                onChange={(e) => setVulnClass(e.target.value)}
                                            >
                                                <option value="">Select Class...</option>
                                                <option value="memory-corruption">Memory Corruption</option>
                                                <option value="auth-bypass">Authentication Bypass</option>
                                                <option value="rce">Remote Code Execution</option>
                                                <option value="deserialization">Deserialization</option>
                                            </select>
                                        </div>
                                    </div>

                                    <div>
                                        <label className="block text-sm font-mono text-gray-400 mb-2">Technical Description</label>
                                        <textarea 
                                            rows="4" 
                                            className="w-full bg-black border border-gray-800 rounded p-3 text-white focus:border-[var(--accent-blue)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-blue)] font-mono text-sm resize-none"
                                            placeholder="Provide technical details, blast radius, and prerequisites..."
                                        ></textarea>
                                    </div>

                                    <div>
                                        <label className="block text-sm font-mono text-gray-400 mb-2">Proof of Concept (Encrypted)</label>
                                        <div className="border-2 border-dashed border-gray-800 rounded-lg p-8 text-center hover:border-[var(--accent-blue)] transition-colors cursor-pointer bg-black bg-opacity-50">
                                            <div className="icon-cloud-upload text-3xl text-gray-500 mx-auto mb-3"></div>
                                            <p className="font-mono text-sm text-gray-400">Drag & drop your PoC archive or click to browse.</p>
                                            <p className="font-mono text-xs text-gray-600 mt-2">Files are encrypted with the syndicate's public key before transmission.</p>
                                        </div>
                                    </div>

                                    <button className="btn-primary w-full flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed" disabled={!target || !vulnClass}>
                                        <div className="icon-lock"></div> Encrypt & Submit to Network
                                    </button>
                                </form>
                            </div>
                        </div>

                        {/* Live Actuarial Pricing Sidebar */}
                        <div className="md:col-span-1">
                            <div className="sticky top-24 space-y-6">
                                <div className="glass-panel p-6 border-t-2 border-t-[var(--accent-blue)]">
                                    <h4 className="font-mono text-sm text-gray-400 mb-4">LIVE SYNDICATE ESTIMATE</h4>
                                    
                                    <div className="mb-6">
                                        <div className="text-xs text-gray-500 font-mono mb-1">Projected Risk Score</div>
                                        <div className="text-4xl font-bold font-mono flex items-end gap-2 text-white">
                                            {score > 0 ? score : '0.0'}
                                            <span className="text-sm font-normal text-gray-500 mb-1">/ 100</span>
                                        </div>
                                    </div>

                                    <div className="mb-6">
                                        <div className="text-xs text-gray-500 font-mono mb-1">Estimated Dynamic Range</div>
                                        <div className="text-xl font-bold font-mono text-green-400">
                                            ${estimatedPayout[0].toLocaleString()} - ${estimatedPayout[1].toLocaleString()}
                                        </div>
                                    </div>

                                    <div className="space-y-3 pt-4 border-t border-gray-800">
                                        <div className="flex justify-between items-center text-xs font-mono">
                                            <span className="text-gray-500">Current Network Liquidity:</span>
                                            <span className="text-white">$14.2M</span>
                                        </div>
                                        <div className="flex justify-between items-center text-xs font-mono">
                                            <span className="text-gray-500">Active Node Operators:</span>
                                            <span className="text-white">412</span>
                                        </div>
                                        <div className="flex justify-between items-center text-xs font-mono">
                                            <span className="text-gray-500">Oracle Status:</span>
                                            <span className="text-[var(--accent-blue)] flex items-center gap-1">
                                                <div className="w-2 h-2 bg-[var(--accent-blue)] rounded-full animate-pulse"></div>
                                                SYNCED
                                            </span>
                                        </div>
                                    </div>
                                </div>
                                
                                <div className="bg-[#0f172a] border border-gray-800 rounded-lg p-5">
                                    <div className="flex gap-3">
                                        <div className="icon-info text-gray-400 text-xl flex-shrink-0"></div>
                                        <p className="text-xs text-gray-400 font-mono leading-relaxed">
                                            Final payout is calculated upon cryptographic verification of the PoC. Settlement occurs instantly via smart contract distributing risk across participating nodes based on their exact telemetry matching your exploit's signature.
                                        </p>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </section>
        );
    } catch (error) {
        console.error('SubmitPortal component error:', error);
        return null;
    }
}