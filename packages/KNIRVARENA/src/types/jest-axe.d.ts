declare module 'jest-axe' {
  export function axe(container: Element, options?: unknown): Promise<unknown>;
  export function toHaveNoViolations(): unknown;
}

declare namespace jest {
  interface Matchers<R> {
    toHaveNoViolations(): R;
  }
}
