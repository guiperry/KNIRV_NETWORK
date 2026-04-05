import { Registry } from './codec';
import { Signer } from './writer';
export declare class Compactor {
    private datasetPath;
    private keyPair?;
    private running;
    private stopCh;
    private compactCallback?;
    constructor(datasetPath: string, keyPair?: Signer);
    maybeCompact(registry: Registry): void;
    start(callback?: () => void): void;
    stop(): void;
    private compact;
    getDatasetPath(): string;
}
//# sourceMappingURL=compactor.d.ts.map