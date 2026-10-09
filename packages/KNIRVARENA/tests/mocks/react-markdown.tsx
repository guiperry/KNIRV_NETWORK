import React from 'react';

// react-markdown ships ESM only; components under test just need its children.
const ReactMarkdown = ({ children }: { children?: React.ReactNode }) => <div data-testid="react-markdown">{children}</div>;

export default ReactMarkdown;
