function Arena({ arenaURL }) {
    const [state, setState] = React.useState({ loading: true, error: '', classes: [], pools: [] });

    React.useEffect(() => {
        const base = window.KNIRV_ACTUARIAL_API || 'http://localhost:8082/api/v1/actuarial';
        Promise.all([fetch(`${base}/risk-classes`), fetch(`${base}/pools`)])
            .then(async ([classes, pools]) => {
                if (!classes.ok || !pools.ok) throw new Error('Actuarial API unavailable');
                setState({ loading: false, error: '', classes: await classes.json(), pools: await pools.json() });
            })
            .catch(() => setState({ loading: false, error: 'Live syndicate data is temporarily unavailable.', classes: [], pools: [] }));
    }, []);

    const poolsByClass = new Map(state.pools.map(pool => [pool.risk_class_id, pool]));
    return <section id="syndicate" className="py-24 bg-[var(--bg-navy)] relative" data-name="syndicate-board">
        <div className="container mx-auto px-6 max-w-6xl">
            <div className="flex flex-col md:flex-row justify-between items-end mb-12">
                <div><h2 className="text-3xl md:text-5xl font-bold mb-4 font-mono uppercase tracking-tight text-white">Live Syndicate Board</h2><p className="text-[var(--text-gray)] max-w-xl">Backend-owned postings and pool capacity across curated code errors and security exploits.</p></div>
                <a href={arenaURL} className="btn-primary mt-6 md:mt-0">Join the Syndicate</a>
            </div>
            {state.loading && <p className="text-gray-400 font-mono">Loading live postings…</p>}
            {state.error && <p className="text-red-300 font-mono">{state.error}</p>}
            {!state.loading && !state.error && <div className="overflow-x-auto"><table className="w-full text-left border-collapse"><thead><tr className="border-b border-gray-800 text-[var(--accent-blue)] font-mono text-sm uppercase tracking-wider"><th className="py-4 px-4">Posting</th><th className="py-4 px-4">Domain</th><th className="py-4 px-4">Status</th><th className="py-4 px-4 text-right">Available capacity</th></tr></thead><tbody className="font-mono text-sm">{state.classes.map(riskClass => { const pool = poolsByClass.get(riskClass.id); const available = pool ? Math.max(0, pool.liquid_balance - pool.reserved_balance) : 0; return <tr key={riskClass.id} className="border-b border-gray-800"><td className="py-5 px-4"><strong className="text-white">{riskClass.display_name}</strong><p className="text-gray-400 mt-1">{riskClass.description}</p></td><td className="py-5 px-4 text-gray-300">{riskClass.domain.replace('_', ' ')}</td><td className="py-5 px-4 text-gray-300">{riskClass.status}</td><td className="py-5 px-4 text-right text-green-400">{available.toLocaleString()} NRN</td></tr>; })}</tbody></table></div>}
        </div>
    </section>;
}
