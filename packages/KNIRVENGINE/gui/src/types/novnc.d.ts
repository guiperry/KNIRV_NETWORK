declare module '@novnc/novnc' {
  export interface RFBOptions {
    shared?: boolean;
    viewOnly?: boolean;
    qualityLevel?: number;
    [key: string]: unknown;
  }

  export default class RFB {
    constructor(target: HTMLElement, url: string | URL, options?: RFBOptions);
    background: string;
    scaleViewport: boolean;
    resizeSession: boolean;
    addEventListener(type: string, listener: (event: unknown) => void): void;
    removeEventListener(type: string, listener: (event: unknown) => void): void;
    disconnect(): void;
    connect(url?: string | URL): void;
  }
}
