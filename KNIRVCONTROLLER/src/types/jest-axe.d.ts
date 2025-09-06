declare module 'jest-axe' {
  export function axe(container: Element, options?: any): Promise<any>;
  export function toHaveNoViolations(): any;
}

declare namespace jest {
  interface Matchers<R> {
    toHaveNoViolations(): R;
  }
}
