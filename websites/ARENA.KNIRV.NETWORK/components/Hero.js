function Hero({ arenaURL }) {
    try {
        return (
            <section className="relative pt-40 pb-20 overflow-hidden" data-name="hero" data-file="components/Hero.js">
                <div className="absolute inset-0 z-0">
                    <div className="absolute inset-0 bg-gradient-to-b from-[var(--bg-navy)] to-[var(--bg-black)] opacity-80"></div>
                    <div className="w-full h-full" style={{
                        backgroundImage: `radial-gradient(var(--accent-blue) 1px, transparent 1px)`,
                        backgroundSize: '40px 40px',
                        opacity: 0.1
                    }}></div>
                </div>
                
                <div className="container mx-auto px-6 relative z-10 max-w-5xl text-center">
                    <div className="inline-block border border-[var(--accent-blue)] text-[var(--accent-blue)] px-4 py-1 rounded-full text-xs font-mono mb-8 bg-[var(--accent-blue)] bg-opacity-10">
                        V0.9.1 SYNDICATE ONLINE
                    </div>
                    <h1 className="text-5xl md:text-7xl font-bold mb-6 leading-tight">
                        The First <span className="text-[var(--accent-blue)]">Actuarial</span><br /> Bug Bounty Syndicate.
                    </h1>
                    <p className="text-xl md:text-2xl text-[var(--text-gray)] mb-10 max-w-3xl mx-auto">
                        No human negotiation. No vendor discretion. <br className="hidden md:block"/>
                        Payouts priced automatically from real eBPF telemetry and on-chain risk pools.
                    </p>
                    <div className="flex flex-col sm:flex-row justify-center gap-4">
                        <a href={arenaURL} className="btn-primary flex items-center justify-center gap-2">
                            <div className="icon-swords"></div> Enter Arena
                        </a>
                        <button className="btn-outline flex items-center justify-center gap-2">
                            <div className="icon-book-open"></div> Read Protocol Docs
                        </button>
                    </div>
                    
                    <div className="mt-20 grid grid-cols-1 md:grid-cols-3 gap-6 text-left">
                        <div className="glass-panel p-6 border-t-2 border-t-[var(--accent-blue)]">
                            <div className="icon-activity text-[var(--accent-blue)] text-3xl mb-4"></div>
                            <h3 className="text-lg font-bold mb-2">eBPF Telemetry</h3>
                            <p className="text-[var(--text-gray)] text-sm">Underwriting based on observed exploit attempt rates and blast radius data from D-TEN network nodes.</p>
                        </div>
                        <div className="glass-panel p-6 border-t-2 border-t-[var(--accent-blue)]">
                            <div className="icon-scale text-[var(--accent-blue)] text-3xl mb-4"></div>
                            <h3 className="text-lg font-bold mb-2">Dynamic Pricing</h3>
                            <p className="text-[var(--text-gray)] text-sm">No gut feelings. Smart contracts automatically calculate payouts using live patch adoption curves and real-time risk tables.</p>
                        </div>
                        <div className="glass-panel p-6 border-t-2 border-t-[var(--accent-blue)]">
                            <div className="icon-network text-[var(--accent-blue)] text-3xl mb-4"></div>
                            <h3 className="text-lg font-bold mb-2">Distributed Risk</h3>
                            <p className="text-[var(--text-gray)] text-sm">Risk exposure distributes proportionally across node operators. Settlement is anchored instantly on-chain.</p>
                        </div>
                    </div>
                </div>
            </section>
        );
    } catch (error) {
        console.error('Hero component error:', error);
        return null;
    }
}
