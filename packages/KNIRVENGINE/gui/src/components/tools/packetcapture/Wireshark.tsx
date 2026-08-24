import React, { useState } from 'react';
import { Waves, Play, Square } from 'lucide-react';

interface Packet {
  no: number;
  time: string;
  src: string;
  dst: string;
  proto: string;
  len: number;
  info: string;
}

const packets: Packet[] = [
  { no: 1, time: '0.000000', src: '10.0.0.14', dst: '10.0.0.1', proto: 'TCP', len: 66, info: '52144 → 443 [SYN] Seq=0 Win=64240' },
  { no: 2, time: '0.021341', src: '10.0.0.1', dst: '10.0.0.14', proto: 'TCP', len: 66, info: '443 → 52144 [SYN, ACK] Seq=0 Ack=1' },
  { no: 3, time: '0.021519', src: '10.0.0.14', dst: '10.0.0.1', proto: 'TLSv1.3', len: 583, info: 'Client Hello (SNI=api.targetapp.local)' },
  { no: 4, time: '0.048220', src: '10.0.0.1', dst: '10.0.0.14', proto: 'TLSv1.3', len: 1514, info: 'Server Hello, Change Cipher Spec' },
  { no: 5, time: '0.049810', src: '10.0.0.14', dst: '10.0.0.1', proto: 'TLSv1.3', len: 128, info: 'Application Data' },
  { no: 6, time: '0.190220', src: '10.0.0.14', dst: '8.8.8.8', proto: 'DNS', len: 78, info: 'Standard query 0x91ab A telemetry.targetapp.local' },
  { no: 7, time: '0.201881', src: '8.8.8.8', dst: '10.0.0.14', proto: 'DNS', len: 94, info: 'Standard query response A 34.120.9.211' },
  { no: 8, time: '0.412009', src: '10.0.0.14', dst: '10.0.0.1', proto: 'HTTP', len: 341, info: 'GET /oracle/checkpoint HTTP/1.1' },
];

const hexDump = `0000  4a 00 05 4d 40 00 40 06 00 00 0a 00 00 0e 0a 00   J..M@.@.........
0010  00 01 cb b0 01 bb b2 3c 91 4e 00 00 00 00 80 02   .......<.N......
0020  fa f0 f6 9a 00 00 02 04 05 b4 04 02 08 0a 3b 9a   ..............;.
0030  ce 12 00 00 00 00 01 03 03 07                     ..........`;

const protoColor: Record<string, string> = {
  TCP: 'text-slate-400',
  'TLSv1.3': 'text-purple-300',
  DNS: 'text-blue-300',
  HTTP: 'text-green-300',
};

const Wireshark: React.FC = () => {
  const [filter, setFilter] = useState('tls.handshake.type == 1 || dns || http');
  const [capturing, setCapturing] = useState(true);
  const [selectedNo, setSelectedNo] = useState(3);

  const selected = packets.find(p => p.no === selectedNo);

  const filtered = packets.filter(p => {
    const q = filter.trim().toLowerCase();
    if (!q) return true;
    if (q.includes('||')) {
      return q.split('||').some(clause => matchClause(p, clause.trim()));
    }
    return matchClause(p, q);
  });

  function matchClause(p: Packet, clause: string) {
    if (clause === 'dns') return p.proto === 'DNS';
    if (clause === 'http') return p.proto === 'HTTP';
    if (clause.startsWith('tls')) return p.proto.startsWith('TLS');
    return `${p.src} ${p.dst} ${p.info}`.toLowerCase().includes(clause);
  }

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-3">
          <div className="p-2 bg-sky-500/20 rounded-lg">
            <Waves className="w-6 h-6 text-sky-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">Wireshark</h1>
            <p className="text-slate-400 text-sm font-mono">tshark -i eth0 -w capture.pcapng</p>
          </div>
        </div>
        <button
          onClick={() => setCapturing(v => !v)}
          className={`flex items-center space-x-2 px-3 py-2 rounded-lg text-sm font-medium border ${
            capturing ? 'bg-sky-500/20 border-sky-500/40 text-sky-300' : 'bg-slate-800/50 border-slate-700/50 text-slate-300'
          }`}
        >
          {capturing ? <Square className="w-4 h-4" /> : <Play className="w-4 h-4" />}
          <span>{capturing ? 'Stop capture' : 'Start capture'}</span>
        </button>
      </div>

      <div className="flex items-center space-x-2 mb-4">
        <span className="font-mono text-slate-500 text-sm">filter:</span>
        <input
          value={filter}
          onChange={e => setFilter(e.target.value)}
          className="flex-1 bg-slate-800/50 border border-slate-700/50 rounded-lg px-3 py-2 text-sm font-mono text-slate-200 focus:outline-none focus:border-sky-500/50"
        />
      </div>

      <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg overflow-hidden mb-4">
        <table className="w-full text-xs font-mono">
          <thead>
            <tr className="border-b border-slate-700/50 text-slate-500 uppercase">
              <th className="text-right px-3 py-2">No.</th>
              <th className="text-left px-3 py-2">Time</th>
              <th className="text-left px-3 py-2">Source</th>
              <th className="text-left px-3 py-2">Destination</th>
              <th className="text-left px-3 py-2">Protocol</th>
              <th className="text-right px-3 py-2">Length</th>
              <th className="text-left px-3 py-2">Info</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map(p => (
              <tr
                key={p.no}
                onClick={() => setSelectedNo(p.no)}
                className={`border-b border-slate-800 cursor-pointer ${selectedNo === p.no ? 'bg-sky-500/10' : 'hover:bg-slate-700/30'}`}
              >
                <td className="px-3 py-1.5 text-right text-slate-600">{p.no}</td>
                <td className="px-3 py-1.5 text-slate-500">{p.time}</td>
                <td className="px-3 py-1.5 text-slate-300">{p.src}</td>
                <td className="px-3 py-1.5 text-slate-300">{p.dst}</td>
                <td className={`px-3 py-1.5 font-semibold ${protoColor[p.proto] ?? 'text-slate-400'}`}>{p.proto}</td>
                <td className="px-3 py-1.5 text-right text-slate-500">{p.len}</td>
                <td className="px-3 py-1.5 text-slate-400 truncate max-w-[320px]">{p.info}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {selected && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-3">
            <div className="text-xs text-slate-600 uppercase mb-2">Frame {selected.no} — protocol tree</div>
            <div className="font-mono text-xs text-slate-400 leading-relaxed">
              <div>▸ Frame {selected.no}: {selected.len} bytes on wire</div>
              <div>▸ Ethernet II, Src: 02:42:ac:11:00:02, Dst: 02:42:ac:11:00:01</div>
              <div>▸ Internet Protocol, Src: {selected.src}, Dst: {selected.dst}</div>
              <div>▸ Transmission Control Protocol{selected.proto.includes('TLS') ? '' : ', Src Port: 52144, Dst Port: 443'}</div>
              {selected.proto.includes('TLS') && <div className="text-purple-300">▾ Transport Layer Security</div>}
              {selected.proto === 'DNS' && <div className="text-blue-300">▾ Domain Name System</div>}
              {selected.proto === 'HTTP' && <div className="text-green-300">▾ Hypertext Transfer Protocol</div>}
            </div>
          </div>
          <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-3">
            <div className="text-xs text-slate-600 uppercase mb-2">Hex dump</div>
            <pre className="font-mono text-[11px] text-slate-400 whitespace-pre overflow-x-auto">{hexDump}</pre>
          </div>
        </div>
      )}
    </div>
  );
};

export default Wireshark;
