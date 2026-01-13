#!/usr/bin/env python3
"""
Script to update slides with responsive design patterns
"""
import os
import re

def update_slide_responsive(slide_path):
    """Update a slide file to use responsive design"""
    
    with open(slide_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # Replace the old body and slide CSS
    old_pattern = r'        body \{[^}]+\}\s*\.slide \{[^}]+\}'
    new_css = '''        html, body {
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

        /* Ensure 16:9 aspect ratio on all screen sizes */
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
        }'''
    
    content = re.sub(old_pattern, new_css, content, flags=re.DOTALL)
    
    # Update header styles
    content = re.sub(
        r'\.header \{\s*margin-bottom: \d+px;\s*\}',
        '.header {\n            margin-bottom: clamp(2vh, 4vh, 6vh);\n        }',
        content
    )
    
    # Update title styles
    content = re.sub(
        r'\.title \{\s*font-size: \d+px;\s*font-weight: 700;\s*margin-bottom: \d+px;\s*\}',
        '.title {\n            font-size: clamp(2.5rem, 4vw, 3.5rem);\n            font-weight: 700;\n            margin-bottom: clamp(0.8rem, 1.5vh, 1.2rem);\n        }',
        content
    )
    
    # Add mobile breakpoints before </style>
    mobile_css = '''
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
        }'''
    
    content = re.sub(r'(\s+)</style>', mobile_css + r'\1</style>', content)
    
    with open(slide_path, 'w', encoding='utf-8') as f:
        f.write(content)
    
    print(f"Updated {slide_path}")

def main():
    """Update slides 8-15"""
    base_path = "docs/business_plan/presenter/presentations/90DayGTMPlanPresentation"
    
    for slide_num in range(8, 16):
        slide_path = os.path.join(base_path, f"Slide_{slide_num}.html")
        if os.path.exists(slide_path):
            update_slide_responsive(slide_path)
        else:
            print(f"Slide {slide_num} not found")

if __name__ == "__main__":
    main()
