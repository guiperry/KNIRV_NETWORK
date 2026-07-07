import { Account, AccountInfo } from './account';
import { AirgapAccount } from './airgap-account';
import { LedgerAccount } from './ledger-account';
import { SeedAccount } from './seed-account';
import { SingleAccount } from './single-account';

export function isSeedAccount(account: Account): account is SeedAccount {
  return account.type === 'HD';
}

export function isLedgerAccount(account: Account): account is LedgerAccount {
  return account.type === 'LEDGER';
}

export function isSingleAccount(account: Account): account is SingleAccount {
  return account.type === 'WEB3_AUTH' || account.type === 'PRIVATE_KEY';
}

export function isAirgapAccount(account: Account): account is AirgapAccount {
  return account.type === 'ADDRESS';
}

export function hasPrivateKeyAccount(account: Account): boolean {
  return isSeedAccount(account) || isSingleAccount(account);
}

export function hasHDPath(account: Account): account is SeedAccount | LedgerAccount {
  return isSeedAccount(account) || isLedgerAccount(account);
}

export function serializeAccount(account: Account) {
  return JSON.stringify(account.toData());
}

export function deserializeAccount(plain: string) {
  const accountInfo: AccountInfo = JSON.parse(plain);
  if (accountInfo.type === 'HD') {
    return SeedAccount.fromData(accountInfo);
  }
  if (accountInfo.type === 'LEDGER') {
    return LedgerAccount.fromData(accountInfo);
  }
  if (accountInfo.type === 'PRIVATE_KEY' || accountInfo.type === 'WEB3_AUTH') {
    return SingleAccount.fromData(accountInfo);
  }
  if (accountInfo.type === 'ADDRESS') {
    return AirgapAccount.fromData(accountInfo);
  }
  throw new Error('Invalid account type');
}
