import React from 'react';
interface CognitiveShellInterfaceProps {
  onStateChange?: (state: unknown) => void;
}
export const CognitiveShellInterface: React.FC<CognitiveShellInterfaceProps> = ({ onStateChange }) => {
  return <div>Test</div>;
};
