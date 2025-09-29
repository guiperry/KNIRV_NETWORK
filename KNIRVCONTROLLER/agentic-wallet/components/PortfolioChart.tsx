import React from 'react';
import { View, StyleSheet, Dimensions, Text, ColorValue } from 'react-native';
import { LinearGradient } from 'expo-linear-gradient';

const { width } = Dimensions.get('window');

interface PortfolioChartProps {
  data: number[];
  change: number;
}

export default function PortfolioChart({ data, change }: PortfolioChartProps) {
  const chartWidth = width - 80;
  const chartHeight = 120;
  const maxValue = Math.max(...data);
  const minValue = Math.min(...data);
  const range = maxValue - minValue;

  const pathData = data
    .map((value, index) => {
      const x = (index / (data.length - 1)) * chartWidth;
      const y = chartHeight - ((value - minValue) / range) * chartHeight;
      return index === 0 ? `M${x},${y}` : `L${x},${y}`;
    })
    .join(' ');

  const isPositive = change >= 0;
  const gradientColors: readonly [ColorValue, ColorValue] = isPositive
    ? ['rgba(0, 210, 255, 0.3)', 'rgba(0, 210, 255, 0)'] as const
    : ['rgba(255, 99, 99, 0.3)', 'rgba(255, 99, 99, 0)'] as const;

  return (
    <View style={styles.container}>
      <View style={styles.chartContainer}>
        <LinearGradient
          colors={gradientColors}
          style={StyleSheet.absoluteFillObject}
        />
        <View style={styles.grid}>
          {Array.from({ length: 5 }).map((_, i) => (
            <View key={i} style={[styles.gridLine, { top: (i * chartHeight) / 4 }]} />
          ))}
        </View>
        <View style={styles.pathContainer}>
          {/* Simulated chart line */}
          {data.map((_, index) => {
            const x = (index / (data.length - 1)) * chartWidth;
            const y = chartHeight - ((data[index] - minValue) / range) * chartHeight;
            return (
              <View
                key={index}
                style={[
                  styles.point,
                  {
                    left: x - 2,
                    top: y - 2,
                    backgroundColor: isPositive ? '#00D2FF' : '#FF6363',
                  },
                ]}
              />
            );
          })}
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    height: 120,
    marginVertical: 16,
  },
  chartContainer: {
    flex: 1,
    position: 'relative',
  },
  grid: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
  },
  gridLine: {
    position: 'absolute',
    left: 0,
    right: 0,
    height: 1,
    backgroundColor: 'rgba(255, 255, 255, 0.05)',
  },
  pathContainer: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
  },
  point: {
    position: 'absolute',
    width: 4,
    height: 4,
    borderRadius: 2,
  },
});