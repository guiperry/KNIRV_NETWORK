import React from 'react';
import { Tabs } from 'expo-router';
import { Chrome as Home, Wallet, Bot, Settings } from 'lucide-react-native';

export default function TabLayout() {
  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarStyle: {
          backgroundColor: '#0A0A0B',
          borderTopWidth: 1,
          borderTopColor: '#1A1A1B',
          height: 80,
          paddingBottom: 20,
          paddingTop: 10,
        },
        tabBarActiveTintColor: '#00D2FF',
        tabBarInactiveTintColor: '#666666',
        tabBarLabelStyle: {
          fontFamily: 'Inter-Medium',
          fontSize: 12,
          marginTop: 4,
        },
      }}>
      <Tabs.Screen
        name="index"
        options={{
          title: 'Home',
          tabBarIcon: ({ size, color }: { size: number; color: string }) => (
            <Home size={size} color={color} strokeWidth={2} />
          ),
        }}
      />
      <Tabs.Screen
        name="wallet"
        options={{
          title: 'Wallet',
          tabBarIcon: ({ size, color }: { size: number; color: string }) => (
            <Wallet size={size} color={color} strokeWidth={2} />
          ),
        }}
      />
      <Tabs.Screen
        name="agents"
        options={{
          title: 'AI Agents',
          tabBarIcon: ({ size, color }: { size: number; color: string }) => (
            <Bot size={size} color={color} strokeWidth={2} />
          ),
        }}
      />
      <Tabs.Screen
        name="settings"
        options={{
          title: 'Settings',
          tabBarIcon: ({ size, color }: { size: number; color: string }) => (
            <Settings size={size} color={color} strokeWidth={2} />
          ),
        }}
      />
    </Tabs>
  );
}