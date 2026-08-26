import React from 'react';
import { ToolStreamConsole } from '../ToolConsoles';

const AflPlusPlus: React.FC = () => <ToolStreamConsole title="AFL++" tool="afl-fuzz" inputLabel="Target path" defaultValue="" inputKey="targetPath" />;
export default AflPlusPlus;
