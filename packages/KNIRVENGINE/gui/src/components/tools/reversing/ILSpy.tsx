import React, { useState } from 'react';
import { Layers3, ChevronRight, ChevronDown } from 'lucide-react';

const tree = {
  assembly: 'TargetApp.Core.dll',
  namespaces: [
    {
      name: 'TargetApp.Core.Licensing',
      classes: ['LicenseValidator', 'TokenCache'],
    },
    {
      name: 'TargetApp.Core.Net',
      classes: ['ApiClient', 'CertificatePinner'],
    },
  ],
};

const csharp: Record<string, string> = {
  LicenseValidator: `public class LicenseValidator
{
    private static readonly byte[] Secret = Convert.FromBase64String("kQ2f...==");

    public bool Verify(string token)
    {
        if (string.IsNullOrEmpty(token) || token.Length < 44)
        {
            return false;
        }
        byte[] payload = Convert.FromBase64String(token);
        using HMACSHA256 hmac = new HMACSHA256(Secret);
        byte[] computed = hmac.ComputeHash(payload, 0, payload.Length - 32);
        return CryptographicOperations.FixedTimeEquals(
            computed, payload.AsSpan(payload.Length - 32).ToArray());
    }
}`,
  CertificatePinner: `public class CertificatePinner
{
    private static readonly string[] PinnedSha256 = new[]
    {
        "sha256/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
    };

    public bool Validate(X509Certificate2 cert)
    {
        string hash = ComputeSpkiHash(cert);
        return PinnedSha256.Contains(hash);
    }
}`,
};

const il: Record<string, string> = {
  LicenseValidator: `.method public hidebysig instance bool Verify (string token) cil managed
{
    .maxstack 3
    IL_0000: ldarg.1
    IL_0001: call bool [System.Runtime]System.String::IsNullOrEmpty(string)
    IL_0006: brtrue.s IL_0030
    IL_0008: ldarg.1
    IL_0009: callvirt instance int32 [System.Runtime]System.String::get_Length()
    IL_000e: ldc.i4.s 44
    IL_0010: blt.s IL_0030
    ...
}`,
};

const ILSpy: React.FC = () => {
  const [expanded, setExpanded] = useState<Set<string>>(new Set(['TargetApp.Core.Licensing']));
  const [selected, setSelected] = useState('LicenseValidator');
  const [lang, setLang] = useState<'csharp' | 'il'>('csharp');

  const toggle = (ns: string) => {
    setExpanded(prev => {
      const next = new Set(prev);
      if (next.has(ns)) {
        next.delete(ns);
      } else {
        next.add(ns);
      }
      return next;
    });
  };

  const source = lang === 'csharp' ? csharp[selected] : il[selected];

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-blue-500/20 rounded-lg">
          <Layers3 className="w-6 h-6 text-blue-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">ILSpy</h1>
          <p className="text-slate-400 text-sm font-mono">{tree.assembly} · .NET 8.0 · PDB: embedded</p>
        </div>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
        {/* assembly tree */}
        <div className="xl:col-span-1 bg-slate-800/50 border border-slate-700/50 rounded-lg p-3">
          <div className="text-xs text-slate-600 uppercase mb-2">Assembly Explorer</div>
          <div className="font-mono text-xs">
            <div className="text-slate-300 mb-1">{tree.assembly}</div>
            {tree.namespaces.map(ns => (
              <div key={ns.name} className="ml-2">
                <button onClick={() => toggle(ns.name)} className="flex items-center space-x-1 text-slate-400 hover:text-white py-1">
                  {expanded.has(ns.name) ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
                  <span>{ns.name}</span>
                </button>
                {expanded.has(ns.name) && (
                  <div className="ml-5 space-y-0.5">
                    {ns.classes.map(cls => (
                      <button
                        key={cls}
                        onClick={() => setSelected(cls)}
                        className={`block w-full text-left px-1.5 py-1 rounded ${
                          selected === cls ? 'bg-blue-500/15 text-blue-300' : 'text-slate-500 hover:text-white'
                        }`}
                      >
                        {cls}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>

        {/* decompiled source */}
        <div className="xl:col-span-2 bg-slate-800/50 border border-slate-700/50 rounded-lg p-3">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-slate-600 uppercase">{selected}</span>
            <div className="flex space-x-1">
              {(['csharp', 'il'] as const).map(l => (
                <button
                  key={l}
                  onClick={() => setLang(l)}
                  className={`text-xs px-2 py-1 rounded font-mono ${
                    lang === l ? 'bg-blue-500/20 text-blue-300' : 'text-slate-500 hover:text-white'
                  }`}
                >
                  {l === 'csharp' ? 'C#' : 'IL'}
                </button>
              ))}
            </div>
          </div>
          <pre className="font-mono text-xs text-slate-200 whitespace-pre-wrap leading-relaxed max-h-[420px] overflow-y-auto">
            {source ?? `// no ${lang === 'csharp' ? 'C#' : 'IL'} view cached for ${selected}`}
          </pre>
        </div>
      </div>
    </div>
  );
};

export default ILSpy;
