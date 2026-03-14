'use client';

import LoadingScreen from "@/components/LoadingScreen";

export default function Loading() {
  return (
    <LoadingScreen
      isVisible={true}
      message="Loading NIM Cortex Builder..."
      progress={null}
    />
  );
}