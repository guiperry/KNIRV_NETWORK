// src/components/ui/stepper.tsx
'use client';

import React from 'react';

interface StepperProps {
  steps: string[];
  current: number; // 0-based
}

const Stepper = ({ steps, current }: StepperProps) => {
  return (
    <div className="w-full">
      <ol className="flex items-center justify-between w-full">
        {steps.map((step, idx) => {
          const isActive = idx === current;
          const isCompleted = idx < current;
          return (
            <li key={step} className="flex-1">
              <div className="flex flex-col items-center">
                <div className={`rounded-full w-8 h-8 flex items-center justify-center mb-2 ${isCompleted ? 'bg-green-600 text-white' : isActive ? 'bg-primary text-white' : 'bg-gray-200 text-gray-700'}`}>
                  {isCompleted ? '✓' : idx + 1}
                </div>
                <div className={`text-xs text-center ${isActive ? 'font-semibold text-primary' : 'text-muted-foreground'}`}>{step}</div>
              </div>
            </li>
          );
        })}
      </ol>
    </div>
  );
};

export default Stepper;
