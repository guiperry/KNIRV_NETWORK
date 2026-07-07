import React from 'react';
import styled from 'styled-components';

import { Text, Button } from '@components/atoms';
import Toggle from '@components/atoms/toggle';
import mixins from '@styles/mixins';
import { fonts, getTheme } from '@styles/theme';
import { useCurrentAccount } from '@hooks/use-current-account';
import { useDVEBadges, useAggregateStake } from '@hooks/dve/use-dve-badges';
import type { DVEBadge } from '@services/dve/dve-badge-manager';

export const BadgeInventoryPage: React.FC = () => {
  const { currentAddress } = useCurrentAccount();
  const {
    allBadges,
    loading,
    error,
    aggregateCapabilities,
    toggleBadge,
    refreshBadges,
  } = useDVEBadges(currentAddress);
  const aggregateStake = useAggregateStake(allBadges);

  const trustTierLabel = (tier: DVEBadge['trustTier']): string => {
    switch (tier) {
      case 'root':
        return 'Root';
      case 'verified':
        return 'Verified';
      case 'standard':
      default:
        return 'Standard';
    }
  };

  const trustTierColor = (tier: DVEBadge['trustTier']): string => {
    switch (tier) {
      case 'root':
        return '#ffd700';
      case 'verified':
        return '#00c853';
      case 'standard':
      default:
        return '#9e9e9e';
    }
  };

  const handleMintBadge = () => {
    // TODO: Navigate to mint badge flow
    console.log('Mint badge clicked');
  };

  if (loading && allBadges.length === 0) {
    return (
      <Wrapper>
        <HeaderSection>
          <TitleText>Badge Inventory</TitleText>
        </HeaderSection>
        <EmptyState>
          <EmptyText>Loading badges...</EmptyText>
        </EmptyState>
      </Wrapper>
    );
  }

  if (error && allBadges.length === 0) {
    return (
      <Wrapper>
        <HeaderSection>
          <TitleText>Badge Inventory</TitleText>
        </HeaderSection>
        <EmptyState>
          <EmptyText>Failed to load badges: {error.message}</EmptyText>
          <RetryButton onClick={refreshBadges}>Retry</RetryButton>
        </EmptyState>
      </Wrapper>
    );
  }

  return (
    <Wrapper>
      <HeaderSection>
        <TitleText>Badge Inventory</TitleText>
        <BadgeCount>{allBadges.length} badge(s)</BadgeCount>
      </HeaderSection>

      {/* Summary Stats */}
      <SummaryCard>
        <SummaryGrid>
          <SummaryItem>
            <SummaryValue>{allBadges.filter((b) => b.active).length}</SummaryValue>
            <SummaryLabel>Active</SummaryLabel>
          </SummaryItem>
          <SummaryItem>
            <SummaryValue>{allBadges.length}</SummaryValue>
            <SummaryLabel>Total</SummaryLabel>
          </SummaryItem>
          <SummaryItem>
            <SummaryValue>{aggregateCapabilities.length}</SummaryValue>
            <SummaryLabel>Capabilities</SummaryLabel>
          </SummaryItem>
          <SummaryItem>
            <SummaryValue>{aggregateStake.toLocaleString()}</SummaryValue>
            <SummaryLabel>Stake (NRN)</SummaryLabel>
          </SummaryItem>
        </SummaryGrid>
      </SummaryCard>

      {/* Capabilities List */}
      {aggregateCapabilities.length > 0 && (
        <CapabilitiesCard>
          <SectionTitle>Aggregate Capabilities</SectionTitle>
          <CapabilityChips>
            {aggregateCapabilities.map((cap) => (
              <CapabilityChip key={cap}>{cap}</CapabilityChip>
            ))}
          </CapabilityChips>
        </CapabilitiesCard>
      )}

      {/* Badge Grid */}
      {allBadges.length === 0 ? (
        <EmptyState>
          <EmptyText>No DVE badges found for this wallet.</EmptyText>
          <MintButton onClick={handleMintBadge}>Mint Badge</MintButton>
        </EmptyState>
      ) : (
        <>
          <BadgeGrid>
            {allBadges.map((badge) => (
              <BadgeCard key={badge.nftTokenID}>
                <BadgeHeader>
                  <BadgeName>{badge.nftTokenID.substring(0, 10)}...</BadgeName>
                  <Toggle
                    activated={badge.active}
                    onToggle={(active) =>
                      toggleBadge(badge.nftTokenID, active)
                    }
                  />
                </BadgeHeader>

                {/* Trust Tier */}
                <TierBadge tier={badge.trustTier}>
                  <TierDot color={trustTierColor(badge.trustTier)} />
                  {trustTierLabel(badge.trustTier)}
                </TierBadge>

                {/* Capabilities */}
                {badge.capabilities.length > 0 && (
                  <CapabilityRow>
                    {badge.capabilities.slice(0, 3).map((cap) => (
                      <MiniChip key={cap}>{cap}</MiniChip>
                    ))}
                    {badge.capabilities.length > 3 && (
                      <ExtraChip>+{badge.capabilities.length - 3}</ExtraChip>
                    )}
                  </CapabilityRow>
                )}

                {/* Policy Count */}
                <PolicyRow>
                  <PolicyLabel>Policies</PolicyLabel>
                  <PolicyCount>{badge.attachedPolicies.length}</PolicyCount>
                </PolicyRow>
              </BadgeCard>
            ))}
          </BadgeGrid>

          <MintButton onClick={handleMintBadge}>Mint Badge</MintButton>
        </>
      )}
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

const BadgeCount = styled.span`
  ${fonts.body2Reg};
  color: ${getTheme('neutral', '_4')};
`;

const SummaryCard = styled.div`
  width: 100%;
  background-color: ${getTheme('neutral', '_8')};
  border-radius: 12px;
  padding: 16px;
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

const CapabilitiesCard = styled.div`
  width: 100%;
  background-color: ${getTheme('neutral', '_8')};
  border-radius: 12px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
`;

const SectionTitle = styled.span`
  ${fonts.body1Bold};
  color: ${getTheme('neutral', '_1')};
`;

const CapabilityChips = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
`;

const CapabilityChip = styled.div`
  padding: 4px 10px;
  border-radius: 12px;
  background-color: ${getTheme('primary', '_7')};
  color: ${getTheme('neutral', '_1')};
  ${fonts.body3Reg};
  font-size: 11px;
`;

const BadgeGrid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  width: 100%;
`;

const BadgeCard = styled.div`
  background-color: ${getTheme('neutral', '_8')};
  border-radius: 12px;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  border: 1px solid ${getTheme('neutral', '_7')};
`;

const BadgeHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
`;

const BadgeName = styled.span`
  ${fonts.body1Bold};
  color: ${getTheme('neutral', '_1')};
`;

const TierBadge = styled.div<{ tier: string }>`
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border-radius: 8px;
  background-color: ${getTheme('neutral', '_9')};
  ${fonts.body3Reg};
  font-size: 10px;
  color: ${getTheme('neutral', '_2')};
  align-self: flex-start;
`;

const TierDot = styled.div<{ color: string }>`
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: ${({ color }) => color};
`;

const CapabilityRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
`;

const MiniChip = styled.div`
  padding: 2px 6px;
  border-radius: 8px;
  background-color: ${getTheme('neutral', '_9')};
  color: ${getTheme('neutral', '_3')};
  ${fonts.body3Reg};
  font-size: 9px;
`;

const ExtraChip = styled.div`
  padding: 2px 6px;
  border-radius: 8px;
  background-color: ${getTheme('neutral', '_9')};
  color: ${getTheme('primary', '_6')};
  ${fonts.body3Reg};
  font-size: 9px;
`;

const PolicyRow = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
`;

const PolicyLabel = styled.span`
  ${fonts.body3Reg};
  color: ${getTheme('neutral', '_4')};
  font-size: 11px;
`;

const PolicyCount = styled.span`
  ${fonts.body3Reg};
  color: ${getTheme('neutral', '_1')};
  font-size: 11px;
`;

const EmptyState = styled.div`
  ${mixins.flex({ direction: 'column', align: 'center' })};
  padding: 40px 20px;
  gap: 16px;
  width: 100%;
`;

const EmptyText = styled.span`
  ${fonts.body2Reg};
  color: ${getTheme('neutral', '_4')};
  text-align: center;
`;

const RetryButton = styled(Button)`
  padding: 8px 24px;
  border-radius: 8px;
  background-color: ${getTheme('primary', '_7')};
  color: ${getTheme('neutral', '_1')};
  ${fonts.body2Bold};
  cursor: pointer;
`;

const MintButton = styled.button`
  width: 100%;
  padding: 14px;
  border-radius: 12px;
  background-color: ${getTheme('primary', '_7')};
  color: ${getTheme('neutral', '_1')};
  ${fonts.body1Bold};
  border: none;
  cursor: pointer;
  transition: background-color 0.2s ease;

  &:hover {
    background-color: ${getTheme('primary', '_6')};
  }
`;
