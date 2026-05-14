// Global test type declarations
/// <reference types="jest" />
import '@testing-library/jest-dom';

import '@testing-library/jest-dom';

// Extend Jest matchers with jest-dom types
declare global {
  namespace jest {
    interface Matchers<R> {
      // Jest DOM matchers
      toBeInTheDocument(): R;
      toHaveClass(...classNames: string[]): R;
      toHaveAttribute(attr: string, value?: string): R;
      toHaveTextContent(text: string | RegExp): R;
      toBeVisible(): R;
      toBeDisabled(): R;
      toBeEnabled(): R;
      toHaveValue(value: string | number): R;
      toHaveDisplayValue(value: string | string[]): R;
      toBeChecked(): R;
      toHaveFocus(): R;
      toHaveStyle(css: string | Record<string, string | number>): R;
      toBeEmptyDOMElement(): R;
      toBeInvalid(): R;
      toBeRequired(): R;
      toBeValid(): R;
      toContainElement(element: HTMLElement | null): R;
      toContainHTML(htmlText: string): R;
      toHaveAccessibleDescription(expectedAccessibleDescription?: string | RegExp): R;
      toHaveAccessibleName(expectedAccessibleName?: string | RegExp): R;
      toHaveDescription(expectedDescription?: string | RegExp): R;
      toHaveErrorMessage(expectedErrorMessage?: string | RegExp): R;
      toHaveFormValues(expectedValues: Record<string, string | number | boolean>): R;
      toBePartiallyChecked(): R;
    }
  }
}

export {};
