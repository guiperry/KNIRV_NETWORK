# KNIRV Presenter System

A responsive presentation system for KNIRV presentations with password protection and mobile-friendly design.

## Overview

The KNIRV Presenter System allows you to create, manage, and present slide-based presentations with:
- Password-protected access
- Responsive design that adapts to different screen sizes
- Mobile-friendly interface with proper aspect ratio handling
- Automatic slide detection and navigation
- Glass morphism styling consistent with KNIRV branding

## Directory Structure

```
KNIRVGATEWAY/presenter/
├── README.md                    # This file
├── index.html                   # Main presentation selector
├── slide-viewer.html           # Individual slide viewer
├── presentation-slider.html    # Alternative presentation interface
├── server.js                   # Express server for development
├── update_slides.py            # Script to update slide styling
├── package.json                # Node.js dependencies
└── presentations/              # Presentation directories
    ├── 90DayGTMPlanPresentation/
    │   ├── auth.json           # Authentication and metadata
    │   ├── Slide_1.html        # Individual slide files
    │   ├── Slide_2.html
    │   └── ...
    ├── InvestorB-PlanPresentation/
    │   ├── auth.json
    │   ├── Slide_1.html
    │   └── ...
    └── Use-Cases/
        ├── Slide_1.html
        └── ...
```

## Setting Up a New Presentation

### 1. Create Presentation Directory

Create a new directory under `presentations/` with your presentation name:

```bash
mkdir presentations/YourPresentationName
```

### 2. Create auth.json File

Each presentation directory must contain an `auth.json` file with the following structure:

```json
{
  "presentationName": "Your Presentation Title",
  "password": "your_secure_password",
  "description": "Brief description of your presentation",
  "lastUpdated": "2024-08-09"
}
```

**Required Fields:**
- `presentationName`: Display name shown in the presentation selector
- `password`: Password required to access the presentation
- `description`: Brief description shown on the presentation card
- `lastUpdated`: Date of last modification (YYYY-MM-DD format)

### 3. Add Slide Files

Create individual HTML slide files named `Slide_1.html`, `Slide_2.html`, etc.

**Slide File Template:**
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Slide Title</title>
    <style>
        html, body {
            width: 100%;
            height: 100%;
            font-family: 'Source Sans Pro', sans-serif;
            overflow: hidden;
            margin: 0;
            padding: 0;
        }
        .slide {
            width: 100vw;
            height: 100vh;
            aspect-ratio: 16/9;
            max-width: 100vw;
            max-height: 100vh;
            position: relative;
            background: linear-gradient(135deg, rgba(25, 25, 112, 0.9), rgba(102, 51, 153, 0.9));
            color: white;
            display: flex;
            flex-direction: column;
            padding: 4vh 5vw;
            overflow: hidden;
        }

        /* Responsive aspect ratio handling */
        @media (max-aspect-ratio: 16/9) {
            .slide {
                width: calc(100vh * 16/9);
                height: 100vh;
                margin: 0 auto;
            }
        }

        @media (min-aspect-ratio: 16/9) {
            .slide {
                width: 100vw;
                height: calc(100vw * 9/16);
                margin: auto 0;
            }
        }

        .header {
            margin-bottom: clamp(2vh, 4vh, 6vh);
        }

        .title {
            font-size: clamp(2.5rem, 4vw, 3.5rem);
            font-weight: 700;
            margin-bottom: clamp(0.8rem, 1.5vh, 1.2rem);
        }

        /* Mobile responsive breakpoints */
        @media (max-width: 768px) {
            .slide {
                padding: 3vh 4vw;
            }
            .content {
                flex-direction: column;
                gap: clamp(1.5rem, 3vh, 2rem);
            }
        }

        @media (max-width: 480px) {
            .slide {
                padding: 2vh 3vw;
            }
            .title {
                font-size: clamp(1.8rem, 6vw, 2.2rem);
            }
        }
    </style>
</head>
<body>
    <div class="slide">
        <div class="header">
            <h1 class="title">Your Slide Title</h1>
        </div>
        <div class="content">
            <!-- Your slide content here -->
        </div>
    </div>
</body>
</html>
```

## Using update_slides.py

The `update_slides.py` script helps update existing slides with responsive design patterns.

### Current Functionality

The script currently updates slides 8-15 in the 90DayGTMPlanPresentation with:
- Responsive CSS for mobile devices
- Proper aspect ratio handling (16:9)
- Clamp-based font sizing for better readability
- Mobile breakpoints for different screen sizes

### Customizing the Script

To use the script for different presentations or slide ranges:

1. **Edit the base path** (line 103):
```python
base_path = "presentations/YourPresentationName"
```

2. **Change the slide range** (line 105):
```python
for slide_num in range(1, 11):  # Updates slides 1-10
```

3. **Run the script**:
```bash
python3 update_slides.py
```

### What the Script Updates

- **Responsive body and slide CSS**: Ensures proper viewport handling
- **Header styles**: Uses clamp() for responsive margins
- **Title styles**: Responsive font sizing with clamp()
- **Mobile breakpoints**: Adds CSS for tablets and phones

## Running the Presentation System

### Development Server (Local Only)

1. **Install dependencies**:
```bash
npm install
```

2. **Start the server**:
```bash
node server.js
```

3. **Access the system**:
   - Open `http://localhost:3000` in your browser
   - Select a presentation from the grid
   - Enter the password when prompted
   - Navigate through slides using arrow keys or navigation buttons

### Production Deployment (KNIRVGATEWAY)

The system is deployed as part of the KNIRVGATEWAY static site on Netlify:

- **Automatic deployment**: Presentations are served as static files
- **Manual setup required**: See `MANUAL_SETUP.md` for adding presentations
- **No server needed**: Runs entirely as static HTML/CSS/JS
- **Netlify integration**: Uses Netlify's static hosting capabilities

## Security Notes

- Passwords are stored in plain text in `auth.json` files
- Import functionality is protected by a separate password in `import-config.json`
- This system is designed for internal presentations, not public-facing content
- Consider implementing proper authentication for production use
- The password protection is client-side only and should not be relied upon for sensitive content

### Import Password Configuration

The import functionality is protected by a password stored in `import-config.json`:

```json
{
  "importPassword": "your_secure_import_password",
  "description": "Password required to access the presentation import functionality",
  "lastUpdated": "2024-08-09"
}
```

**To change the import password:**
1. Edit the `importPassword` field in `import-config.json`
2. Update the `lastUpdated` field
3. Save and deploy the changes

## Troubleshooting

### Presentation Not Showing
- Ensure `auth.json` exists in the presentation directory
- Check that slide files are named correctly (`Slide_1.html`, `Slide_2.html`, etc.)
- Verify the presentation directory name matches what's expected

### Styling Issues
- Run `update_slides.py` to apply responsive design patterns
- Check that CSS is properly formatted in slide files
- Ensure viewport meta tag is present in slide HTML

### Password Issues
- Verify password in `auth.json` matches what you're entering
- Check for extra spaces or special characters
- Passwords are case-sensitive

## Contributing

When adding new presentations:
1. Follow the directory structure outlined above
2. Use the provided slide template for consistency
3. Test on multiple screen sizes and devices
4. Update this README if adding new features
