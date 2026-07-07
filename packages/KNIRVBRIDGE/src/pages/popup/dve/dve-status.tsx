import React, { useEffect } from 'react';
import styled from 'styled-components';

import { Copy, StatusDot, Text } from '@components/atoms';
import Toggle from '@components/atoms/toggle';
import mixins from '@styles/mixins';
import { fonts, getTheme } from '@styles/theme';
import { useCurrentAccount } from '@hooks/use-current-account';
import { useDVEStatus, useDVETasks } from '@hooks/dve/use-dve-status';
import { useDVEBadges, useAggregateStake } from '@hooks/dve/use-dve-badges';

export const DVEStatusPage: React.FC = () => {
  const { currentAddress } = useCurrentAccount();
  const { status, loading, error, connect, disconnect } =
    useDVEStatus(currentAddress);
  const { tasks } = useDVETasks(currentAddress);
  const { badges, aggregateCapabilities } = useDVEBadges(currentAddress);
  const aggregateStake = useAggregateStake(badges);

  // Auto-connect when wallet address is available
  useEffect(() => {
    if (currentAddress && !status.isActive && status.status === 'offline') {
      connect();
    }
  }, [currentAddress, status.isActive, status.status, connect]);

  const truncatedNodeID =
    status.nodeID.length > 16
      ? `${status.nodeID.substring(0, 8)}...${status.nodeID.substring(status.nodeID.length - 4)}`
      : status.nodeID;

  const handleToggleDVE = () => {
    if (status.isActive) {
      disconnect();
    } else {
      connect();
    }
  };

  const statusLabel =
    status.status === 'online'
      ? 'Online'
      : status.status === 'connecting'
        ? 'Connecting...'
        : 'Offline';

  return (
    <Wrapper>
      <HeaderSection>
        <TitleText>DVE Status</TitleText>
      </HeaderSection>

      {/* Node Identity Card */}
      <Card>
        <CardHeader>
          <NodeLabel>Node ID</NodeLabel>
        </CardHeader>
        <NodeIDRow>
          <NodeIDText>{truncatedNodeID}</NodeIDText>
          {status.nodeID && <Copy copyStr={status.nodeID} />}
        </NodeIDRow>
        <DVERow>
          <DVELabel>DVE URI</DVELabel>
          <DVEURIText>{status.dveURI || '-'}</DVEURIText>
        </DVERow>
        <StatusRow>
          <StatusLabel>Status</StatusLabel>
          <StatusIndicator>
            <StatusDot
              status={status.status === 'online'}
              tooltipText={statusLabel}
            />
            <StatusText>{statusLabel}</StatusText>
          </StatusIndicator>
        </StatusRow>
      </Card>

      {/* DVE Activation Toggle */}
      <Card>
        <CardHeader>
          <ActivationTitle>DVE Node</ActivationTitle>
          <Toggle
            activated={status.isActive}
            onToggle={handleToggleDVE}
          />
        </CardHeader>
        {loading && <LoadingText>Connecting DVE node...</LoadingText>}
        {error && <ErrorText>Error: {error.message}</ErrorText>}
      </Card>

      {/* Badge & Capabilities Summary */}
      <Card>
        <SummaryGrid>
          <SummaryItem>
            <SummaryValue>{badges.length}</SummaryValue>
            <SummaryLabel>Active Badges</SummaryLabel>
          </SummaryItem>
          <SummaryItem>
            <SummaryValue>{aggregateCapabilities.length}</SummaryValue>
            <SummaryLabel>Capabilities</SummaryLabel>
          </SummaryItem>
        </SummaryGrid>
      </Card>

      {/* Task Counters */}
      <Card>
        <CardHeader>
          <SectionTitle>Tasks</SectionTitle>
        </CardHeader>
        <TaskGrid>
          <TaskItem>
            <TaskValue pending>{tasks.pending}</TaskValue>
            <TaskLabel>Pending</TaskLabel>
          </TaskItem>
          <TaskItem>
            <TaskValue completed>{tasks.completed}</TaskValue>
            <TaskLabel>Completed</TaskLabel>
          </TaskItem>
          <TaskItem>
            <TaskValue failed>{tasks.failed}</TaskValue>
            <TaskLabel>Failed</TaskLabel>
          </TaskItem>
        </TaskGrid>
      </Card>

      {/* Stake & Reputation */}
      <Card>
        <DetailRow>
          <DetailLabel>Stake Amount</DetailLabel>
          <DetailValue>{aggregateStake.toLocaleString()} NRN</DetailValue>
        </DetailRow>
        <DetailRow>
          <DetailLabel>Reputation Score</DetailLabel>
          <DetailValue>{status.reputationScore}</DetailValue>
        </DetailRow>
      </Card>
    </Wrapper>
  );
};

// Styled components

const Wrapper = styled.main`
  ${mixins.flex({ justify: 'flex-start' })};
  width: 100%;
  height: 100%;
  padding: 24px 20px 80px 20px;
  overflow-y: auto;
  gap: 16px;
`;

const HeaderSection = styled.div`
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
`;

const TitleText = styled.span`
  ${fonts.header4};
  color: ${getTheme('neutral', '_1')};
`;

const Card = styled.div`
  width: 100%;
  background-color: ${getTheme('neutral', '_8')};
  border-radius: 12px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
`;

const CardHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
`;

const NodeLabel = styled.span`
  ${fonts.body2Reg};
  color: ${getTheme('neutral', '_4')};
`;

const NodeIDRow = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`;

const NodeIDText = styled.span`
  ${fonts.body1Reg};
  color: ${getTheme('neutral', '_1')};
  font-family: monospace;
`;

const DVERow = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
`;

const DVELabel = styled.span`
  ${fonts.body2Reg};
  color: ${getTheme('neutral', '_4')};
`;

const DVEURIText = styled.span`
  ${fonts.body2Reg};
  color: ${getTheme('neutral', '_2')};
  font-family: monospace;
  word-break: break-all;
`;

const StatusRow = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
`;

const StatusLabel = styled.span`
  ${fonts.body2Reg};
  color: ${getTheme('neutral', '_4')};
`;

const StatusIndicator = styled.div`
  display: flex;
  align-items: center;
  gap: 6px;
`;

const StatusText = styled.span`
  ${fonts.body2Reg};
  color: ${getTheme('neutral', '_1')};
`;

const ActivationTitle = styled.span`
  ${fonts.body1Bold};
  color: ${getTheme('neutral', '_1')};
`;

const LoadingText = styled.span`
  ${fonts.body3Reg};
  color: ${getTheme('neutral', '_5')};
`;

const ErrorText = styled.span`
  ${fonts.body3Reg};
  color: ${getTheme('red', '_6')};
`;

const SummaryGrid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
`;

const SummaryItem = styled.div`
  ${mixins.flex({ direction: 'column', align: 'center' })};
  gap: 4px;
  padding: 12px;
  background-color: ${getTheme('neutral', '_9')};
  border-radius: 8px;
`;

const SummaryValue = styled.span`
  ${fonts.header5};
  color: ${getTheme('primary', '_6')};
`;

const SummaryLabel = styled.span`
  ${fonts.body3Reg};
  color: ${getTheme('neutral', '_4')};
`;

const SectionTitle = styled.span`
  ${fonts.body1Bold};
  color: ${getTheme('neutral', '_1')};
`;

const TaskGrid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 8px;
`;

const TaskItem = styled.div`
  ${mixins.flex({ direction: 'column', align: 'center' })};
  gap: 4px;
  padding: 8px;
  background-color: ${getTheme('neutral', '_9')};
  border-radius: 8px;
`;

const TaskValue = styled.span<{ pending?: boolean; completed?: boolean; failed?: boolean }>`
  ${fonts.header5};
  color: ${({ pending, completed, failed, theme }) =>
    pending
      ? theme.neutral._1
      : completed
        ? theme.green._6
        : failed
          ? theme.red._6
          : theme.neutral._1};
`;

const TaskLabel = styled.span`
  ${fonts.body3Reg};
  color: ${getTheme('neutral', '_4')};
`;

const DetailRow = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
`;

const DetailLabel = styled.span`
  ${fonts.body2Reg};
  color: ${getTheme('neutral', '_4')};
`;

const DetailValue = styled.span`
  ${fonts.body1Bold};
  color: ${getTheme('neutral', '_1')};
`;
