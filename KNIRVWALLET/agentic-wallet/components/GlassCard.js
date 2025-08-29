import { jsx as _jsx } from "react/jsx-runtime";
import React from 'react';
import { View, StyleSheet } from 'react-native';
import { LinearGradient } from 'expo-linear-gradient';
export default function GlassCard({ children, style, variant = 'primary' }) {
    const gradients = {
        primary: ['rgba(0, 210, 255, 0.1)', 'rgba(0, 210, 255, 0.05)'],
        secondary: ['rgba(123, 104, 238, 0.1)', 'rgba(123, 104, 238, 0.05)'],
        accent: ['rgba(255, 215, 0, 0.1)', 'rgba(255, 215, 0, 0.05)'],
    };
    return (_jsx(View, { style: [styles.container, style], children: _jsx(LinearGradient, { colors: gradients[variant], style: styles.gradient, children: _jsx(View, { style: styles.content, children: children }) }) }));
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
//# sourceMappingURL=GlassCard.js.map