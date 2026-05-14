import React from 'react';
import { View, ViewStyle, StyleSheet } from 'react-native';

interface GlassCardProps {
  children: React.ReactNode;
  style?: ViewStyle;
  className?: string;
}

export const GlassCard: React.FC<GlassCardProps> = ({ children, style, className: _className }) => {
  return (
    <View style={[styles.glassCard, style]} testID="glass-card">
      {children}
    </View>
  );
};

const styles = StyleSheet.create({
  glassCard: {
    backgroundColor: 'rgba(255, 255, 255, 0.1)',
    borderRadius: 12,
    padding: 16,
    borderWidth: 1,
    borderColor: 'rgba(255, 255, 255, 0.2)',
    shadowColor: '#000',
    shadowOffset: {
      width: 0,
      height: 2,
    },
    shadowOpacity: 0.25,
    shadowRadius: 3.84,
    elevation: 5,
  },
});

export default GlassCard;
