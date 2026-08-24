import React, { useState } from 'react';
import { FileCode2, Search, Wand2 } from 'lucide-react';

const tree = [
  { pkg: 'com.targetapp.android', classes: ['MainActivity', 'a.b.c (obfuscated)', 'NetworkModule'] },
  { pkg: 'com.targetapp.android.crypto', classes: ['KeyStoreHelper', 'a.d.e (obfuscated)'] },
];

const java: Record<string, string> = {
  MainActivity: `public class MainActivity extends AppCompatActivity {
    private NetworkModule network;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);
        this.network = new NetworkModule(this);
        if (!KeyStoreHelper.hasValidLicense(this)) {
            finish();
            return;
        }
        network.syncAgentRegistry();
    }
}`,
  NetworkModule: `public class NetworkModule {
    private static final String BASE_URL = "https://api.targetapp.local/";

    public void syncAgentRegistry() {
        OkHttpClient client = new OkHttpClient.Builder()
            .certificatePinner(Pinner.build())
            .build();
        Request req = new Request.Builder().url(BASE_URL + "v1/agents").build();
        client.newCall(req).enqueue(this.callback);
    }
}`,
};

const smali: Record<string, string> = {
  MainActivity: `.method protected onCreate(Landroid/os/Bundle;)V
    .locals 1
    invoke-super {p0, p1}, Landroidx/appcompat/app/AppCompatActivity;->onCreate(Landroid/os/Bundle;)V
    const v0, 0x7f0b001c
    invoke-virtual {p0, v0}, Lcom/targetapp/android/MainActivity;->setContentView(I)V
    new-instance v0, Lcom/targetapp/android/NetworkModule;
    invoke-direct {v0, p0}, Lcom/targetapp/android/NetworkModule;-><init>(Landroid/content/Context;)V
    iput-object v0, p0, Lcom/targetapp/android/MainActivity;->network:Lcom/targetapp/android/NetworkModule;
.end method`,
};

const Jadx: React.FC = () => {
  const [selected, setSelected] = useState('MainActivity');
  const [view, setView] = useState<'java' | 'smali'>('java');
  const [deobfuscate, setDeobfuscate] = useState(true);
  const [query, setQuery] = useState('');

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-orange-500/20 rounded-lg">
          <FileCode2 className="w-6 h-6 text-orange-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">JADX</h1>
          <p className="text-slate-400 text-sm font-mono">target-app-release.apk · min-sdk 24 · dex2jar path skipped (native DEX input)</p>
        </div>
      </div>

      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center space-x-2">
          <Search className="w-3.5 h-3.5 text-slate-500" />
          <input
            value={query}
            onChange={e => setQuery(e.target.value)}
            placeholder="search classes / methods / strings..."
            className="w-72 bg-slate-800/50 border border-slate-700/50 rounded-lg px-3 py-1.5 text-sm font-mono text-slate-200"
          />
        </div>
        <label className="flex items-center space-x-2 text-sm text-slate-300">
          <Wand2 className="w-3.5 h-3.5 text-orange-400" />
          <span className="font-mono text-xs">deobfuscation</span>
          <input type="checkbox" checked={deobfuscate} onChange={e => setDeobfuscate(e.target.checked)} className="accent-orange-500" />
        </label>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
        <div className="xl:col-span-1 bg-slate-800/50 border border-slate-700/50 rounded-lg p-3">
          <div className="text-xs text-slate-600 uppercase mb-2">Sources</div>
          {tree.map(node => (
            <div key={node.pkg} className="mb-2">
              <div className="text-xs text-slate-500 font-mono px-1 mb-1">{node.pkg}</div>
              <div className="ml-2 space-y-0.5">
                {node.classes
                  .filter(c => c.toLowerCase().includes(query.toLowerCase()))
                  .map(c => {
                    const clean = c.replace(' (obfuscated)', '');
                    const isObf = c.includes('obfuscated');
                    return (
                      <button
                        key={c}
                        onClick={() => !isObf && setSelected(clean)}
                        className={`block w-full text-left px-2 py-1 rounded font-mono text-xs ${
                          isObf
                            ? 'text-slate-600 italic cursor-not-allowed'
                            : selected === clean
                              ? 'bg-orange-500/15 text-orange-300'
                              : 'text-slate-400 hover:text-white'
                        }`}
                      >
                        {isObf && deobfuscate ? c.replace('a.b.c', 'C0142a').replace('a.d.e', 'C0198e') : c}
                      </button>
                    );
                  })}
              </div>
            </div>
          ))}
        </div>

        <div className="xl:col-span-2 bg-slate-800/50 border border-slate-700/50 rounded-lg p-3">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-slate-600 uppercase">{selected}.java</span>
            <div className="flex space-x-1">
              {(['java', 'smali'] as const).map(v => (
                <button
                  key={v}
                  onClick={() => setView(v)}
                  className={`text-xs px-2 py-1 rounded font-mono ${
                    view === v ? 'bg-orange-500/20 text-orange-300' : 'text-slate-500 hover:text-white'
                  }`}
                >
                  {v}
                </button>
              ))}
            </div>
          </div>
          <pre className="font-mono text-xs text-slate-200 whitespace-pre-wrap leading-relaxed max-h-[420px] overflow-y-auto">
            {(view === 'java' ? java[selected] : smali[selected]) ?? `// ${view} not cached for ${selected}`}
          </pre>
        </div>
      </div>
    </div>
  );
};

export default Jadx;
