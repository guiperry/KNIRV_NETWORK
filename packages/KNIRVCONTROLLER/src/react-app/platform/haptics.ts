import { Capacitor } from '@capacitor/core';
import { Haptics, ImpactStyle, NotificationType } from '@capacitor/haptics';

export async function impactMedium() {
  if (!Capacitor.isNativePlatform()) return;
  await Haptics.impact({ style: ImpactStyle.Medium });
}

export async function impactHeavy() {
  if (!Capacitor.isNativePlatform()) return;
  await Haptics.impact({ style: ImpactStyle.Heavy });
}

export async function notificationError() {
  if (!Capacitor.isNativePlatform()) return;
  await Haptics.notification({ type: NotificationType.Error });
}
