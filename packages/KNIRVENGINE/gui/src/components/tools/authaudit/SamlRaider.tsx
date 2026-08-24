import React, { useState } from 'react';
import { FileSignature, Play } from 'lucide-react';

const sampleAssertion = `<samlp:Response ID="_a1b2c3" IssueInstant="2026-08-24T10:00:00Z">
  <saml:Assertion ID="_d4e5f6">
    <saml:Issuer>https://idp.knirv.network</saml:Issuer>
    <ds:Signature>
      <ds:SignedInfo>...</ds:SignedInfo>
      <ds:SignatureValue>MEUCIQ...</ds:SignatureValue>
    </ds:Signature>
    <saml:Subject>
      <saml:NameID>operator@knirv.network</saml:NameID>
    </saml:Subject>
    <saml:AttributeStatement>
      <saml:Attribute Name="role">
        <saml:AttributeValue>user</saml:AttributeValue>
      </saml:Attribute>
    </saml:AttributeStatement>
  </saml:Assertion>
</samlp:Response>`;

const xswPayloads = [
  {
    id: 'xsw1',
    label: 'XSW #1 — cloned response, signed original moved',
    detail: 'Duplicates the Response, moves the signed copy outside, injects an unsigned forged copy as the one the SP parser reads first.',
  },
  {
    id: 'xsw2',
    label: 'XSW #2 — cloned response, signed as sibling',
    detail: 'Places the original signed Response as a sibling of the forged one — targets parsers that read the first element with a matching ID.',
  },
  {
    id: 'xsw3',
    label: 'XSW #3 — cloned assertion, original detached',
    detail: 'Detaches the signed Assertion from the Response entirely, then supplies a forged Assertion as a direct child.',
  },
  {
    id: 'certswap',
    label: 'Certificate swap',
    detail: 'Self-signs the assertion with an attacker-controlled certificate and swaps the trust anchor, testing certificate-pinning enforcement.',
  },
];

const SamlRaider: React.FC = () => {
  const [xml, setXml] = useState(sampleAssertion);
  const [nameId, setNameId] = useState('operator@knirv.network');
  const [role, setRole] = useState('user');
  const [selected, setSelected] = useState<string | null>(null);
  const [result, setResult] = useState<string | null>(null);

  const applyEdits = () => {
    setXml(prev =>
      prev
        .replace(/<saml:NameID>.*<\/saml:NameID>/, `<saml:NameID>${nameId}</saml:NameID>`)
        .replace(/<saml:AttributeValue>.*<\/saml:AttributeValue>/, `<saml:AttributeValue>${role}</saml:AttributeValue>`)
    );
  };

  const runAttack = (id: string) => {
    setSelected(id);
    const outcomes: Record<string, string> = {
      xsw1: 'SP parser read the forged (unsigned) Response — signature validation bypassed. Vulnerable to XSW1.',
      xsw2: 'SP rejected the sibling structure — schema validation caught the duplicate element. Not vulnerable to XSW2.',
      xsw3: 'SP parser resolved the assertion by ID lookup and accepted the forged assertion. Vulnerable to XSW3.',
      certswap: 'SP rejected the self-signed certificate — trust chain validated against a pinned CA. Not vulnerable.',
    };
    setResult(outcomes[id]);
  };

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-violet-500/20 rounded-lg">
          <FileSignature className="w-6 h-6 text-violet-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">SAML Raider</h1>
          <p className="text-slate-400 text-sm font-mono">intercepted SAMLResponse · POST binding · idp.knirv.network</p>
        </div>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
        <div className="xl:col-span-2 bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-slate-500 uppercase">Assertion editor</span>
            <div className="flex items-center space-x-2">
              <input
                value={nameId}
                onChange={e => setNameId(e.target.value)}
                className="w-56 bg-slate-900/60 border border-slate-700/50 rounded px-2 py-1 text-xs font-mono text-slate-200"
              />
              <select
                value={role}
                onChange={e => setRole(e.target.value)}
                className="bg-slate-900/60 border border-slate-700/50 rounded px-2 py-1 text-xs font-mono text-slate-200"
              >
                <option value="user">user</option>
                <option value="admin">admin</option>
                <option value="root">root</option>
              </select>
              <button onClick={applyEdits} className="text-xs px-2 py-1 rounded bg-violet-500/20 text-violet-300 hover:bg-violet-500/30">
                apply
              </button>
            </div>
          </div>
          <textarea
            value={xml}
            onChange={e => setXml(e.target.value)}
            spellCheck={false}
            className="w-full h-[380px] bg-slate-900/60 border border-slate-700/50 rounded px-3 py-2 text-xs font-mono text-slate-300 resize-none focus:outline-none"
          />
        </div>

        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="text-xs text-slate-500 uppercase mb-3">XML signature wrapping</div>
          <div className="space-y-2 mb-4">
            {xswPayloads.map(p => (
              <button
                key={p.id}
                onClick={() => runAttack(p.id)}
                className={`w-full text-left p-2.5 rounded-lg border transition-colors ${
                  selected === p.id ? 'bg-violet-500/15 border-violet-500/40' : 'bg-slate-900/40 border-slate-700/50 hover:border-slate-600'
                }`}
              >
                <div className="flex items-center space-x-2 mb-1">
                  <Play className="w-3.5 h-3.5 text-violet-400" />
                  <span className="text-xs font-mono text-violet-300">{p.label}</span>
                </div>
                <div className="text-[11px] text-slate-500">{p.detail}</div>
              </button>
            ))}
          </div>

          <div className="text-xs text-slate-500 uppercase mb-1">Result</div>
          <div className={`text-xs font-mono rounded p-2 min-h-[56px] ${
            result?.includes('Vulnerable') ? 'bg-red-500/10 text-red-300 border border-red-500/20' :
            result ? 'bg-green-500/10 text-green-300 border border-green-500/20' :
            'bg-slate-900/60 text-slate-700'
          }`}>
            {result ?? 'run an attack to see the SP response'}
          </div>
        </div>
      </div>
    </div>
  );
};

export default SamlRaider;
