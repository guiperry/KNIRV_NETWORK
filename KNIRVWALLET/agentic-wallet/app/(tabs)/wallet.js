import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import React, { useState } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { LinearGradient } from 'expo-linear-gradient';
import { Send, Download, ArrowUpDown, Filter, Search } from 'lucide-react-native';
import GlassCard from '@/components/GlassCard';
import CryptoCard from '@/components/CryptoCard';
import { MetaAccountDashboard } from '../../src/components/MetaAccountDashboard';
import { getXionConfig } from '../../src/config/xion-config';
export default function WalletScreen() {
    const [activeTab, setActiveTab] = useState('all');
    const cryptoAssets = [
        {
            symbol: 'BTC',
            name: 'Bitcoin',
            price: '$47,842.50',
            change: 3.24,
            amount: '0.2845',
            value: '$13,613.25',
            icon: 'https://images.pexels.com/photos/844124/pexels-photo-844124.jpeg?auto=compress&cs=tinysrgb&w=100&h=100&dpr=2',
        },
        {
            symbol: 'ETH',
            name: 'Ethereum',
            price: '$2,845.32',
            change: -1.86,
            amount: '4.2567',
            value: '$12,115.89',
            icon: 'https://images.pexels.com/photos/844124/pexels-photo-844124.jpeg?auto=compress&cs=tinysrgb&w=100&h=100&dpr=2',
        },
        {
            symbol: 'SOL',
            name: 'Solana',
            price: '$98.45',
            change: 8.73,
            amount: '125.42',
            value: '$12,348.29',
            icon: 'https://images.pexels.com/photos/844124/pexels-photo-844124.jpeg?auto=compress&cs=tinysrgb&w=100&h=100&dpr=2',
        },
        {
            symbol: 'MATIC',
            name: 'Polygon',
            price: '$0.87',
            change: 5.12,
            amount: '2847.92',
            value: '$2,477.69',
            icon: 'https://images.pexels.com/photos/844124/pexels-photo-844124.jpeg?auto=compress&cs=tinysrgb&w=100&h=100&dpr=2',
        },
        {
            symbol: 'LINK',
            name: 'Chainlink',
            price: '$14.67',
            change: -2.34,
            amount: '143.85',
            value: '$2,110.11',
            icon: 'https://images.pexels.com/photos/844124/pexels-photo-844124.jpeg?auto=compress&cs=tinysrgb&w=100&h=100&dpr=2',
        },
    ];
    const transactions = [
        {
            type: 'receive',
            asset: 'ETH',
            amount: '+2.5 ETH',
            value: '+$7,113.30',
            time: '2 hours ago',
            status: 'Completed',
        },
        {
            type: 'send',
            asset: 'BTC',
            amount: '-0.0856 BTC',
            value: '-$4,095.15',
            time: '1 day ago',
            status: 'Completed',
        },
        {
            type: 'swap',
            asset: 'SOL → USDC',
            amount: '25 SOL',
            value: '$2,461.25',
            time: '3 days ago',
            status: 'Completed',
        },
    ];
    const tabs = [
        { key: 'all', label: 'All Assets' },
        { key: 'crypto', label: 'Crypto' },
        { key: 'nft', label: 'NFTs' },
        { key: 'defi', label: 'DeFi' },
        { key: 'xion', label: 'XION Meta' },
    ];
    return (_jsx(LinearGradient, { colors: ['#0A0A0B', '#1A1A1B', '#0A0A0B'], style: styles.container, children: _jsx(SafeAreaView, { style: styles.safeArea, children: _jsxs(ScrollView, { style: styles.scrollView, showsVerticalScrollIndicator: false, children: [_jsxs(View, { style: styles.header, children: [_jsx(Text, { style: styles.title, children: "Wallet" }), _jsxs(View, { style: styles.headerActions, children: [_jsx(TouchableOpacity, { style: styles.headerButton, children: _jsx(Search, { size: 20, color: "#666666" }) }), _jsx(TouchableOpacity, { style: styles.headerButton, children: _jsx(Filter, { size: 20, color: "#666666" }) })] })] }), _jsxs(GlassCard, { style: styles.balanceCard, children: [_jsx(Text, { style: styles.balanceLabel, children: "Total Balance" }), _jsx(Text, { style: styles.balanceValue, children: "$48,234.67" }), _jsx(Text, { style: styles.balanceChange, children: "+$5,387.23 (+12.5%) today" }), _jsxs(View, { style: styles.actionButtons, children: [_jsxs(TouchableOpacity, { style: [styles.actionButton, styles.primaryButton], children: [_jsx(Send, { size: 16, color: "#000000" }), _jsx(Text, { style: styles.primaryButtonText, children: "Send" })] }), _jsxs(TouchableOpacity, { style: [styles.actionButton, styles.secondaryButton], children: [_jsx(Download, { size: 16, color: "#00D2FF" }), _jsx(Text, { style: styles.secondaryButtonText, children: "Receive" })] }), _jsxs(TouchableOpacity, { style: [styles.actionButton, styles.secondaryButton], children: [_jsx(ArrowUpDown, { size: 16, color: "#7B68EE" }), _jsx(Text, { style: styles.secondaryButtonText, children: "Swap" })] })] })] }), _jsx(View, { style: styles.tabContainer, children: tabs.map((tab) => (_jsx(TouchableOpacity, { style: [
                                styles.tab,
                                activeTab === tab.key && styles.activeTab,
                            ], onPress: () => setActiveTab(tab.key), children: _jsx(Text, { style: [
                                    styles.tabText,
                                    activeTab === tab.key && styles.activeTabText,
                                ], children: tab.label }) }, tab.key))) }), activeTab === 'xion' ? (_jsx(MetaAccountDashboard, { config: getXionConfig('testnet') })) : (_jsx(_Fragment, { children: _jsx(View, { style: styles.assetsList, children: cryptoAssets.map((asset, index) => (_jsx(CryptoCard, { symbol: asset.symbol, name: asset.name, price: asset.price, change: asset.change, amount: asset.amount, value: asset.value, icon: asset.icon }, index))) }) })), activeTab !== 'xion' && (_jsxs(View, { style: styles.transactions, children: [_jsx(Text, { style: styles.sectionTitle, children: "Recent Transactions" }), transactions.map((tx, index) => (_jsxs(GlassCard, { style: styles.transactionCard, children: [_jsxs(View, { style: styles.transactionHeader, children: [_jsxs(View, { style: styles.transactionInfo, children: [_jsx(Text, { style: styles.transactionType, children: tx.asset }), _jsx(Text, { style: styles.transactionTime, children: tx.time })] }), _jsxs(View, { style: styles.transactionAmounts, children: [_jsx(Text, { style: [
                                                            styles.transactionAmount,
                                                            { color: tx.type === 'receive' ? '#00FF88' : '#FFFFFF' }
                                                        ], children: tx.amount }), _jsx(Text, { style: styles.transactionValue, children: tx.value })] })] }), _jsx(View, { style: [
                                            styles.statusBadge,
                                            { backgroundColor: '#00FF8820' }
                                        ], children: _jsx(Text, { style: [styles.statusText, { color: '#00FF88' }], children: tx.status }) })] }, index)))] }))] }) }) }));
}
const styles = StyleSheet.create({
    container: {
        flex: 1,
    },
    safeArea: {
        flex: 1,
    },
    scrollView: {
        flex: 1,
        paddingHorizontal: 20,
    },
    header: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        paddingTop: 20,
        marginBottom: 24,
    },
    title: {
        fontSize: 28,
        fontFamily: 'Inter-Bold',
        color: '#FFFFFF',
    },
    headerActions: {
        flexDirection: 'row',
        gap: 12,
    },
    headerButton: {
        width: 40,
        height: 40,
        borderRadius: 20,
        backgroundColor: 'rgba(255, 255, 255, 0.1)',
        justifyContent: 'center',
        alignItems: 'center',
    },
    balanceCard: {
        marginBottom: 24,
        alignItems: 'center',
    },
    balanceLabel: {
        fontSize: 14,
        fontFamily: 'Inter-Regular',
        color: '#999999',
        marginBottom: 8,
    },
    balanceValue: {
        fontSize: 36,
        fontFamily: 'Inter-Bold',
        color: '#FFFFFF',
        marginBottom: 8,
    },
    balanceChange: {
        fontSize: 14,
        fontFamily: 'Inter-Medium',
        color: '#00FF88',
        marginBottom: 24,
    },
    actionButtons: {
        flexDirection: 'row',
        gap: 12,
        width: '100%',
    },
    actionButton: {
        flex: 1,
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'center',
        paddingVertical: 12,
        borderRadius: 12,
        gap: 8,
    },
    primaryButton: {
        backgroundColor: '#00D2FF',
    },
    secondaryButton: {
        backgroundColor: 'transparent',
        borderWidth: 1,
        borderColor: 'rgba(255, 255, 255, 0.2)',
    },
    primaryButtonText: {
        fontSize: 14,
        fontFamily: 'Inter-SemiBold',
        color: '#000000',
    },
    secondaryButtonText: {
        fontSize: 14,
        fontFamily: 'Inter-SemiBold',
        color: '#FFFFFF',
    },
    tabContainer: {
        flexDirection: 'row',
        marginBottom: 20,
        backgroundColor: 'rgba(255, 255, 255, 0.05)',
        borderRadius: 12,
        padding: 4,
    },
    tab: {
        flex: 1,
        paddingVertical: 8,
        alignItems: 'center',
        borderRadius: 8,
    },
    activeTab: {
        backgroundColor: '#00D2FF',
    },
    tabText: {
        fontSize: 12,
        fontFamily: 'Inter-Medium',
        color: '#999999',
    },
    activeTabText: {
        color: '#000000',
    },
    assetsList: {
        marginBottom: 24,
    },
    transactions: {
        marginBottom: 40,
    },
    sectionTitle: {
        fontSize: 18,
        fontFamily: 'Inter-SemiBold',
        color: '#FFFFFF',
        marginBottom: 16,
    },
    transactionCard: {
        marginBottom: 12,
    },
    transactionHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 8,
    },
    transactionInfo: {
        flex: 1,
    },
    transactionType: {
        fontSize: 14,
        fontFamily: 'Inter-SemiBold',
        color: '#FFFFFF',
        marginBottom: 2,
    },
    transactionTime: {
        fontSize: 12,
        fontFamily: 'Inter-Regular',
        color: '#999999',
    },
    transactionAmounts: {
        alignItems: 'flex-end',
    },
    transactionAmount: {
        fontSize: 14,
        fontFamily: 'Inter-SemiBold',
        marginBottom: 2,
    },
    transactionValue: {
        fontSize: 12,
        fontFamily: 'Inter-Regular',
        color: '#999999',
    },
    statusBadge: {
        alignSelf: 'flex-start',
        paddingHorizontal: 8,
        paddingVertical: 4,
        borderRadius: 8,
    },
    statusText: {
        fontSize: 10,
        fontFamily: 'Inter-SemiBold',
        textTransform: 'uppercase',
    },
});
//# sourceMappingURL=wallet.js.map