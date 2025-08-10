# KNIRVCORTEX User Guide and Troubleshooting

The KNIRVCORTEX is a mobile-native application that brings autonomous AI capabilities to your existing assistants. It acts as your primary interface to the knirv.com D-TEN platform, allowing you to control various functions using voice commands.

## Getting Started

### Prerequisites

* A modern web browser with Web Speech API support (most modern browsers are compatible).
* A microphone enabled and accessible to your browser.
* Node.js 20+ (only needed if you want to build the application from source).

### Installation (for developers only)

If you are a developer and want to build the application from source, follow these steps:

1. Clone the KNIRVCORTEX repository.
2. Navigate to the project directory in your terminal.
3. Run `npm install` to install the necessary dependencies.
4. Run `npm run dev` to start the development server.

To use the application, simply open `test-voice-integration.html` in your browser.

### Using the KNIRVCORTEX

1. **Enable Voice:** Click the microphone button (usually located in the bottom-right corner).
2. **Speak Commands:** Use the voice commands listed below.
3. **Visual Feedback:** Observe the screen edges change color to indicate the application's status (see "Visual Feedback System" section below).
4. **Cognitive Mode:** Toggle advanced voice processing features (if available) using a voice command.

### Troubleshooting Tips

Before reaching out to support, try the following steps:

* Ensure your microphone is enabled and accessible to your browser.
* Check browser permissions and grant necessary permissions.
* Speak clearly and use the correct voice commands.
* Check for error messages displayed on the screen.

## Voice Commands

The KNIRVCORTEX supports a variety of voice commands categorized as follows:

### Navigation

* "Show skills page" / "Navigate to skills"
* "Open wallet" / "Navigate to wallet"
* "Show UDC" / "Open certificate panel"
* "Go home" / "Show agents"

### System

* "Toggle cognitive mode" / "Enable advanced mode"
* "Check agent status" / "Show agent health"
* "Check NRN balance" / "Show balance"
* "Show network status" / "Check connections"

### Skills

* "Activate skill [skill name]" / "Enable skill [skill name]"
* "Deactivate skill [skill name]" / "Disable skill [skill name]"
* "Show available skills"

## Visual Feedback System

The KNIRVCORTEX uses dynamic edge coloring to provide visual feedback on its status:

* **🟢 Green:** Idle state
* **🔵 Teal:** Listening for commands
* **🔵 Blue:** Processing voice input
* **🟣 Purple:** Speaking/responding
* **🔴 Red:** Error state

The brightness of the edge also reflects the activity level.  Color transitions are smooth and animated.

## Advanced Troubleshooting

If you are experiencing issues with the application, try the following steps:

* Check the browser console for error messages.
* Ensure that the application is running in a compatible browser.
* Try restarting the application or refreshing the page.
* If the issue persists, contact support with the following information:
	+ Browser version and type.
	+ Operating system and version.
	+ A detailed description of the issue.

## Future Enhancements

Future updates will include: multi-language support, custom wake words, voice biometrics, offline processing, advanced NLP, and voice shortcuts.

Improvements Needed:

* Add more detailed information on voice command syntax and usage.
* Provide examples of voice commands for each category.
* Include screenshots or diagrams to illustrate the Visual Feedback System.
* Consider adding a FAQ section to address common questions and issues.
* Update the "Future Enhancements" section to include more specific details and timelines.

<div class="footer-links">
<a href="#/legal/CODE_OF_CONDUCT.md" class="footer-link">Contributor Covenant Code of Conduct</a> | <a href="#/legal/PRIVACY_POLICY.md" class="footer-link">PRIVACY_POLICY.md</a> | <a href="#/legal/TERMS_AND_CONDITIONS.md" class="footer-link">TERMS AND CONDITIONS</a>

© 2025 KNIRV Network
</div>
