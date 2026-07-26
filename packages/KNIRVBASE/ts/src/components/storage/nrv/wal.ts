import * as fs from 'fs';
import * as path from 'path';

export interface WALEntry {
  frameId: string;
  lastGoodLength: number;
  committed: boolean;
}

export class WAL {
  private path: string;

  constructor(path: string) {
    this.path = path;
  }

  async begin(entry: WALEntry): Promise<void> {
    const data = JSON.stringify(entry) + '\n';
    await fs.promises.appendFile(this.path, data, 'utf8');
  }

  async commit(frameId: string): Promise<void> {
    const entries = await this.readEntries();

    const lines: string[] = [];
    for (const entry of entries) {
      if (entry.frameId === frameId) {
        entry.committed = true;
      }
      lines.push(JSON.stringify(entry));
    }

    await fs.promises.writeFile(this.path, lines.join('\n') + '\n', 'utf8');
  }

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

    if (minUncommitted === -1 && entries.length > 0) {
      return -1;
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

  private async readEntries(): Promise<WALEntry[]> {
    try {
      const content = await fs.promises.readFile(this.path, 'utf8');
      const lines = content.split('\n').filter(line => line.trim());
      return lines.map(line => JSON.parse(line) as WALEntry);
    } catch {
      return [];
    }
  }
}
