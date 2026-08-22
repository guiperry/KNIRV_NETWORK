// Important: DO NOT remove this `ErrorBoundary` component.
class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error) {
    return { hasError: true, error };
  }

  componentDidCatch(error, errorInfo) {
    console.error('ErrorBoundary caught an error:', error, errorInfo.componentStack);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="min-h-screen flex items-center justify-center bg-[var(--bg-black)] text-white">
          <div className="text-center glass-panel p-8">
            <h1 className="text-2xl font-bold text-red-500 mb-4 font-mono">SYSTEM FAILURE</h1>
            <p className="text-[var(--text-gray)] mb-4 font-mono">An unexpected error occurred in the syndicate UI.</p>
            <button
              onClick={() => window.location.reload()}
              className="btn-primary"
            >
              Reboot Terminal
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

function App() {
  try {
    const arenaGateways = {
      testnet: 'https://testnet-gateway.knirv.network',
      mainnet: 'https://gateway.knirv.network',
    };
    const [selectedGateway, setSelectedGateway] = React.useState('testnet');
    const arenaURL = `${arenaGateways[selectedGateway]}/arena/?arena=1`;

    return (
      <div className="min-h-screen flex flex-col" data-name="app" data-file="app.js">
        <Header
          selectedGateway={selectedGateway}
          onGatewayChange={setSelectedGateway}
          arenaURL={arenaURL}
        />
        <main className="flex-grow">
            <Hero arenaURL={arenaURL} />
            <Features />
            <Arena arenaURL={arenaURL} />
        </main>
        <Footer arenaURL={arenaURL} />
      </div>
    );
  } catch (error) {
    console.error('App component error:', error);
    return null;
  }
}

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <ErrorBoundary>
    <App />
  </ErrorBoundary>
);
