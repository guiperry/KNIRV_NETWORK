# AltGui Frontend - Blockchain Dashboard

This project is a React-based frontend application designed to provide a comprehensive dashboard for interacting with blockchain data and functionalities. It features a modern, visually appealing design with a "glassmorphism" aesthetic, offering various tools and information related to blockchain management, analytics, and more.

## Table of Contents

- [Features](#features)
- [Project Structure](#project-structure)
- [Technologies Used](#technologies-used)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
  - [Running the Application](#running-the-application)
- [Key Components](#key-components)
  - [Dashboard Layouts](#dashboard-layouts)
  - [Pages](#pages)
  - [Other Components](#other-components)
- [Styling](#styling)
- [Contributing](#contributing)
- [License](#license)

## Features

- **Responsive Design:** The application is built to be responsive and adapt to various screen sizes.
- **Glassmorphism UI:** Utilizes a modern "glassmorphism" design with translucent backgrounds and blurred effects.
- **Blockchain Data Display:** Shows key blockchain metrics and data, including asset values, transaction history, and network information.
- **Interactive Sidebar:** A navigable sidebar allows users to switch between different sections of the dashboard.
- **Top Navigation:** A top navigation bar provides quick access to search, notifications, settings, and user profile.
- **Multiple Dashboard Views:** Includes different dashboard layouts (Basic, Simple, Advanced) to cater to various user needs.
- **Wallet Management:** Features for managing blockchain wallets, including sending and receiving assets.
- **Network Management:** Tools for connecting to and managing different blockchain networks.
- **Donation Support:** A section dedicated to supporting blockchain projects through donations.
- **Price Prediction:** Displays cryptocurrency price predictions.
- **Analytics Overview:** Provides a visual overview of blockchain analytics.
- **Search Functionality:** Allows users to search within the blockchain data.

## Project Structure

The project is organized into the following directories:

altgui/
├── src/
│ ├── components/ # Reusable UI components
│ │ ├── BlockchainDashboard.js # Complex dashboard with sidebar and top nav
│ │ ├── BootstrapTest.js # Test for react-bootstrap components
│ │ ├── DashboardLayout.js # Another complex dashboard layout
│ │ ├── MinimalDashboard.js # Very basic dashboard
│ │ ├── NewDashboardLayout.js # Simplified dashboard layout
│ │ ├── SimpleBlockchain.js # Simple dashboard with cards
│ │ ├── SimpleDashboard.js # Another simple dashboard
│ │ └── TestComponent.js # Test component
│ ├── pages/ # Main pages of the application
│ │ ├── \_app.js # Main application wrapper
│ │ ├── advanced.js # Advanced dashboard page
│ │ ├── basic.js # Basic dashboard page
│ │ ├── dashboard.js # Another basic dashboard page
│ │ ├── index.js # Home page
│ │ └── test.js # Test page
│ └── styles/ # Global CSS styles
│ └── globals.css
└── ...

## Technologies Used

- **React:** A JavaScript library for building user interfaces.
- **React Bootstrap:** A UI library that provides pre-built React components styled with Bootstrap.
- **react-icons:** A library for using popular icons in React projects.
- **Next.js:** A React framework for building server-rendered and statically generated web applications.
- **CSS:** For styling and layout.

## Getting Started

### Prerequisites

- **Node.js:** Make sure you have Node.js installed on your machine. You can download it from [https://nodejs.org/](https://nodejs.org/).
- **npm (or yarn):** Node Package Manager (npm) is usually installed with Node.js. Alternatively, you can use yarn.

### Installation

1.  **Clone the repository:**

    ```bash
    git clone <repository-url>
    ```

2.  **Navigate to the project directory:**

    ```bash
    cd sable-station-frontend_v1.2
    ```

3.  **Install dependencies:**

    ```bash
    npm install
    # or
    yarn install
    ```

### Running the Application

1.  **Start the development server:**

    ```bash
    npm run dev
    # or
    yarn dev
    ```

2.  **Open your browser and navigate to `http://localhost:3000`** to view the application.

## Key Components

### Dashboard Layouts

- **`BlockchainDashboard.js`:** A complex dashboard component with a sidebar, top navigation, and various data cards.
- **`DashboardLayout.js`:** Another complex dashboard layout with a sidebar, top navigation, and data cards, using a "glassmorphism" style.
- **`NewDashboardLayout.js`:** A simplified dashboard layout with a sidebar, top navigation, and fewer data cards.
- **`SimpleDashboard.js`:** A basic dashboard layout with a title, description, and two data cards.
- **`SimpleBlockchain.js`:** Similar to `SimpleDashboard.js`, another basic dashboard layout.
- **`MinimalDashboard.js`:** A very basic dashboard with only a title and description.

### Pages

- **`_app.js`:** The main application wrapper that includes global styles.
- **`advanced.js`:** An advanced dashboard page with multiple sections (Dashboard, Wallet, Networks, Donations, Prediction).
- **`basic.js`:** A basic dashboard page with a title, description, and two data cards.
- **`dashboard.js`:** Another basic dashboard page using React Bootstrap components.
- **`index.js`:** The home page of the application, featuring a sidebar, top navigation, and data cards.
- **`test.js`:** A simple test page to verify routing.

### Other Components

- **`BootstrapTest.js`:** A component to test individual React Bootstrap component imports.
- **`TestComponent.js`:** A simple test component to verify imports.

## Styling

- **`globals.css`:** Contains global CSS styles applied throughout the application.
- **Inline Styles:** Many components use inline styles for quick adjustments and the "glassmorphism" effect.
- **React Bootstrap:** Used for pre-built components and styling.
- **Glassmorphism:** The application uses a glassmorphism style, which is achieved by using translucent backgrounds, blur effects, and subtle shadows.

## Contributing

Contributions are welcome! If you'd like to contribute to this project, please follow these steps:

1.  Fork the repository.
2.  Create a new branch for your feature or bug fix.
3.  Make your changes and commit them.
4.  Push your changes to your forked repository.
5.  Submit a pull request.

## License

This project is licensed under the MIT License.
