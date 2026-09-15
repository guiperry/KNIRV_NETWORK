/**
 * Wallet helpers for DVE ownership (FE-2).
 *
 * The dashboard's "connected wallet" is the Data Fabric wallet configured
 * during onboarding (dataWalletConfig.walletName). There is no browser
 * extension/on-chain signer in this codebase, so the wallet identifier is the
 * wallet name itself — the same value the dashboard uses everywhere else as the
 * wallet/org identifier. Swapping this for a real on-chain address later only
 * requires changing resolveOwnerWallet.
 */

// Minimum NRN stake required to create a DVE, mirrored from
// DVECreationService.minStakeAmount in backend_server.
export const MIN_NRN_STAKE = 1000;

export const isValidStakeAmount = (value: number): boolean => {
  return Number.isFinite(value) && value >= MIN_NRN_STAKE;
};

export const stakeAmountError = (value: number): string | null => {
  if (!Number.isFinite(value) || value <= 0) {
    return 'Please enter a valid NRN stake amount.';
  }
  if (value < MIN_NRN_STAKE) {
    return `Minimum stake is ${MIN_NRN_STAKE} NRN.`;
  }
  return null;
};

export interface OwnerWalletContext {
  walletName?: string;
}

export const resolveOwnerWallet = (ctx: OwnerWalletContext): string => {
  const wallet = ctx.walletName?.trim();
  return wallet ?? '';
};