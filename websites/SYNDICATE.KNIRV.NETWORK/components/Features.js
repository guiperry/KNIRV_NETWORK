function Features() {
    try {
        return (
            <section id="features" className="py-24 bg-[var(--bg-black)] border-t border-[var(--bg-navy)]" data-name="features" data-file="components/Features.js">
                <div className="container mx-auto px-6 max-w-6xl">
                    <div className="text-center mb-16">
                        <h2 className="text-3xl md:text-5xl font-bold mb-6">The Market is <span className="text-[var(--accent-blue)]">Broken.</span></h2>
                        <p className="text-[var(--text-gray)] text-lg max-w-2xl mx-auto">Current bug bounty programs rely on flat fees derived from gut feelings, not risk data. We're replacing arbitrary negotiations with hard actuarial math.</p>
                    </div>

                    <div className="grid md:grid-cols-2 gap-12 items-center">
                        <div className="space-y-8">
                            <div className="flex gap-4">
                                <div className="mt-1">
                                    <div className="w-8 h-8 rounded bg-red-900 bg-opacity-30 flex items-center justify-center border border-red-500 border-opacity-50">
                                        <div className="icon-x text-red-500"></div>
                                    </div>
                                </div>
                                <div>
                                    <h4 className="text-xl font-bold mb-2">The Old Way</h4>
                                    <p className="text-[var(--text-gray)]">"We'll pay $5k for an auth bypass, maybe $10k if the vendor feels generous." Endless negotiations, out-of-scope arguments, and delayed wire transfers.</p>
                                </div>
                            </div>
                            <div className="flex gap-4">
                                <div className="mt-1">
                                    <div className="w-8 h-8 rounded bg-[var(--accent-blue)] bg-opacity-20 flex items-center justify-center border border-[var(--accent-blue)]">
                                        <div className="icon-check text-[var(--accent-blue)]"></div>
                                    </div>
                                </div>
                                <div>
                                    <h4 className="text-xl font-bold mb-2">The KNIRV Way</h4>
                                    <p className="text-[var(--text-gray)]">Submit PoC. Contract executes. Payout scales automatically based on blast radius, telemetry risk scores, and current syndicate liquidity. Guaranteed execution.</p>
                                </div>
                            </div>
                        </div>

                        <div className="glass-panel p-8 relative overflow-hidden">
                            <div className="absolute top-0 right-0 w-32 h-32 bg-[var(--accent-blue)] opacity-5 blur-3xl rounded-full"></div>
                            <h3 className="font-mono text-[var(--accent-blue)] mb-6 border-b border-gray-800 pb-4">TARGET AUDIENCE</h3>
                            
                            <div className="space-y-6">
                                <div>
                                    <h4 className="font-bold flex items-center gap-2 mb-1">
                                        <div className="icon-shield-check text-[var(--accent-blue)]"></div>
                                        Cyber Insurance Carriers
                                    </h4>
                                    <p className="text-sm text-[var(--text-gray)]">Finally build accurate actuarial tables for software risk based on granular, real-world exploitation telemetry rather than historical guess-work.</p>
                                </div>
                                <div>
                                    <h4 className="font-bold flex items-center gap-2 mb-1">
                                        <div className="icon-chart-line text-[var(--accent-blue)]"></div>
                                        Quant Hedge Funds
                                    </h4>
                                    <p className="text-sm text-[var(--text-gray)]">Write policies and underwrite technical risk with predictable data sets. Profit from accurate risk assessment on memory corruption and JVM deserialization flaws.</p>
                                </div>
                                <div>
                                    <h4 className="font-bold flex items-center gap-2 mb-1">
                                        <div className="icon-terminal-square text-[var(--accent-blue)]"></div>
                                        Elite Researchers
                                    </h4>
                                    <p className="text-sm text-[var(--text-gray)]">Get paid exactly what your exploit is mathematically worth to the network. Stop begging vendors for fair compensation.</p>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </section>
        );
    } catch (error) {
        console.error('Features component error:', error);
        return null;
    }
}