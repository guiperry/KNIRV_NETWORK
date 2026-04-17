"use client"

import { useEffect, useState } from "react"
import { Inter } from 'next/font/google'

const inter = Inter({ subsets: ['latin'] })
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card"
import { Button } from "./ui/button"
import { Switch } from "./ui/switch"
import { Dialog, DialogContent, DialogHeader, DialogTitle as DialogTitlePrimitive } from "./ui/dialog"
import { Volume2, Palette, Globe, Zap, Shield } from "lucide-react"

interface SettingsModalProps {
  isOpen: boolean
  onClose: () => void
}

interface SettingsState {
  soundEnabled: boolean
  animationsEnabled: boolean
  darkMode: boolean
  language: string
  performanceMode: boolean
  securityMode: boolean
}

const defaultSettings: SettingsState = {
  soundEnabled: true,
  animationsEnabled: true,
  darkMode: true,
  language: "en",
  performanceMode: false,
  securityMode: true,
}

const SettingsModal = ({ isOpen, onClose }: SettingsModalProps) => {
  const [settings, setSettings] = useState<SettingsState>(defaultSettings)

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }

    if (isOpen) {
      document.addEventListener('keydown', handleEscape)
    }

    return () => {
      document.removeEventListener('keydown', handleEscape)
    }
  }, [isOpen, onClose])

  const handleSettingChange = (key: keyof SettingsState, value: boolean | string) => {
    setSettings(prev => ({
      ...prev,
      [key]: value
    }))
  }

  const resetSettings = () => {
    setSettings(defaultSettings)
  }

  const saveSettings = () => {
    // Here you would typically save to localStorage or backend
    console.log('Saving settings:', settings)
    onClose()
  }

  if (!isOpen) return null

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className={`w-full max-w-2xl mx-4 bg-gray-900/95 border-cyan-500/30 text-white ${inter.className}`}>
        <DialogHeader>
          <DialogTitlePrimitive className={`text-cyan-400 text-xl font-bold flex items-center gap-2 ${inter.className}`}>
            <Shield className="h-5 w-5" />
            Settings
          </DialogTitlePrimitive>
        </DialogHeader>
        
        <CardContent className="space-y-6">
          {/* Audio Settings */}
          <Card className="bg-gray-800/50 border-cyan-500/20">
            <CardHeader className="pb-3">
              <CardTitle className="text-cyan-300 text-base flex items-center gap-2">
                <Volume2 className="h-4 w-4" />
                Audio
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex items-center justify-between">
                <label className={`text-sm text-gray-300 ${inter.className}`}>Sound Effects</label>
                <Switch
                  checked={settings.soundEnabled}
                  onCheckedChange={(checked) => handleSettingChange('soundEnabled', checked)}
                />
              </div>
            </CardContent>
          </Card>

          {/* Visual Settings */}
          <Card className="bg-gray-800/50 border-cyan-500/20">
            <CardHeader className="pb-3">
              <CardTitle className="text-cyan-300 text-base flex items-center gap-2">
                <Palette className="h-4 w-4" />
                Visual
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex items-center justify-between">
                <label className={`text-sm text-gray-300 ${inter.className}`}>Animations</label>
                <Switch
                  checked={settings.animationsEnabled}
                  onCheckedChange={(checked) => handleSettingChange('animationsEnabled', checked)}
                />
              </div>
              <div className="flex items-center justify-between">
                <label className={`text-sm text-gray-300 ${inter.className}`}>Dark Mode</label>
                <Switch
                  checked={settings.darkMode}
                  onCheckedChange={(checked) => handleSettingChange('darkMode', checked)}
                />
              </div>
            </CardContent>
          </Card>

          {/* Performance Settings */}
          <Card className="bg-gray-800/50 border-cyan-500/20">
            <CardHeader className="pb-3">
              <CardTitle className="text-cyan-300 text-base flex items-center gap-2">
                <Zap className="h-4 w-4" />
                Performance
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex items-center justify-between">
                <label className={`text-sm text-gray-300 ${inter.className}`}>Performance Mode</label>
                <Switch
                  checked={settings.performanceMode}
                  onCheckedChange={(checked) => handleSettingChange('performanceMode', checked)}
                />
              </div>
            </CardContent>
          </Card>

          {/* Security Settings */}
          <Card className="bg-gray-800/50 border-cyan-500/20">
            <CardHeader className="pb-3">
              <CardTitle className="text-cyan-300 text-base flex items-center gap-2">
                <Shield className="h-4 w-4" />
                Security
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex items-center justify-between">
                <label className={`text-sm text-gray-300 ${inter.className}`}>Enhanced Security</label>
                <Switch
                  checked={settings.securityMode}
                  onCheckedChange={(checked) => handleSettingChange('securityMode', checked)}
                />
              </div>
            </CardContent>
          </Card>

          {/* Action Buttons */}
          <div className="flex justify-between pt-4 border-t border-gray-700">
            <Button
              variant="outline"
              onClick={resetSettings}
              className="border-cyan-500/30 text-cyan-400 hover:bg-cyan-500/10"
            >
              Reset to Default
            </Button>
            <div className="flex space-x-2">
              <Button
                variant="outline"
                onClick={onClose}
                className="border-cyan-500/30 text-cyan-400 hover:bg-cyan-500/10"
              >
                Cancel
              </Button>
              <Button
                onClick={saveSettings}
                className="bg-cyan-500 hover:bg-cyan-600 text-black font-semibold"
              >
                Save Settings
              </Button>
            </div>
          </div>
        </CardContent>
      </DialogContent>
    </Dialog>
  )
}

export default SettingsModal