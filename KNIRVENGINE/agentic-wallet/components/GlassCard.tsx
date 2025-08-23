import React from 'react';
import { View, StyleSheet, ViewStyle } from 'react-native';
import { LinearGradient } from 'expo-linear-gradient';

interface GlassCardProps {
  children: React.ReactNode;
  style?: ViewStyle;
  variant?: 'primary' | 'secondary' | 'accent';
}

export default function GlassCard({ children, style, variant = 'primary' }: GlassCardProps) {
  const gradients = {
    primary: ['rgba(0, 210, 255, 0.1)', 'rgba(0, 210, 255, 0.05)'],
    secondary: ['rgba(123, 104, 238, 0.1)', 'rgba(123, 104, 238, 0.05)'],
    accent: ['rgba(255, 215, 0, 0.1)', 'rgba(255, 215, 0, 0.05)'],
  };

  return (
    <View style={[styles.container, style]}>
      <LinearGradient
        colors={gradients[variant]}
        style={styles.gradient}
      >
        <View style={styles.content}>
          {children}
        </View>
      </LinearGradient>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    borderRadius: 16,
    backgroundColor: 'rgba(26, 26, 27, 0.6)',
    borderWidth: 1,
    borderColor: 'rgba(255, 255, 255, 0.1)',
    overflow: 'hidden',
  },
  gradient: {
    flex: 1,
  },
  content: {
    flex: 1,
    padding: 20,
  },
});