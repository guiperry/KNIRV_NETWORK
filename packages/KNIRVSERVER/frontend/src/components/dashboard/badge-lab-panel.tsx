'use client';

import React, { useState, useRef, useEffect } from 'react';
import { 
  Upload, Wand2, Download, Loader2, RefreshCcw, 
  Settings2, Info, Award 
} from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';

interface BadgeLabPanelProps {
  className?: string;
}

const values = [
  "Guidelines",
  "Customs",
  "Etiquette",
  "Mission Statement",
  "Stated Values",
  "Goals & Objectives",
  "Insights"
];

const ontology = [
  "Trade Secrets",
  "Business Logic",
  "User Data",
  "Rules",
  "Regulations",
  "Procedures",
  "Policies",
  "FAQs",
  "Customer Service Bullets"
];

export const BadgeLabPanel: React.FC<BadgeLabPanelProps> = ({ className }) => {
  const [prompt, setPrompt] = useState('');
  const [isProcessing, setIsProcessing] = useState(false);
  const [outputUrl, setOutputUrl] = useState<string | null>(null);
  const [selectedValues, setSelectedValues] = useState<string[]>([]);
  const [selectedOntology, setSelectedOntology] = useState<string[]>([]);
  
  const fileInputRef = useRef<HTMLInputElement>(null);

  const toggleValue = (value: string) => {
    setSelectedValues(prev => 
      prev.includes(value) ? prev.filter(v => v !== value) : [...prev, value]
    );
  };

  const toggleOntology = (item: string) => {
    setSelectedOntology(prev => 
      prev.includes(item) ? prev.filter(v => v !== item) : [...prev, item]
    );
  };

  const handleGenerateBadge = async () => {
    if (!prompt) return;

    setIsProcessing(true);
    setOutputUrl(null);

    await new Promise(resolve => setTimeout(resolve, 3000));

    const svgContent = `
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 400 400" width="400" height="400">
        <defs>
          <linearGradient id="goldGrad" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" style="stop-color:#FBBF24"/>
            <stop offset="50%" style="stop-color:#D97706"/>
            <stop offset="100%" style="stop-color:#B45309"/>
          </linearGradient>
          <linearGradient id="darkGrad" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" style="stop-color:#1F2937"/>
            <stop offset="100%" style="stop-color:#111827"/>
          </linearGradient>
        </defs>
        <rect width="400" height="400" fill="#02040a"/>
        <circle cx="200" cy="200" r="180" fill="url(#darkGrad)" stroke="url(#goldGrad)" stroke-width="8"/>
        <circle cx="200" cy="200" r="150" fill="none" stroke="url(#goldGrad)" stroke-width="2" opacity="0.5"/>
        <polygon points="200,80 230,140 295,140 245,180 265,245 200,205 135,245 155,180 105,140 170,140" fill="url(#goldGrad)"/>
        <text x="200" y="280" text-anchor="middle" fill="#FBBF24" font-family="Arial Black" font-size="24" font-weight="bold">${prompt.substring(0, 20)}</text>
        <text x="200" y="310" text-anchor="middle" fill="#9CA3AF" font-family="Arial" font-size="12">KNIRV VERIFIED</text>
      </svg>
    `;
    
    const blob = new Blob([svgContent], { type: 'image/svg+xml' });
    const url = URL.createObjectURL(blob);
    setOutputUrl(url);
    setIsProcessing(false);
  };

  return (
    <div className={`space-y-4 ${className}`}>
      <Card className="knirv-card-gradient border-amber-500/30">
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <div className="p-2 bg-amber-500/10 text-amber-400 border border-amber-500/20 rounded-lg">
                <Award size={18} />
              </div>
              <span className="text-lg font-bold">Badge Lab</span>
            </div>
            <Badge variant="outline" className="text-amber-400 border-amber-400/30">
              NFT Badge Designer
            </Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
            {/* Left: Controls */}
            <div className="lg:col-span-4 space-y-4">
              <div className="rounded-xl border border-gray-800 bg-gray-950/40 p-4 space-y-4">
                <div>
                  <label className="text-[10px] font-black text-gray-500 uppercase tracking-widest block mb-3">
                    Badge Purpose
                  </label>
                  <textarea 
                    value={prompt}
                    onChange={(e) => setPrompt(e.target.value)}
                    placeholder="Describe what this badge represents... (e.g., 'Certified Blockchain Professional', 'Excellent Customer Service Award')"
                    className="w-full bg-gray-900 border border-gray-800 rounded-lg p-3 text-xs focus:outline-none focus:border-amber-500/50 min-h-[100px] resize-none font-medium text-gray-200"
                  />
                </div>

                <div>
                  <label className="text-[10px] font-black text-gray-500 uppercase tracking-widest block mb-3">
                    Values to Emphasize <span className="text-amber-400">(Select 3-5)</span>
                  </label>
                  <div className="grid grid-cols-2 gap-1.5">
                    {values.map((value) => (
                      <button
                        key={value}
                        onClick={() => toggleValue(value)}
                        className={`px-2 py-1.5 rounded-md text-[9px] font-bold transition-all border ${
                          selectedValues.includes(value)
                            ? 'bg-amber-500/20 border-amber-500/40 text-amber-400'
                            : 'bg-gray-900 border-gray-800 text-gray-500 hover:bg-gray-800 hover:text-gray-300'
                        }`}
                      >
                        {value}
                      </button>
                    ))}
                  </div>
                </div>

                <div>
                  <label className="text-[10px] font-black text-gray-500 uppercase tracking-widest block mb-3">
                    Ontology Elements <span className="text-amber-400">(Select relevant)</span>
                  </label>
                  <div className="grid grid-cols-2 gap-1.5">
                    {ontology.map((item) => (
                      <button
                        key={item}
                        onClick={() => toggleOntology(item)}
                        className={`px-2 py-1.5 rounded-md text-[9px] font-bold transition-all border ${
                          selectedOntology.includes(item)
                            ? 'bg-amber-500/20 border-amber-500/40 text-amber-400'
                            : 'bg-gray-900 border-gray-800 text-gray-500 hover:bg-gray-800 hover:text-gray-300'
                        }`}
                      >
                        {item}
                      </button>
                    ))}
                  </div>
                </div>

                <button 
                  onClick={handleGenerateBadge}
                  disabled={isProcessing || !prompt}
                  className={`w-full py-3 rounded-lg font-black uppercase tracking-widest flex items-center justify-center gap-2 text-xs transition-all ${
                    isProcessing 
                      ? 'bg-gray-800 text-gray-500 cursor-wait' 
                      : 'bg-amber-600 hover:bg-amber-500 text-white'
                  }`}
                >
                  {isProcessing ? (
                    <Loader2 className="animate-spin" size={16} />
                  ) : (
                    <Wand2 size={16} />
                  )}
                  {isProcessing ? 'Designing Badge...' : 'Generate Badge'}
                </button>

                <div className="flex items-start gap-2 text-[9px] text-gray-600 font-medium bg-gray-900/40 p-2 rounded-lg border border-gray-800/50">
                  <Info size={12} className="shrink-0 text-gray-500 mt-0.5" />
                  <p>
                    Badge generation creates professional designs suitable for NFT minting. 
                    Selected values and ontology elements will be visually incorporated.
                  </p>
                </div>
              </div>
            </div>

            {/* Right: Stage */}
            <div className="lg:col-span-8 rounded-xl border border-gray-800 bg-gray-950/40 overflow-hidden flex flex-col">
              <div className="px-4 py-3 border-b border-gray-800 flex items-center gap-2">
                <div className={`w-2 h-2 rounded-full ${
                  isProcessing ? 'bg-amber-500 animate-ping' : outputUrl ? 'bg-emerald-500' : 'bg-gray-600'
                }`} />
                <span className="text-[10px] font-black font-mono text-gray-400 tracking-[0.2em] uppercase">
                  Badge_Design
                </span>
              </div>

              <div className="flex-1 flex items-center justify-center p-8 bg-[#02040a]/20 min-h-[300px]">
                {isProcessing ? (
                  <div className="flex flex-col items-center gap-4 text-amber-400/50">
                    <div className="relative">
                      <div className="absolute inset-0 bg-amber-500/10 blur-3xl rounded-full" />
                      <Award size={60} className="relative" />
                    </div>
                    <div className="flex flex-col items-center">
                      <span className="text-xs font-black uppercase tracking-[0.4em]">Designing_Badge</span>
                      <div className="flex gap-1 mt-2">
                        <div className="w-16 h-1 bg-amber-500/20 rounded-full overflow-hidden">
                          <div className="h-full bg-amber-500 animate-pulse" style={{width: '60%'}} />
                        </div>
                      </div>
                    </div>
                  </div>
                ) : outputUrl ? (
                  <div className="relative group max-w-full max-h-full flex items-center justify-center">
                    <img 
                      src={outputUrl} 
                      className="max-w-full max-h-[280px] object-contain rounded-xl shadow-[0_0_60px_rgba(0,0,0,0.5)]" 
                      alt="Badge Design" 
                    />
                  </div>
                ) : (
                  <div className="flex flex-col items-center gap-3 text-gray-800 text-center max-w-xs">
                    <div className="p-6 rounded-full bg-gray-900/40 border border-gray-800/50">
                      <Settings2 size={36} className="opacity-10" />
                    </div>
                    <div className="space-y-1">
                      <p className="text-[10px] font-black uppercase tracking-[0.2em] text-gray-700">
                        Badge Design Studio
                      </p>
                      <p className="text-[9px] font-medium text-gray-600 leading-relaxed italic">
                        Configure your badge specifications and click "Generate Badge" to create a professional 
                        NFT badge design.
                      </p>
                    </div>
                  </div>
                )}
              </div>

              {outputUrl && (
                <div className="p-4 bg-gray-950/60 border-t border-gray-800/50 flex justify-between items-center">
                  <div className="flex items-center gap-3">
                    <div className="flex flex-col">
                      <span className="text-[8px] font-black text-gray-600 uppercase tracking-widest">
                        Badge Elements
                      </span>
                      <span className="text-[10px] font-bold text-amber-400">
                        {selectedValues.length} Values • {selectedOntology.length} Ontology Items
                      </span>
                    </div>
                    <div className="w-px h-6 bg-gray-800" />
                    <div className="flex flex-col">
                      <span className="text-[8px] font-black text-gray-600 uppercase tracking-widest">
                        Dimensions
                      </span>
                      <span className="text-[10px] font-bold text-gray-400">400x400 (NFT Ready)</span>
                    </div>
                  </div>
                  
                  <div className="flex gap-2">
                    <button 
                      onClick={() => { setOutputUrl(null); setPrompt(''); setSelectedValues([]); setSelectedOntology([]); }}
                      className="flex items-center gap-1.5 px-4 py-2 bg-gray-900 hover:bg-gray-800 text-gray-400 rounded-lg text-[10px] font-black uppercase transition-all border border-gray-800"
                    >
                      <RefreshCcw size={12} /> Reset
                    </button>
                    <a 
                      href={outputUrl} 
                      download="badge-design.svg"
                      className="flex items-center gap-1.5 px-5 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-[10px] font-black uppercase transition-all"
                    >
                      <Download size={12} /> Download Badge
                    </a>
                  </div>
                </div>
              )}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default BadgeLabPanel;
