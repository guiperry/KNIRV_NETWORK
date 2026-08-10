function Arena() {
    try {
        const bounties = [
            { id: "KNV-092", class: "Memory Corruption", target: "Rust Hypervisor", riskScore: 94.2, base: 25000, max: 185000, trend: "up" },
            { id: "KNV-114", class: "Auth Bypass", target: "GraphQL API Core", riskScore: 88.5, base: 10000, max: 92000, trend: "up" },
            { id: "KNV-047", class: "Deserialization", target: "JVM Runtime", riskScore: 71.8, base: 5000, max: 45000, trend: "down" },
            { id: "KNV-103", class: "Race Condition", target: "Cosmos SDK Node", riskScore: 82.1, base: 15000, max: 110000, trend: "up" },
        ];

        return (
            <section id="arena" className="py-24 bg-[var(--bg-navy)] relative" data-name="arena" data-file="components/Arena.js">
                <div className="container mx-auto px-6 max-w-6xl">
                    <div className="flex flex-col md:flex-row justify-between items-end mb-12">
                        <div>
                            <h2 className="text-3xl md:text-5xl font-bold mb-4 font-mono uppercase tracking-tight text-white flex items-center gap-3">
                                <div className="icon-gamepad-2 text-[var(--accent-blue)]"></div>
                                Live Arena
                            </h2>
                            <p className="text-[var(--text-gray)] max-w-xl">Real-time risk pooling. Bounties fluctuate dynamically based on live node telemetry and syndicate staking.</p>
                        </div>
                        <button className="btn-primary mt-6 md:mt-0">Connect to Network</button>
                    </div>

                    <div className="overflow-x-auto">
                        <table className="w-full text-left border-collapse">
                            <thead>
                                <tr className="border-b border-gray-800 text-[var(--accent-blue)] font-mono text-sm uppercase tracking-wider">
                                    <th className="py-4 px-4">Contract ID</th>
                                    <th className="py-4 px-4">Vulnerability Class</th>
                                    <th className="py-4 px-4">Target Environment</th>
                                    <th className="py-4 px-4">Actuarial Risk Score</th>
                                    <th className="py-4 px-4 text-right">Dynamic Payout Range</th>
                                    <th className="py-4 px-4"></th>
                                </tr>
                            </thead>
                            <tbody className="font-mono text-sm">
                                {bounties.map((bounty, idx) => (
                                    <tr key={idx} className="border-b border-gray-800 hover:bg-white hover:bg-opacity-5 transition-colors group cursor-pointer">
                                        <td className="py-5 px-4 text-gray-400">{bounty.id}</td>
                                        <td className="py-5 px-4 font-bold text-white">{bounty.class}</td>
                                        <td className="py-5 px-4 text-[var(--text-gray)]">{bounty.target}</td>
                                        <td className="py-5 px-4">
                                            <div className="flex items-center gap-2">
                                                <div className={`w-2 h-2 rounded-full ${bounty.riskScore > 90 ? 'bg-red-500' : bounty.riskScore > 80 ? 'bg-orange-500' : 'bg-yellow-500'}`}></div>
                                                {bounty.riskScore}
                                            </div>
                                        </td>
                                        <td className="py-5 px-4 text-right text-green-400">
                                            ${bounty.base.toLocaleString()} - ${bounty.max.toLocaleString()}
                                        </td>
                                        <td className="py-5 px-4 text-right">
                                            <a href="submit.html" className="inline-block border border-gray-700 hover:border-[var(--accent-blue)] text-gray-300 hover:text-[var(--accent-blue)] px-3 py-1 rounded transition-colors text-xs">Submit PoC</a>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                </div>
            </section>
        );
    } catch (error) {
        console.error('Arena component error:', error);
        return null;
    }
}