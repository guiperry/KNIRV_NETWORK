function Footer() {
    try {
        return (
            <footer className="bg-[var(--bg-black)] border-t border-gray-900 py-12" data-name="footer" data-file="components/Footer.js">
                <div className="container mx-auto px-6">
                    <div className="grid grid-cols-1 md:grid-cols-4 gap-8 mb-8">
                        <div className="col-span-1 md:col-span-2">
                            <div className="flex items-center gap-2 mb-4">
                                <div className="icon-terminal text-[var(--accent-blue)] text-2xl"></div>
                                <span className="font-mono font-bold text-xl tracking-widest text-white">KNIRV</span>
                            </div>
                            <p className="text-[var(--text-gray)] text-sm max-w-sm mb-4">
                                The actuarial bug bounty syndicate. Pricing risk with data, not negotiation.
                            </p>
                        </div>
                        <div>
                            <h4 className="font-bold text-white mb-4 font-mono">Syndicate</h4>
                            <ul className="space-y-2 text-sm text-[var(--text-gray)] font-mono">
                                <li><a href="#" className="hover:text-[var(--accent-blue)]">Initialize Node</a></li>
                                <li><a href="#" className="hover:text-[var(--accent-blue)]">Underwriting Docs</a></li>
                                <li><a href="#" className="hover:text-[var(--accent-blue)]">Smart Contracts</a></li>
                            </ul>
                        </div>
                        <div>
                            <h4 className="font-bold text-white mb-4 font-mono">Legal</h4>
                            <ul className="space-y-2 text-sm text-[var(--text-gray)] font-mono">
                                <li><a href="#" className="hover:text-[var(--accent-blue)]">Terms of Service</a></li>
                                <li><a href="#" className="hover:text-[var(--accent-blue)]">Privacy Policy</a></li>
                            </ul>
                        </div>
                    </div>
                    <div className="pt-8 border-t border-gray-900 text-center text-xs text-gray-600 font-mono">
                        &copy; 2026 KNIRV Syndicate. All rights reserved. Not financial advice.
                    </div>
                </div>
            </footer>
        );
    } catch (error) {
        console.error('Footer component error:', error);
        return null;
    }
}