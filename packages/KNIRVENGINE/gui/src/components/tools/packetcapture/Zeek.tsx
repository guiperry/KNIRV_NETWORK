import React, { useState } from 'react';
import { Fish } from 'lucide-react';

const logs = {
  'conn.log': {
    headers: ['ts', 'id.orig_h', 'id.orig_p', 'id.resp_h', 'id.resp_p', 'proto', 'service', 'duration', 'conn_state'],
    rows: [
      ['1724457601.203', '10.0.0.14', '52144', '10.0.0.1', '443', 'tcp', 'ssl', '4.812', 'SF'],
      ['1724457602.881', '10.0.0.14', '52148', '34.120.9.211', '443', 'tcp', 'http', '0.221', 'S0'],
      ['1724457605.014', '10.0.0.14', '38221', '8.8.8.8', '53', 'udp', 'dns', '0.011', 'SF'],
    ],
  },
  'dns.log': {
    headers: ['ts', 'id.orig_h', 'query', 'qtype_name', 'rcode_name', 'answers'],
    rows: [
      ['1724457605.014', '10.0.0.14', 'telemetry.targetapp.local', 'A', 'NOERROR', '34.120.9.211'],
      ['1724457611.402', '10.0.0.14', 'api.knirv.network', 'A', 'NOERROR', '10.0.0.1'],
      ['1724457618.902', '10.0.0.14', 'c2-relay.suspicious-tld.xyz', 'A', 'NXDOMAIN', '-'],
    ],
  },
  'ssl.log': {
    headers: ['ts', 'id.resp_h', 'version', 'cipher', 'server_name', 'validation_status'],
    rows: [
      ['1724457601.203', '10.0.0.1', 'TLSv1.3', 'TLS_AES_128_GCM_SHA256', 'api.targetapp.local', 'ok'],
      ['1724457602.881', '34.120.9.211', 'TLSv1.2', 'ECDHE-RSA-AES128-GCM-SHA256', 'telemetry.targetapp.local', 'self signed certificate'],
    ],
  },
  'notice.log': {
    headers: ['ts', 'id.orig_h', 'note', 'msg', 'sub'],
    rows: [
      ['1724457618.902', '10.0.0.14', 'DNS::Detect_ToRe_Resolution', 'query to known-bad TLD suspicious-tld.xyz', 'DGA-like pattern'],
      ['1724457602.881', '34.120.9.211', 'SSL::Invalid_Server_Cert', 'self-signed certificate presented for telemetry.targetapp.local', ''],
    ],
  },
} as const;

type LogName = keyof typeof logs;

const Zeek: React.FC = () => {
  const [active, setActive] = useState<LogName>('conn.log');
  const current = logs[active];

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-teal-500/20 rounded-lg">
          <Fish className="w-6 h-6 text-teal-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Zeek</h1>
          <p className="text-slate-400 text-sm font-mono">zeek -i eth0 local · logs/current/</p>
        </div>
      </div>

      <div className="flex space-x-1 mb-4">
        {(Object.keys(logs) as LogName[]).map(name => (
          <button
            key={name}
            onClick={() => setActive(name)}
            className={`px-3 py-1.5 rounded-lg text-xs font-mono border ${
              active === name
                ? name === 'notice.log'
                  ? 'bg-red-500/20 border-red-500/40 text-red-300'
                  : 'bg-teal-500/20 border-teal-500/40 text-teal-300'
                : 'border-slate-700/50 text-slate-500 hover:text-white'
            }`}
          >
            {name}
          </button>
        ))}
      </div>

      <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg overflow-x-auto">
        <table className="w-full text-xs font-mono">
          <thead>
            <tr className="border-b border-slate-700/50 text-slate-500">
              {current.headers.map(h => (
                <th key={h} className="text-left px-3 py-2 whitespace-nowrap">#{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {current.rows.map((row, i) => (
              <tr key={i} className={`border-b border-slate-800 ${active === 'notice.log' ? 'bg-red-500/5' : ''}`}>
                {row.map((cell, j) => (
                  <td key={j} className={`px-3 py-1.5 whitespace-nowrap ${j === 0 ? 'text-slate-600' : 'text-slate-300'}`}>
                    {cell}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {active === 'notice.log' && (
        <div className="mt-3 text-xs text-red-300 bg-red-500/10 border border-red-500/20 rounded px-3 py-2">
          2 active notices — DNS query to a known-bad TLD and a self-signed certificate on the telemetry endpoint.
        </div>
      )}
    </div>
  );
};

export default Zeek;
