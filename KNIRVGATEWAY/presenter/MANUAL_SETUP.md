# Manual Presentation Setup Guide

Since the KNIRV Presenter runs as a static site on Netlify, the automated import functionality is not available in the deployed version. Here's how to manually add presentations:

> **Note:** The import functionality is password-protected. The import password is stored in `import-config.json` in the presenter directory.

## 📁 Directory Structure

Create your presentation folder in the `presentations/` directory:

```
presentations/
├── YourPresentationName/
│   ├── auth.json          # Required: Authentication and metadata
│   ├── Slide_1.html       # Required: First slide
│   ├── Slide_2.html       # Additional slides
│   ├── Slide_3.html
│   └── ...
```

## 🔐 Creating auth.json

Each presentation folder must contain an `auth.json` file:

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

## 🎨 Slide Template

Use this responsive template for your slides:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Your Slide Title</title>
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

## 🚀 Deployment

1. **Add your presentation folder** to the `presentations/` directory
2. **Commit and push** to your repository
3. **Netlify will automatically deploy** the changes
4. **Your presentation will appear** on the main page automatically

## 🔗 Sharing

Once deployed, you can share presentations using:
- **Direct links**: `https://yoursite.netlify.app/share.html?p=PresentationFolderName`
- **Share buttons**: Click the share button (🔗) on any presentation card

## 📝 Tips

- **File naming**: Use `Slide_1.html`, `Slide_2.html`, etc.
- **Folder naming**: Use PascalCase without spaces (e.g., `MyPresentation`)
- **Responsive design**: The template includes mobile-friendly CSS
- **KNIRV branding**: Consistent with the KNIRV design system

## 🔧 Local Development

For local development with the Express server:
```bash
cd presenter/
npm install
node server.js
```

Then visit `http://localhost:3000` to test your presentations locally before deployment.
