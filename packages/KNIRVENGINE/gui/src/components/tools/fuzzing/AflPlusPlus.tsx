import React from 'react';
import { ToolStreamConsole } from '../ToolConsoles';

const AflPlusPlus: React.FC = () => <><div className="mx-6 mt-6 rounded border border-amber-500/30 bg-amber-500/10 p-3 text-sm text-amber-100">AFL++ uses documented sandbox-safe fallbacks for host core-dump routing and CPU frequency scaling. Campaigns run normally, but a host crash reporter can classify a crash as a timeout and CPU scaling can reduce fuzzing throughput.</div><ToolStreamConsole title="AFL++" tool="afl-fuzz" inputLabel="Target path" defaultValue="" inputKey="targetPath" /></>;
export default AflPlusPlus;
