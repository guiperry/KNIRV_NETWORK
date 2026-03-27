# KNIRV Footer Functionality Test Suite

This document explains how to test the KNIRV Universal Footer across different applications and contexts.

## Overview

The footer test suite validates:
- ✅ Configuration loading and merging
- ✅ Footer initialization and rendering
- ✅ Link resolution across different path contexts
- ✅ Cross-application compatibility
- ✅ Responsive design functionality
- ✅ Error handling and fallbacks

## Quick Start

### Option 1: Browser Console Testing

1. Open the KNIRV website in your browser
2. Open Developer Tools (F12)
3. Go to the Console tab
4. Run the following commands:

```javascript
// Load the test suite
const script = document.createElement('script');
script.src = 'js/test-footer.js';
document.head.appendChild(script);

// Wait for it to load, then run tests
setTimeout(() => {
    window.runFooterTests().then(results => {
        console.log('Test Results:', results);
    });
}, 1000);
```

### Option 2: Enable Auto-Testing

Uncomment the test script in `index.html`:

```html
<!-- Change this line: -->
<!-- <script src="js/test-footer.js"></script> -->

<!-- To this: -->
<script src="js/test-footer.js"></script>
```

The tests will run automatically when the page loads.

## Test Categories

### 1. Configuration Loading Tests
- Verifies unified configuration system works
- Tests oracle config integration
- Validates configuration structure

### 2. Footer Initialization Tests
- Checks if footer element is created
- Validates footer content structure
- Ensures required sections are present

### 3. Link Resolution Tests
- Tests navigation links (main site, docs, explorer)
- Validates footer links (social, legal, resources)
- Checks external service links

### 4. Path Resolution Tests
- Tests absolute URLs (https://...)
- Validates relative paths (documentation/...)
- Checks root paths (/path/...)

### 5. Cross-Application Compatibility Tests
- Tests network-website context paths
- Validates root-level application paths
- Ensures consistent behavior across contexts

### 6. Responsive Design Tests
- Validates responsive CSS rules exist
- Tests viewport compatibility
- Checks mobile-friendly structure

### 7. Error Handling Tests
- Tests invalid configuration paths
- Validates fallback behavior
- Ensures graceful error recovery

## Expected Test Results

### ✅ Passing Tests (Expected)
- Configuration Loading: Should pass if config files exist
- Footer Initialization: Should pass if footer renders
- Link Resolution: Should pass for valid configuration paths
- Path Resolution: Should pass for properly formatted URLs
- Error Handling: Should pass for graceful fallbacks

### ⚠️ Expected Failures (Known Issues)
- Some external service links may fail if services are down
- Legacy documentation paths may fail during migration
- Network-dependent features may fail in offline mode

## Manual Testing Checklist

### Visual Inspection
- [ ] Footer appears at bottom of page
- [ ] All footer sections are visible
- [ ] Links are properly styled
- [ ] Footer is responsive on mobile

### Link Testing
- [ ] Main site link works
- [ ] Documentation links work
- [ ] Social media links open correctly
- [ ] Legal links point to correct documents

### Cross-Application Testing
- [ ] Test footer in network-website context
- [ ] Test footer in root-level applications
- [ ] Verify consistent behavior

## Troubleshooting

### Common Issues

1. **Configuration Not Loading**
   - Check if `portal-links.yaml` exists
   - Verify YAML syntax is correct
   - Check browser console for errors

2. **Links Returning '#'**
   - Verify configuration paths are correct
   - Check if target files/pages exist
   - Ensure path resolution is working

3. **Footer Not Appearing**
   - Check if JavaScript is enabled
   - Verify no JavaScript errors in console
   - Ensure DOM is ready before footer loads

### Debug Commands

```javascript
// Check configuration
console.log('Config loaded:', window.knirvConfig?.isLoaded);
console.log('Config data:', window.knirvConfig?.config);

// Check footer
console.log('Footer element:', document.querySelector('footer.knirv-footer'));
console.log('Footer content:', document.querySelector('footer.knirv-footer')?.innerHTML);

// Test specific links
const footer = window.knirvFooter;
console.log('Main site link:', footer.getLink('navigation.main_site'));
console.log('GitHub link:', footer.getLink('footer.social.github'));
```

## Integration Testing

### Testing Across Applications

1. **Network Website Context** (`/network-website/`)
   - Test relative path resolution
   - Verify application-specific links
   - Check cross-application navigation

2. **Root Level Context** (`/`)
   - Test absolute path resolution
   - Verify root-level application links
   - Check oracle functionality

### Performance Testing

- Monitor footer load time
- Check for layout shifts
- Verify smooth animations
- Test on slow networks

## Continuous Integration

For automated testing, the test suite can be integrated with:

- **Jest/Puppeteer**: For headless browser testing
- **Cypress**: For end-to-end testing
- **Playwright**: For cross-browser testing

## Reporting Issues

When reporting footer issues, please include:

1. **Test Results**: Run `window.runFooterTests()` and include output
2. **Browser Info**: Browser name, version, and context
3. **Configuration**: Current configuration state
4. **Screenshots**: Visual evidence of issues
5. **Steps to Reproduce**: Exact steps to trigger the issue

## Maintenance

The test suite should be updated when:
- New footer sections are added
- Configuration structure changes
- New applications are integrated
- Path resolution logic is modified

---

*Last Updated: September 2025*
*Test Suite Version: 1.0.0*
