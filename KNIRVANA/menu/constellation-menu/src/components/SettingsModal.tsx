import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Slider } from "@/components/ui/slider";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useSettings, useUpdateSettings, useResetSettings } from "@/hooks/use-settings";
import { motion } from "framer-motion";
import { Loader2, RefreshCw } from "lucide-react";

interface SettingsModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SettingsModal({ open, onOpenChange }: SettingsModalProps) {
  const { data: settings, isLoading } = useSettings();
  const updateSettings = useUpdateSettings();
  const resetSettings = useResetSettings();

  if (isLoading) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="bg-black/80 border-[#00f3ff]/30 backdrop-blur-xl text-[#00f3ff] max-w-md font-['Rajdhani'] box-glow">
        <DialogHeader>
          <DialogTitle className="text-2xl font-bold font-['Orbitron'] text-center tracking-widest text-glow uppercase">
            System Configuration
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-8 py-4">
          {/* Audio Section */}
          <div className="space-y-4">
            <h3 className="text-sm font-semibold text-white/70 uppercase tracking-widest border-b border-white/10 pb-2">Audio Systems</h3>
            <div className="space-y-4">
              <div className="space-y-2">
                <div className="flex justify-between">
                  <Label>Music Volume</Label>
                  <span className="text-xs font-mono">{settings?.musicVolume}%</span>
                </div>
                <Slider
                  value={[settings?.musicVolume || 80]}
                  max={100}
                  step={1}
                  className="cursor-pointer"
                  onValueChange={(val) => updateSettings.mutate({ musicVolume: val[0] })}
                />
              </div>
              <div className="space-y-2">
                <div className="flex justify-between">
                  <Label>SFX Volume</Label>
                  <span className="text-xs font-mono">{settings?.sfxVolume}%</span>
                </div>
                <Slider
                  value={[settings?.sfxVolume || 80]}
                  max={100}
                  step={1}
                  className="cursor-pointer"
                  onValueChange={(val) => updateSettings.mutate({ sfxVolume: val[0] })}
                />
              </div>
            </div>
          </div>

          {/* Graphics Section */}
          <div className="space-y-4">
            <h3 className="text-sm font-semibold text-white/70 uppercase tracking-widest border-b border-white/10 pb-2">Visual Processing</h3>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Graphics Quality</Label>
                <Select
                  value={settings?.graphicsQuality}
                  onValueChange={(val) => updateSettings.mutate({ graphicsQuality: val as any })}
                >
                  <SelectTrigger className="bg-black/50 border-[#00f3ff]/30 text-[#00f3ff] focus:ring-[#00f3ff]/50">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent className="bg-[#020617] border-[#00f3ff]/30 text-[#00f3ff]">
                    <SelectItem value="low">Low Latency</SelectItem>
                    <SelectItem value="medium">Balanced</SelectItem>
                    <SelectItem value="high">High Fidelity</SelectItem>
                    <SelectItem value="ultra">Ultra Realistic</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              
              <div className="space-y-2">
                <Label>Language Data</Label>
                <Select
                  value={settings?.language}
                  onValueChange={(val) => updateSettings.mutate({ language: val })}
                >
                  <SelectTrigger className="bg-black/50 border-[#00f3ff]/30 text-[#00f3ff] focus:ring-[#00f3ff]/50">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent className="bg-[#020617] border-[#00f3ff]/30 text-[#00f3ff]">
                    <SelectItem value="en">English (US)</SelectItem>
                    <SelectItem value="jp">Japanese</SelectItem>
                    <SelectItem value="de">German</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>

          <div className="pt-4 flex justify-end">
            <button
              onClick={() => resetSettings.mutate()}
              disabled={resetSettings.isPending}
              className="flex items-center gap-2 px-4 py-2 text-xs uppercase tracking-widest text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded transition-colors disabled:opacity-50"
            >
              {resetSettings.isPending ? <Loader2 className="w-3 h-3 animate-spin" /> : <RefreshCw className="w-3 h-3" />}
              Reset to Factory
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
