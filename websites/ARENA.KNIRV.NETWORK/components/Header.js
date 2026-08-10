function Header() {
    try {
        return (
            <header className="fixed top-0 w-full z-50 glass-panel !border-t-0 !border-l-0 !border-r-0 !rounded-none" data-name="header" data-file="components/Header.js">
                <div className="container mx-auto px-6 py-4 flex justify-between items-center">
                    <a href="index.html" className="flex items-center gap-2 hover:opacity-80 transition-opacity">
                        <div className="icon-terminal text-[var(--accent-blue)] text-2xl"></div>
                        <span className="font-mono font-bold text-xl tracking-widest">KNIRV</span>
                    </a>
                    <nav className="hidden md:flex gap-8 items-center font-mono text-sm">
                        <a href="index.html#features" className="text-[var(--text-gray)] hover:text-[var(--accent-blue)] transition-colors">Actuarial Logic</a>
                        <a href="index.html#arena" className="text-[var(--text-gray)] hover:text-[var(--accent-blue)] transition-colors">The Arena</a>
                        <a href="submit.html" className="text-[var(--accent-blue)] hover:text-white transition-colors border-b border-transparent hover:border-[var(--accent-blue)]">Submit PoC</a>
                    </nav>
                    <div>
                        <button className="btn-outline text-sm py-2 hidden sm:block">Initialize Node</button>
                    </div>
                </div>
            </header>
        );
    } catch (error) {
        console.error('Header component error:', error);
        return null;
    }
}