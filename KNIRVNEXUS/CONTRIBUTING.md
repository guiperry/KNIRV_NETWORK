# Contributing to Agentic Engine

First off, thank you for considering contributing to the Agentic Engine! We welcome any contributions that help us improve and grow this platform. Whether it's reporting a bug, discussing improvements, or contributing code, your help is appreciated.

This document provides guidelines for contributing to the project. Please read it carefully to ensure a smooth and effective collaboration.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Enhancements](#suggesting-enhancements)
  - [Your First Code Contribution](#your-first-code-contribution)
  - [Pull Requests](#pull-requests)
- [Development Setup](#development-setup)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
- [Coding Guidelines](#coding-guidelines)
  - [Go (Backend)](#go-backend)
  - [React/TypeScript (Frontend)](#reacttypescript-frontend)
  - [Commit Messages](#commit-messages)
- [Testing](#testing)
- [Community](#community)

## Code of Conduct

This project and everyone participating in it is governed by our [Code of Conduct](CODE_OF_CONDUCT.md) (You'll need to create this file, e.g., based on the Contributor Covenant). By participating, you are expected to uphold this code. Please report unacceptable behavior.

## How Can I Contribute?

### Reporting Bugs

If you find a bug, please ensure the bug was not already reported by searching on GitHub under [Issues](https://github.com/Inc-liner/Agentic-Engine/issues).

If you're unable to find an open issue addressing the problem, [open a new one](https://github.com/inc-liner/Agentic-Engine/issues/new). Be sure to include a **title and clear description**, as much relevant information as possible, and a **code sample or an executable test case** demonstrating the expected behavior that is not occurring.

Provide information about your environment:
- Operating System
- Go version
- Node.js version
- Browser version (if frontend related)
- Agentic Engine version/commit

### Suggesting Enhancements

If you have an idea for an enhancement, please first discuss the change you wish to make via a GitHub issue, or any other method with the owners of this repository before making a change.

When suggesting an enhancement:
- Explain the problem you're trying to solve.
- Describe the enhancement and its benefits.
- Provide examples of how it might be used.
- Consider potential drawbacks or trade-offs.

Refer to the [Frontend Implementation Plan](./docs/frontend_implementation_plan.md) and [Merge Implementation Plan](./docs/merge_implementation_plan.md) for an overview of planned features and architecture.

### Your First Code Contribution

Unsure where to begin contributing? You can start by looking through `good first issue` or `help wanted` issues:

*   Good first issues - issues which should only require a few lines of code, and a test or two.
*   Help wanted issues - issues which should be a bit more involved than `good first issues`.

### Pull Requests

1.  **Fork the repository**: Click the "Fork" button on the top right of the Agentic Engine GitHub page.
2.  **Clone your fork**: `git clone https://github.com/inc-liner/Agentic-Engine.git`
3.  **Create a new branch**: `git checkout -b feature/your-feature-name` or `fix/your-bug-fix-name`.
4.  **Make your changes**: Implement your feature or bug fix.
5.  **Commit your changes**: Use clear and descriptive commit messages (see Commit Messages).
6.  **Push to your branch**: `git push origin feature/your-feature-name`.
7.  **Open a Pull Request (PR)**: Go to the original repository and click "New pull request".
    *   Ensure the PR description clearly describes the problem and solution. Include the relevant issue number if applicable.
    *   Ensure your PR passes all CI checks (if configured).
    *   Be prepared to discuss your changes and make adjustments if requested.

## Development Setup

Please refer to the README.md for detailed instructions on setting up the development environment.

### Prerequisites

*   **Go:** Version 1.21 or later.
*   **Node.js and npm/yarn:** Latest LTS version recommended.
*   **AI Provider Accounts:** (Optional, for full feature testing) Cerebras, Gemini, DeepSeek.

### Installation

Briefly:
1.  Clone the repository.
2.  Set up backend API keys in a `.env` file in the backend's root directory.
3.  Build the Go backend (`go build -o agentic-engine-backend .` from the backend directory).
4.  Install frontend dependencies (`npm install` or `yarn install` from the `gui` directory).

To run the application:
1.  Start the backend server.
2.  Start the frontend development server (`npm run dev` or `yarn dev` from the `gui` directory).

## Coding Guidelines

### Go (Backend)

*   Follow standard Go formatting (`gofmt` or `goimports`).
*   Write clear, concise, and well-commented code.
*   Adhere to effective Go principles (e.g., error handling, interfaces).
*   Write unit tests for new functionality.
*   Be mindful of concurrency and use mutexes appropriately. Refer to the Mutex Hierarchy document for existing patterns.
*   When adding new API endpoints, ensure they are documented and follow RESTful principles.

### React/TypeScript (Frontend)

*   Follow standard TypeScript and React best practices.
*   Use functional components with Hooks where possible.
*   Maintain a clear component structure.
*   Use Tailwind CSS for styling.
*   Ensure components are responsive and accessible.
*   Write tests for components and logic (e.g., using Jest and React Testing Library).

### Commit Messages

*   Use a clear and concise subject line (e.g., 50 characters max).
*   Separate subject from body with a blank line.
*   Explain what and why in the commit message body, not how.
*   Use the imperative mood (e.g., "Add feature" not "Added feature" or "Adds feature").
*   Reference issue numbers if applicable (e.g., `Fixes #123`).

Example:
```
feat: Add agent creation modal

Implements the agent creation flow as per issue #42.
Includes form validation and API integration for POST /api/v1/agents.
```

## Testing

*   **Backend**: Write unit tests for Go packages. Aim for good test coverage.
    ```bash
    go test ./...
    ```
*   **Frontend**: Write unit and integration tests for React components.
    ```bash
    # In the gui directory
    npm test
    # or
    yarn test
    ```
*   **End-to-End (E2E)**: Manual E2E testing is crucial. Ensure your changes work as expected in the browser and integrate correctly with the backend.

## Community

If you have questions or want to discuss ideas, please use the GitHub Issues page.

Thank you for contributing!

