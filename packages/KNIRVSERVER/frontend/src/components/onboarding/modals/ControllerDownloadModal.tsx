'use client';

import React from 'react';
import { QRCodeSVG } from 'qrcode.react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Download, Smartphone } from "lucide-react";

type DownloadPlatform = 'android' | 'ios';

interface ControllerDownloadModalProps {
  isOpen: boolean;
  onClose: () => void;
  platform: DownloadPlatform;
  downloadUrl: string;
}

const platformMeta: Record<DownloadPlatform, {
  title: string;
  description: string;
  packageLabel: string;
  endpointNote: string;
}> = {
  android: {
    title: 'Android Download QR',
    description: 'Scan with any general QR scanner on your phone to open the Cloudflare download endpoint for the Android package.',
    packageLabel: 'APK',
    endpointNote: 'Point this URL at your Android Cloudflare endpoint when you are ready.',
  },
  ios: {
    title: 'iOS Download QR',
    description: 'Scan with any general QR scanner on your phone to open the Cloudflare download endpoint for the iOS package.',
    packageLabel: 'IPA',
    endpointNote: 'Point this URL at your iOS Cloudflare endpoint when you are ready.',
  },
};

export function ControllerDownloadModal({
  isOpen,
  onClose,
  platform,
  downloadUrl,
}: ControllerDownloadModalProps) {
  const meta = platformMeta[platform];

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="w-[96vw] max-w-2xl max-h-[92vh] overflow-hidden bg-[#0a0a0c] border-white/10 text-slate-200 flex flex-col">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-600/20 rounded-lg">
              <Download className="text-blue-500" size={24} />
            </div>
            <div>
              <DialogTitle className="text-xl font-bold text-white">{meta.title}</DialogTitle>
              <DialogDescription className="text-slate-400 text-sm">
                {meta.description}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto py-4 pr-1 custom-scrollbar space-y-5">
          <div className="flex items-center justify-between">
            <Badge variant="outline" className="border-blue-500/30 text-blue-400">
              {meta.packageLabel}
            </Badge>
            <Badge variant="outline" className="border-white/10 text-slate-400">
              Cloudflare endpoint
            </Badge>
          </div>

          <div className="flex flex-col items-center gap-4 rounded-2xl border border-white/10 bg-white/5 p-6">
            <div className="rounded-2xl bg-white p-4 shadow-[0_0_30px_rgba(59,130,246,0.2)]">
              <QRCodeSVG value={downloadUrl} size={224} bgColor="#ffffff" fgColor="#111827" includeMargin />
            </div>
            <div className="text-center">
              <div className="text-sm font-semibold text-slate-200">{meta.packageLabel} download QR</div>
              <p className="mt-1 text-xs text-slate-400">
                Use your phone&apos;s default QR scanner to open the download link.
              </p>
            </div>
          </div>

          <div className="rounded-xl border border-white/10 bg-black/30 p-4">
            <div className="flex items-center gap-2 text-sm text-slate-300">
              <Smartphone size={16} className="text-blue-500" />
              <span>Download URL</span>
            </div>
            <div className="mt-2 break-all font-mono text-xs text-slate-500">
              {downloadUrl}
            </div>
            <p className="mt-2 text-xs text-slate-500">{meta.endpointNote}</p>
          </div>

          <div className="flex justify-end">
            <Button onClick={onClose} variant="ghost">
              Close
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

export default ControllerDownloadModal;
