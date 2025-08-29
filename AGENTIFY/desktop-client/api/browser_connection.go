package api

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// BrowserConnection implements TargetSystemConnection for browser automation
type BrowserConnection struct {
	target    *TargetSystem
	browser   *rod.Browser
	launcher  *launcher.Launcher
	pages     map[string]*rod.Page
	mutex     sync.RWMutex
	connected bool
	startTime time.Time
}

// NewBrowserConnection creates a new browser connection
func NewBrowserConnection(target *TargetSystem) (TargetSystemConnection, error) {
	return &BrowserConnection{
		target: target,
		pages:  make(map[string]*rod.Page),
	}, nil
}

// Connect establishes the browser connection
func (c *BrowserConnection) Connect(ctx context.Context) error {
	_ = ctx // Context not used in current implementation
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connected {
		return nil
	}

	// Get browser type from config (default to chrome)
	browserType := "chrome"
	if bt, ok := c.target.Config["browserType"].(string); ok {
		browserType = bt
	}

	// Get headless mode from config (default to true)
	headless := true
	if h, ok := c.target.Config["headless"].(bool); ok {
		headless = h
	}

	// Create launcher
	c.launcher = launcher.New()

	// Configure browser based on type
	switch strings.ToLower(browserType) {
	case "chrome", "chromium":
		c.launcher = c.launcher.Bin(findChromeBinary())
	case "firefox":
		return fmt.Errorf("firefox support not yet implemented")
	case "edge":
		c.launcher = c.launcher.Bin(findEdgeBinary())
	default:
		return fmt.Errorf("unsupported browser type: %s", browserType)
	}

	// Configure headless mode
	if headless {
		c.launcher = c.launcher.Headless(true)
	} else {
		c.launcher = c.launcher.Headless(false)
	}

	// Add additional chrome flags for better automation
	c.launcher = c.launcher.Set("disable-blink-features", "AutomationControlled").
		Set("disable-web-security").
		Set("disable-features", "VizDisplayCompositor").
		Set("no-sandbox").
		Set("disable-dev-shm-usage")

	// Launch browser
	url, err := c.launcher.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %v", err)
	}

	// Connect to browser
	c.browser = rod.New().ControlURL(url)
	if err := c.browser.Connect(); err != nil {
		return fmt.Errorf("failed to connect to browser: %v", err)
	}

	c.connected = true
	c.startTime = time.Now()

	return nil
}

// Disconnect closes the browser connection
func (c *BrowserConnection) Disconnect(ctx context.Context) error {
	_ = ctx // Context not used in current implementation
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.connected {
		return nil
	}

	// Close all pages
	for _, page := range c.pages {
		page.Close()
	}
	c.pages = make(map[string]*rod.Page)

	// Close browser
	if c.browser != nil {
		c.browser.Close()
	}

	// Kill launcher process
	if c.launcher != nil {
		c.launcher.Kill()
	}

	c.connected = false
	return nil
}

// IsConnected returns the connection status
func (c *BrowserConnection) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected
}

// GetCapabilities returns available browser capabilities
func (c *BrowserConnection) GetCapabilities() []string {
	return []string{
		"navigate",
		"click",
		"type",
		"screenshot",
		"get_text",
		"get_html",
		"execute_script",
		"wait_for_element",
		"get_cookies",
		"set_cookies",
		"get_page_source",
		"scroll",
		"hover",
		"drag_and_drop",
		"upload_file",
		"download_file",
	}
}

// Execute executes a browser operation
func (c *BrowserConnection) Execute(ctx context.Context, operation string, params map[string]interface{}) (interface{}, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("browser not connected")
	}

	switch operation {
	case "navigate":
		return c.navigate(ctx, params)
	case "click":
		return c.click(ctx, params)
	case "type":
		return c.typeText(ctx, params)
	case "screenshot":
		return c.screenshot(ctx, params)
	case "get_text":
		return c.getText(ctx, params)
	case "get_html":
		return c.getHTML(ctx, params)
	case "execute_script":
		return c.executeScript(ctx, params)
	case "wait_for_element":
		return c.waitForElement(ctx, params)
	case "get_cookies":
		return c.getCookies(ctx, params)
	case "set_cookies":
		return c.setCookies(ctx, params)
	case "get_page_source":
		return c.getPageSource(ctx, params)
	case "scroll":
		return c.scroll(ctx, params)
	case "hover":
		return c.hover(ctx, params)
	case "new_page":
		return c.newPage(ctx, params)
	case "close_page":
		return c.closePage(ctx, params)
	case "list_pages":
		return c.listPages(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// GetStatus returns detailed browser status
func (c *BrowserConnection) GetStatus() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	status := map[string]interface{}{
		"connected": c.connected,
		"type":      "browser",
		"pageCount": len(c.pages),
		"uptime":    time.Since(c.startTime).String(),
	}

	if c.connected && c.browser != nil {
		// Get browser version info
		if version, err := c.browser.Version(); err == nil {
			status["browserVersion"] = version.Product
			status["userAgent"] = version.UserAgent
		}

		// Get page titles
		var pageTitles []string
		for _, page := range c.pages {
			if info, err := page.Info(); err == nil {
				pageTitles = append(pageTitles, info.Title)
			}
		}
		status["pageTitles"] = pageTitles
	}

	return status
}

// GetType returns the target system type
func (c *BrowserConnection) GetType() TargetSystemType {
	return TargetTypeBrowser
}

// Helper functions

// findChromeBinary finds the Chrome/Chromium binary path
func findChromeBinary() string {
	// Common Chrome/Chromium paths
	paths := []string{
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/snap/bin/chromium",
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
		"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
	}

	for _, path := range paths {
		if _, err := exec.LookPath(path); err == nil {
			return path
		}
	}

	// Fallback to system PATH
	if path, err := exec.LookPath("google-chrome"); err == nil {
		return path
	}
	if path, err := exec.LookPath("chromium"); err == nil {
		return path
	}
	if path, err := exec.LookPath("chromium-browser"); err == nil {
		return path
	}

	return "google-chrome" // Default fallback
}

// findEdgeBinary finds the Microsoft Edge binary path
func findEdgeBinary() string {
	paths := []string{
		"/usr/bin/microsoft-edge",
		"/usr/bin/microsoft-edge-stable",
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		"C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe",
		"C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe",
	}

	for _, path := range paths {
		if _, err := exec.LookPath(path); err == nil {
			return path
		}
	}

	if path, err := exec.LookPath("microsoft-edge"); err == nil {
		return path
	}

	return "microsoft-edge" // Default fallback
}

// getOrCreatePage gets an existing page or creates a new one
func (c *BrowserConnection) getOrCreatePage(pageID string) (*rod.Page, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if page, ok := c.pages[pageID]; ok {
		return page, nil
	}

	// Create new page
	page, err := c.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %v", err)
	}

	c.pages[pageID] = page
	return page, nil
}

// Browser operation implementations

// navigate navigates to a URL
func (c *BrowserConnection) navigate(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	url := getStringParam(params, "url", "")
	if url == "" {
		return nil, fmt.Errorf("url parameter is required")
	}

	pageID := getStringParam(params, "pageId", "default")
	page, err := c.getOrCreatePage(pageID)
	if err != nil {
		return nil, err
	}

	if err := page.Navigate(url); err != nil {
		return nil, fmt.Errorf("failed to navigate: %v", err)
	}

	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("failed to wait for page load: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"url":     url,
		"title":   page.MustInfo().Title,
	}, nil
}

// click clicks on an element
func (c *BrowserConnection) click(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	selector := getStringParam(params, "selector", "")
	if selector == "" {
		return nil, fmt.Errorf("selector parameter is required")
	}

	pageID := getStringParam(params, "pageId", "default")
	page, err := c.getOrCreatePage(pageID)
	if err != nil {
		return nil, err
	}

	element, err := page.Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element not found: %v", err)
	}

	if err := element.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("failed to click: %v", err)
	}

	return map[string]interface{}{
		"success": true,
	}, nil
}

// typeText types text into an element
func (c *BrowserConnection) typeText(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	selector := getStringParam(params, "selector", "")
	text := getStringParam(params, "text", "")
	if selector == "" {
		return nil, fmt.Errorf("selector parameter is required")
	}

	pageID := getStringParam(params, "pageId", "default")
	page, err := c.getOrCreatePage(pageID)
	if err != nil {
		return nil, err
	}

	element, err := page.Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element not found: %v", err)
	}

	// Clear existing text if requested
	if getBoolParam(params, "clear", false) {
		if err := element.SelectAllText(); err == nil {
			element.Input("")
		}
	}

	if err := element.Input(text); err != nil {
		return nil, fmt.Errorf("failed to type text: %v", err)
	}

	return map[string]interface{}{
		"success": true,
	}, nil
}

// screenshot takes a screenshot
func (c *BrowserConnection) screenshot(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	pageID := getStringParam(params, "pageId", "default")
	page, err := c.getOrCreatePage(pageID)
	if err != nil {
		return nil, err
	}

	// Take screenshot
	data, err := page.Screenshot(true, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to take screenshot: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"data":    data,
		"format":  "png",
	}, nil
}

// getText gets text content from an element
func (c *BrowserConnection) getText(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	selector := getStringParam(params, "selector", "")
	if selector == "" {
		return nil, fmt.Errorf("selector parameter is required")
	}

	pageID := getStringParam(params, "pageId", "default")
	page, err := c.getOrCreatePage(pageID)
	if err != nil {
		return nil, err
	}

	element, err := page.Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element not found: %v", err)
	}

	text, err := element.Text()
	if err != nil {
		return nil, fmt.Errorf("failed to get text: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"text":    text,
	}, nil
}

// getHTML gets HTML content
func (c *BrowserConnection) getHTML(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	selector := getStringParam(params, "selector", "")
	pageID := getStringParam(params, "pageId", "default")

	page, err := c.getOrCreatePage(pageID)
	if err != nil {
		return nil, err
	}

	var html string
	if selector == "" {
		// Get entire page HTML
		var htmlErr error
		html, htmlErr = page.HTML()
		if htmlErr != nil {
			return nil, fmt.Errorf("failed to get page HTML: %v", htmlErr)
		}
	} else {
		// Get specific element HTML
		element, elementErr := page.Element(selector)
		if elementErr != nil {
			return nil, fmt.Errorf("element not found: %v", elementErr)
		}
		var htmlErr error
		html, htmlErr = element.HTML()
		if htmlErr != nil {
			return nil, fmt.Errorf("failed to get element HTML: %v", htmlErr)
		}
	}

	return map[string]interface{}{
		"success": true,
		"html":    html,
	}, nil
}

// executeScript executes JavaScript
func (c *BrowserConnection) executeScript(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	script := getStringParam(params, "script", "")
	if script == "" {
		return nil, fmt.Errorf("script parameter is required")
	}

	pageID := getStringParam(params, "pageId", "default")
	page, err := c.getOrCreatePage(pageID)
	if err != nil {
		return nil, err
	}

	result, err := page.Eval(script)
	if err != nil {
		return nil, fmt.Errorf("failed to execute script: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"result":  result.Value,
	}, nil
}

// waitForElement waits for an element to appear
func (c *BrowserConnection) waitForElement(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	selector := getStringParam(params, "selector", "")
	if selector == "" {
		return nil, fmt.Errorf("selector parameter is required")
	}

	timeout := getIntParam(params, "timeout", 30) // Default 30 seconds
	pageID := getStringParam(params, "pageId", "default")

	page, err := c.getOrCreatePage(pageID)
	if err != nil {
		return nil, err
	}

	// Wait for element with timeout
	element, err := page.Timeout(time.Duration(timeout) * time.Second).Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element not found within timeout: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"found":   element != nil,
	}, nil
}

// getCookies gets cookies from the page
func (c *BrowserConnection) getCookies(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	pageID := getStringParam(params, "pageId", "default")
	page, err := c.getOrCreatePage(pageID)
	if err != nil {
		return nil, err
	}

	cookies, err := page.Cookies([]string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cookies: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"cookies": cookies,
	}, nil
}

// setCookies sets cookies on the page
func (c *BrowserConnection) setCookies(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	pageID := getStringParam(params, "pageId", "default")
	page, err := c.getOrCreatePage(pageID)
	if err != nil {
		return nil, err
	}

	// Extract cookies from params
	cookiesParam, ok := params["cookies"]
	if !ok {
		return nil, fmt.Errorf("cookies parameter is required")
	}

	// Convert to proper cookie format
	var cookies []*proto.NetworkCookieParam
	if cookieList, ok := cookiesParam.([]interface{}); ok {
		for _, cookieData := range cookieList {
			if cookieMap, ok := cookieData.(map[string]interface{}); ok {
				cookie := &proto.NetworkCookieParam{
					Name:  getStringParam(cookieMap, "name", ""),
					Value: getStringParam(cookieMap, "value", ""),
				}
				if domain := getStringParam(cookieMap, "domain", ""); domain != "" {
					cookie.Domain = domain
				}
				if path := getStringParam(cookieMap, "path", ""); path != "" {
					cookie.Path = path
				}
				cookies = append(cookies, cookie)
			}
		}
	}

	if err := page.SetCookies(cookies); err != nil {
		return nil, fmt.Errorf("failed to set cookies: %v", err)
	}

	return map[string]interface{}{
		"success": true,
	}, nil
}

// getPageSource gets the page source
func (c *BrowserConnection) getPageSource(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	pageID := getStringParam(params, "pageId", "default")
	page, err := c.getOrCreatePage(pageID)
	if err != nil {
		return nil, err
	}

	html, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("failed to get page source: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"source":  html,
	}, nil
}

// scroll scrolls the page or an element
func (c *BrowserConnection) scroll(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	pageID := getStringParam(params, "pageId", "default")
	page, err := c.getOrCreatePage(pageID)
	if err != nil {
		return nil, err
	}

	x := getIntParam(params, "x", 0)
	y := getIntParam(params, "y", 0)
	selector := getStringParam(params, "selector", "")

	if selector != "" {
		// Scroll to element
		element, err := page.Element(selector)
		if err != nil {
			return nil, fmt.Errorf("element not found: %v", err)
		}
		if err := element.ScrollIntoView(); err != nil {
			return nil, fmt.Errorf("failed to scroll to element: %v", err)
		}
	} else {
		// Scroll by coordinates
		if err := page.Mouse.Scroll(float64(x), float64(y), 1); err != nil {
			return nil, fmt.Errorf("failed to scroll: %v", err)
		}
	}

	return map[string]interface{}{
		"success": true,
	}, nil
}

// hover hovers over an element
func (c *BrowserConnection) hover(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	selector := getStringParam(params, "selector", "")
	if selector == "" {
		return nil, fmt.Errorf("selector parameter is required")
	}

	pageID := getStringParam(params, "pageId", "default")
	page, err := c.getOrCreatePage(pageID)
	if err != nil {
		return nil, err
	}

	element, err := page.Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element not found: %v", err)
	}

	if err := element.Hover(); err != nil {
		return nil, fmt.Errorf("failed to hover: %v", err)
	}

	return map[string]interface{}{
		"success": true,
	}, nil
}

// newPage creates a new page
func (c *BrowserConnection) newPage(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	pageID := getStringParam(params, "pageId", fmt.Sprintf("page_%d", time.Now().UnixNano()))

	page, err := c.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %v", err)
	}

	c.mutex.Lock()
	c.pages[pageID] = page
	c.mutex.Unlock()

	return map[string]interface{}{
		"success": true,
		"pageId":  pageID,
	}, nil
}

// closePage closes a page
func (c *BrowserConnection) closePage(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	pageID := getStringParam(params, "pageId", "default")

	c.mutex.Lock()
	page, ok := c.pages[pageID]
	if ok {
		delete(c.pages, pageID)
	}
	c.mutex.Unlock()

	if !ok {
		return nil, fmt.Errorf("page not found: %s", pageID)
	}

	if err := page.Close(); err != nil {
		return nil, fmt.Errorf("failed to close page: %v", err)
	}

	return map[string]interface{}{
		"success": true,
	}, nil
}

// listPages lists all open pages
func (c *BrowserConnection) listPages(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx    // Context not used in current implementation
	_ = params // Parameters not used in current implementation
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var pages []map[string]interface{}
	for pageID, page := range c.pages {
		info := page.MustInfo()
		pages = append(pages, map[string]interface{}{
			"pageId": pageID,
			"title":  info.Title,
			"url":    info.URL,
		})
	}

	return map[string]interface{}{
		"success": true,
		"pages":   pages,
	}, nil
}
