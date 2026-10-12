import * as fs from 'fs';
import * as path from 'path';

export interface WALEntry {
  frameId: string;
  lastGoodLength: number;
  committed: boolean;
}

// On disk the WAL is Go's format: one JSON object per line with snake_case keys
// (internal/storage/wal.go). Legacy camelCase lines written by older TS
// versions are still read.
interface WALLine {
  frame_id: string;
  last_good_length: number;
  committed: boolean;
}

const toLine = (e: WALEntry): WALLine => ({
  frame_id: e.frameId,
  last_good_length: e.lastGoodLength,
  committed: e.committed,
});

function fromLine(raw: any): WALEntry | null {
  if (!raw || typeof raw !== 'object') return null;
  const frameId = raw.frame_id ?? raw.frameId;
  const lastGoodLength = raw.last_good_length ?? raw.lastGoodLength;
  if (typeof frameId !== 'string' || typeof lastGoodLength !== 'number') return null;
  return { frameId, lastGoodLength, committed: !!raw.committed };
}

export class WAL {
  private path: string;

  constructor(path: string) {
    this.path = path;
  }

  async begin(entry: WALEntry): Promise<void> {
    await fs.promises.appendFile(this.path, JSON.stringify(toLine(entry)) + '\n', 'utf8');
  }

  async commit(frameId: string): Promise<void> {
    const entries = await this.readEntries();
    const lines = entries.map((entry) => {
      if (entry.frameId === frameId) entry.committed = true;
      return JSON.stringify(toLine(entry));
    });
    await fs.promises.writeFile(this.path, lines.map((l) => l + '\n').join(''), 'utf8');
  }

  /** Smallest lastGoodLength among uncommitted entries, or -1 if none. */
  async recover(): Promise<number> {
    const entries = await this.readEntries();

    let minUncommitted = -1;
    for (const entry of entries) {
      if (!entry.committed) {
        if (minUncommitted === -1 || entry.lastGoodLength < minUncommitted) {
          minUncommitted = entry.lastGoodLength;
        }
      }
    }
    return minUncommitted;
  }

  async truncate(): Promise<void> {
    try {
      await fs.promises.unlink(this.path);
    } catch {
      // Ignore if file doesn't exist
    }
  }

  // Skips malformed lines (e.g. a torn final write) instead of discarding the log.
  private async readEntries(): Promise<WALEntry[]> {
    let content: string;
    try {
      content = await fs.promises.readFile(this.path, 'utf8');
    } catch {
      return [];
    }
    const out: WALEntry[] = [];
    for (const line of content.split('\n')) {
      if (!line.trim()) continue;
      try {
        const e = fromLine(JSON.parse(line));
        if (e) out.push(e);
      } catch {
        // torn or corrupt line
      }
    }
    return out;
  }
}
