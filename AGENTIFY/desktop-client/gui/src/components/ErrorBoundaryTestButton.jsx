import React, { useState } from 'react';

// Component that can trigger rendering errors for testing
const ErrorBoundaryTestButton = () => {
  const [shouldError, setShouldError] = useState(false);

  const triggerRenderingError = () => {
    console.log('Triggering rendering error for ErrorBoundary test...');
    setShouldError(true);
  };

  // This will cause a rendering error when shouldError is true
  if (shouldError) {
    // Simulate a common rendering error - trying to access property of undefined
    const undefinedObject = undefined;
    return <div>{undefinedObject.nonExistentProperty.map(item => item.name)}</div>;
  }

  return (
    <button
      onClick={triggerRenderingError}
      className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg transition-colors duration-200 text-sm"
      title="Test ErrorBoundary functionality"
    >
      🧪 Test Rendering Error
    </button>
  );
};

export default ErrorBoundaryTestButton;
