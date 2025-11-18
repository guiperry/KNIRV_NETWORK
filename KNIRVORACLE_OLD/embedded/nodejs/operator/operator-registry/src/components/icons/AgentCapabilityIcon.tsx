// src/components/icons/AgentCapabilityIcon.tsx
import { Cog, LucideProps, BrainCircuit, ShieldCheck, MessageSquareText, DatabaseZap, Palette, Code2, FileText, LucideIcon } from 'lucide-react';

interface AgentCapabilityIconProps extends LucideProps {
  capability?: string;
}

const capabilityIconMap: Record<string, LucideIcon> = {
  'Document Translation': FileText,
  'Language Identification': MessageSquareText,
  'Text Summarization': MessageSquareText,
  'Data Anonymization': ShieldCheck,
  'PII Detection': ShieldCheck,
  'Privacy Policy Enforcement': ShieldCheck,
  'Image Generation': Palette,
  'Style Transfer': Palette,
  'Image Upscaling': Palette,
  'Code Generation': Code2,
  'Debugging Assistance': Code2,
  'API Documentation Lookup': Code2,
  'Generic AI': BrainCircuit,
  'Security': ShieldCheck,
  'Database': DatabaseZap,
};

const AgentCapabilityIcon = ({ capability, className, ...props }: AgentCapabilityIconProps) => {
  const IconComponent = capability && capabilityIconMap[capability] ? capabilityIconMap[capability] : Cog;
  return <IconComponent className={className} {...props} />;
};

export default AgentCapabilityIcon;
