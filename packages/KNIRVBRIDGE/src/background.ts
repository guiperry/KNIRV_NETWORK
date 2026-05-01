import { AlarmKey, SCHEDULE_ALARMS } from '@common/constants/alarm-key.constant';
import { ADENA_WALLET_EXTENSION_ID } from '@common/constants/storage.constant';
import { DVE_CONSTANTS } from '@common/constants/dve.constant';
import { MemoryProvider } from '@common/provider/memory/memory-provider';
import { ChromeLocalStorage } from '@common/storage';
import { CommandHandler } from '@inject/message/command-handler';
import { CommandMessage, isCommandMessageData } from '@inject/message/command-message';
import { clearInMemoryKey } from '@inject/message/commands/encrypt';
import { MessageHandler } from './inject/message';
import { DVEIdentity, deriveDVEIdentity } from '@services/dve/dve-identity';
import { DVERegistryService } from '@services/dve/dve-registry';
import { DVEWebSocketClient } from '@services/dve/dve-ws-client';
import { DVEBadgeManager } from '@services/dve/dve-badge-manager';
import { ValidationRuntime } from '@services/dve/validation-runtime';

const inMemoryProvider = new MemoryProvider();
inMemoryProvider.init();

// DVE state variables
let dveIdentity: DVEIdentity | null = null;
let dveRegistry: DVERegistryService | null = null;
let dveWSClient: DVEWebSocketClient | null = null;
let dveBadgeManager: DVEBadgeManager | null = null;
let dveValidationRuntime: ValidationRuntime | null = null;
let dveEnabled: boolean = false;

// DVE server URL — configurable via chrome.storage
const DEFAULT_DVE_SERVER_URL = 'http://localhost:8084';
let dveServerURL: string = DEFAULT_DVE_SERVER_URL;

initAlarms();

function existsWallet(): Promise<boolean> {
  const storage = new ChromeLocalStorage();
  return storage
    .get('SERIALIZED')
    .then(async (serialized) => typeof serialized === 'string' && serialized.length !== 0)
    .catch(() => false);
}

function setupPopup(existWallet: boolean): boolean {
  const popupUri = existWallet ? 'popup.html' : '';
  chrome.action.setPopup({ popup: popupUri });
  return true;
}

/**
 * Initialize DVE when a wallet unlocks. Reads config from storage,
 * derives identity, creates services, registers with server,
 * connects WebSocket, and starts badge monitoring.
 */
async function initializeDVE(walletAddress: string, authToken: string): Promise<void> {
  if (dveEnabled) {
    console.info('DVE already initialized, skipping');
    return;
  }

  if (!walletAddress || !authToken) {
    console.warn('DVE initialize skipped: missing wallet address or auth token');
    return;
  }

  try {
    // Load DVE server URL from storage if configured
    const storageResult = await chrome.storage.local.get('DVE_SERVER_URL');
    if (storageResult.DVE_SERVER_URL) {
      dveServerURL = storageResult.DVE_SERVER_URL;
    }

    // Load DVE enabled flag from storage
    const enabledResult = await chrome.storage.local.get('DVE_ENABLED');
    if (enabledResult.DVE_ENABLED === false) {
      console.info('DVE is disabled via storage setting');
      return;
    }

    // Derive identity from wallet address
    dveIdentity = await deriveDVEIdentity(walletAddress);
    console.info('DVE identity derived:', dveIdentity.nodeID, dveIdentity.dveURI);

    // Create registry service
    dveRegistry = new DVERegistryService(dveServerURL, authToken);

    // Create badge manager and fetch badges
    dveBadgeManager = new DVEBadgeManager(dveServerURL);
    const badges = await dveBadgeManager.getBadgesFromWallet(walletAddress);
    const capabilities = dveBadgeManager.computeAggregateCapabilities(badges);
    const badgeNFTIDs = badges.map((b) => b.nftTokenID);
    console.info('DVE badges fetched:', badgeNFTIDs.length, 'badges, capabilities:', capabilities);

    // Register with the server
    const serverNodeID = await dveRegistry.register(dveIdentity, capabilities, badgeNFTIDs);
    console.info('DVE registered with server, node ID:', serverNodeID);

    // Create and connect WebSocket
    dveWSClient = new DVEWebSocketClient(dveServerURL, authToken);
    dveWSClient.connect();

    // Create validation runtime for task execution
    dveValidationRuntime = new ValidationRuntime();

    // Start watching badge changes to keep capabilities in sync
    if (dveBadgeManager) {
      dveBadgeManager.watchBadgeChanges(async (updatedBadges) => {
        if (!dveRegistry || !dveIdentity) {
          return;
        }
        try {
          const newCapabilities = dveBadgeManager!.computeAggregateCapabilities(updatedBadges);
          const newBadgeNFTIDs = updatedBadges.map((b) => b.nftTokenID);
          await dveRegistry.syncCapabilities(dveIdentity.nodeID, newCapabilities);
          await dveRegistry.syncBadges(dveIdentity.nodeID, newBadgeNFTIDs);
          console.info('DVE badges synced to server');
        } catch (err) {
          console.error('Failed to sync DVE badges to server:', err);
        }
      });
    }

    dveEnabled = true;
    console.info('DVE initialized successfully');
  } catch (error) {
    console.error('Failed to initialize DVE:', error);
    dveEnabled = false;
  }
}

/**
 * Shut down DVE — deregister from server, disconnect WebSocket,
 * destroy badge manager and validation runtime.
 */
async function shutdownDVE(): Promise<void> {
  if (!dveEnabled) {
    return;
  }

  console.info('Shutting down DVE...');

  // Deregister from server
  if (dveRegistry && dveIdentity) {
    try {
      await dveRegistry.deregister(dveIdentity.nodeID);
      console.info('DVE deregistered from server');
    } catch (err) {
      console.error('Failed to deregister DVE:', err);
    }
  }

  // Disconnect WebSocket
  if (dveWSClient) {
    dveWSClient.disconnect();
    dveWSClient = null;
  }

  // Destroy badge manager
  if (dveBadgeManager) {
    dveBadgeManager.destroy();
    dveBadgeManager = null;
  }

  // Clean up validation runtime
  if (dveValidationRuntime) {
    dveValidationRuntime = null;
  }

  dveRegistry = null;
  dveIdentity = null;
  dveEnabled = false;

  console.info('DVE shut down complete');
}

/**
 * Read the current wallet address from storage.
 */
async function getWalletAddress(): Promise<string | null> {
  try {
    const result = await chrome.storage.local.get('CURRENT_WALLET_ADDRESS');
    return result.CURRENT_WALLET_ADDRESS || null;
  } catch {
    return null;
  }
}

/**
 * Read the current auth token from storage.
 */
async function getAuthToken(): Promise<string | null> {
  try {
    const result = await chrome.storage.session.get('AUTH_TOKEN');
    return result.AUTH_TOKEN || null;
  } catch {
    return null;
  }
}

chrome.runtime.onInstalled.addListener((details) => {
  if (details.reason === 'install') {
    chrome.tabs.create({
      url: chrome.runtime.getURL('/register.html'),
    });
  } else if (details.reason === 'update') {
    existsWallet().then((existWallet) => {
      setupPopup(existWallet);
    });
  }
});

chrome.tabs.onCreated.addListener(() => {
  existsWallet().then((existWallet) => {
    setupPopup(existWallet);
  });
});

chrome.action.onClicked.addListener(async () => {
  existsWallet().then((existWallet) => {
    setupPopup(existWallet);
    if (!existWallet) {
      chrome.tabs.create({
        url: chrome.runtime.getURL('/register.html'),
      });
    }
  });
});

chrome.runtime.onConnect.addListener(async (port) => {
  if (port.name !== ADENA_WALLET_EXTENSION_ID) {
    return;
  }

  inMemoryProvider.addConnection();
  inMemoryProvider.updateExpiredTimeBy(null);

  // When wallet connects, attempt to start DVE
  const walletAddress = await getWalletAddress();
  const authToken = await getAuthToken();
  if (walletAddress && authToken) {
    initializeDVE(walletAddress, authToken).catch((err) => {
      console.error('DVE initialization from connect listener failed:', err);
    });
  }

  port.onDisconnect.addListener(async () => {
    inMemoryProvider.removeConnection();

    // When wallet disconnects, shut down DVE
    await shutdownDVE();

    if (!inMemoryProvider.isActive()) {
      const expiredTime = new Date().getTime() + inMemoryProvider.getExpiredPasswordDurationTime();
      inMemoryProvider.updateExpiredTimeBy(expiredTime);

      console.info('Password Expired time:', new Date(expiredTime));
    }
  });
});

chrome.alarms.onAlarm.addListener(async (alarm) => {
  try {
    const currentTime = new Date().getTime();
    chrome.storage.local.set({ SESSION: currentTime });

    switch (alarm?.name) {
      case AlarmKey.EXPIRED_PASSWORD:
        if (!inMemoryProvider.isExpired(currentTime)) {
          return;
        }

        await chrome.storage.session.clear();
        await clearInMemoryKey(inMemoryProvider);

        inMemoryProvider.updateExpiredTimeBy(null);
        console.info('Password Expired');

        break;
      case AlarmKey.DVE_HEARTBEAT:
        if (dveWSClient) {
          dveWSClient.sendHeartbeat();
        }
        break;
      default:
        break;
    }
  } catch (error) {
    console.error(error);
  }

  return true;
});

chrome.tabs.onUpdated.addListener(async (tabId, changeInfo) => {
  if (changeInfo.status === 'complete') {
    try {
      chrome.tabs
        .sendMessage(
          tabId,
          CommandMessage.command('checkMetadata', {
            gnoMessageInfo: null,
            gnoConnectInfo: null,
          }),
        )
        .catch(console.info);
    } catch (e) {
      console.warn('Failed to send message(checkMetadata)', e);
    }
  }
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (isCommandMessageData(message)) {
    CommandHandler.createHandler(inMemoryProvider, message, sender, sendResponse);
    return true;
  }

  return MessageHandler.createHandler(inMemoryProvider, message, sender, sendResponse);
});

function initAlarms(): void {
  SCHEDULE_ALARMS.map(initAlarmWithDelay);
}

function initAlarmWithDelay(alarm: { key: string; periodInMinutes: number; delay: number }): void {
  if (alarm.delay === 0) {
    chrome.alarms.create(alarm.key, {
      periodInMinutes: alarm.periodInMinutes,
    });
    return;
  }

  setTimeout(
    () =>
      chrome.alarms.create(alarm.key, {
        periodInMinutes: alarm.periodInMinutes,
      }),
    alarm.delay,
  );
}
