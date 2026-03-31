import { test, expect, Page } from '@playwright/test'

test.describe('KNIRVANA Game Interactions', () => {
  let page: Page

  test.beforeEach(async ({ page: testPage }) => {
    page = testPage
    await page.goto('/')
    
    // Wait for the game to load
    await page.waitForSelector('[data-testid="knirvana-game"]', { timeout: 30000 })
    
    // Wait for WebGL context to initialize
    await page.waitForTimeout(2000)
  })

  test('should load the game interface successfully', async () => {
    // Check that the main game container is present
    await expect(page.locator('[data-testid="knirvana-game"]')).toBeVisible()
    
    // Check that the canvas element is present (for Three.js)
    const canvas = page.locator('canvas')
    await expect(canvas).toBeVisible()
    
    // Verify the canvas has proper dimensions
    const canvasBox = await canvas.boundingBox()
    expect(canvasBox?.width).toBeGreaterThan(0)
    expect(canvasBox?.height).toBeGreaterThan(0)
  })

  test('should respond to keyboard controls', async () => {
    // Focus on the game area
    await page.click('[data-testid="knirvana-game"]')
    
    // Test movement controls
    await page.keyboard.press('KeyW') // Forward
    await page.waitForTimeout(100)
    
    await page.keyboard.press('KeyS') // Backward
    await page.waitForTimeout(100)
    
    await page.keyboard.press('KeyA') // Left
    await page.waitForTimeout(100)
    
    await page.keyboard.press('KeyD') // Right
    await page.waitForTimeout(100)
    
    // Test action controls
    await page.keyboard.press('Space') // Select
    await page.waitForTimeout(100)
    
    await page.keyboard.press('KeyR') // Deploy
    await page.waitForTimeout(100)
    
    // Verify the game is still responsive
    await expect(page.locator('[data-testid="knirvana-game"]')).toBeVisible()
  })

  test('should handle mouse interactions', async () => {
    const gameArea = page.locator('[data-testid="knirvana-game"]')
    
    // Get the game area bounds
    const gameBox = await gameArea.boundingBox()
    expect(gameBox).toBeTruthy()
    
    if (gameBox) {
      // Click in the center of the game area
      await page.mouse.click(
        gameBox.x + gameBox.width / 2,
        gameBox.y + gameBox.height / 2
      )
      
      // Test mouse movement
      await page.mouse.move(
        gameBox.x + gameBox.width * 0.3,
        gameBox.y + gameBox.height * 0.3
      )
      
      // Test right-click
      await page.mouse.click(
        gameBox.x + gameBox.width * 0.7,
        gameBox.y + gameBox.height * 0.7,
        { button: 'right' }
      )
      
      // Test drag operation
      await page.mouse.move(
        gameBox.x + gameBox.width * 0.2,
        gameBox.y + gameBox.height * 0.2
      )
      await page.mouse.down()
      await page.mouse.move(
        gameBox.x + gameBox.width * 0.8,
        gameBox.y + gameBox.height * 0.8
      )
      await page.mouse.up()
    }
    
    // Verify the game is still responsive after interactions
    await expect(gameArea).toBeVisible()
  })

  test('should handle window resize gracefully', async () => {
    // Get initial canvas size
    const canvas = page.locator('canvas')
    const initialBox = await canvas.boundingBox()
    
    // Resize the window
    await page.setViewportSize({ width: 1200, height: 800 })
    await page.waitForTimeout(500)
    
    // Check that canvas adjusted to new size
    const resizedBox = await canvas.boundingBox()
    expect(resizedBox?.width).not.toBe(initialBox?.width)
    expect(resizedBox?.height).not.toBe(initialBox?.height)
    
    // Resize to mobile dimensions
    await page.setViewportSize({ width: 375, height: 667 })
    await page.waitForTimeout(500)
    
    // Verify game is still functional on mobile
    await expect(canvas).toBeVisible()
    
    // Restore original size
    await page.setViewportSize({ width: 1920, height: 1080 })
    await page.waitForTimeout(500)
  })

  test('should maintain performance during gameplay', async () => {
    // Start performance monitoring
    await page.evaluate(() => {
      (window as any).performanceMetrics = {
        frameCount: 0,
        startTime: performance.now(),
        frameRates: []
      }
      
      const measureFrameRate = () => {
        (window as any).performanceMetrics.frameCount++
        requestAnimationFrame(measureFrameRate)
      }
      requestAnimationFrame(measureFrameRate)
    })
    
    // Simulate gameplay for 5 seconds
    const gameArea = page.locator('[data-testid="knirvana-game"]')
    await gameArea.click()
    
    // Perform various actions
    for (let i = 0; i < 50; i++) {
      await page.keyboard.press('KeyW')
      await page.waitForTimeout(50)
      await page.keyboard.press('KeyA')
      await page.waitForTimeout(50)
    }
    
    // Check performance metrics
    const metrics = await page.evaluate(() => {
      const m = (window as any).performanceMetrics
      const duration = (performance.now() - m.startTime) / 1000
      const avgFrameRate = m.frameCount / duration
      return { frameCount: m.frameCount, duration, avgFrameRate }
    })
    
    // Verify reasonable performance (at least 30 FPS)
    expect(metrics.avgFrameRate).toBeGreaterThan(30)
    expect(metrics.frameCount).toBeGreaterThan(150) // 5 seconds * 30 FPS
  })

  test('should handle error states gracefully', async () => {
    // Simulate network error by intercepting requests
    await page.route('**/api/**', route => {
      route.abort('failed')
    })
    
    // Try to perform an action that would require network
    await page.keyboard.press('KeyR') // Deploy action
    await page.waitForTimeout(1000)
    
    // Game should still be responsive despite network errors
    await expect(page.locator('[data-testid="knirvana-game"]')).toBeVisible()
    
    // Clear the route interception
    await page.unroute('**/api/**')
  })

  test('should support fullscreen mode', async () => {
    // Check if fullscreen is supported
    const supportsFullscreen = await page.evaluate(() => {
      return 'requestFullscreen' in document.documentElement
    })
    
    if (supportsFullscreen) {
      // Trigger fullscreen (this would typically be done via a button)
      await page.evaluate(() => {
        document.documentElement.requestFullscreen()
      })
      
      await page.waitForTimeout(1000)
      
      // Verify fullscreen state
      const isFullscreen = await page.evaluate(() => {
        return document.fullscreenElement !== null
      })
      
      if (isFullscreen) {
        // Exit fullscreen
        await page.evaluate(() => {
          document.exitFullscreen()
        })
        
        await page.waitForTimeout(1000)
      }
    }
    
    // Game should remain functional regardless of fullscreen support
    await expect(page.locator('[data-testid="knirvana-game"]')).toBeVisible()
  })

  test('should handle touch interactions on mobile', async () => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 })
    
    const gameArea = page.locator('[data-testid="knirvana-game"]')
    const gameBox = await gameArea.boundingBox()
    
    if (gameBox) {
      // Simulate touch tap
      await page.touchscreen.tap(
        gameBox.x + gameBox.width / 2,
        gameBox.y + gameBox.height / 2
      )
      
      // Simulate swipe gesture
      await page.touchscreen.tap(
        gameBox.x + gameBox.width * 0.2,
        gameBox.y + gameBox.height * 0.2
      )
      
      // Simulate pinch gesture (zoom)
      await page.evaluate(() => {
        const canvas = document.querySelector('canvas')
        if (canvas) {
          const touchStart = new TouchEvent('touchstart', {
            touches: [
              new Touch({
                identifier: 0,
                target: canvas,
                clientX: 100,
                clientY: 100
              }),
              new Touch({
                identifier: 1,
                target: canvas,
                clientX: 200,
                clientY: 200
              })
            ]
          })
          canvas.dispatchEvent(touchStart)
          
          setTimeout(() => {
            const touchMove = new TouchEvent('touchmove', {
              touches: [
                new Touch({
                  identifier: 0,
                  target: canvas,
                  clientX: 80,
                  clientY: 80
                }),
                new Touch({
                  identifier: 1,
                  target: canvas,
                  clientX: 220,
                  clientY: 220
                })
              ]
            })
            canvas.dispatchEvent(touchMove)
          }, 100)
        }
      })
    }
    
    await page.waitForTimeout(500)
    await expect(gameArea).toBeVisible()
  })

  test('should maintain state across page interactions', async () => {
    // Perform some game actions
    await page.click('[data-testid="knirvana-game"]')
    await page.keyboard.press('KeyW')
    await page.keyboard.press('Space')
    
    // Simulate losing and regaining focus
    await page.evaluate(() => {
      window.dispatchEvent(new Event('blur'))
    })
    
    await page.waitForTimeout(500)
    
    await page.evaluate(() => {
      window.dispatchEvent(new Event('focus'))
    })
    
    // Game should still be responsive
    await expect(page.locator('[data-testid="knirvana-game"]')).toBeVisible()
    
    // Should still respond to controls
    await page.keyboard.press('KeyS')
    await page.waitForTimeout(100)
  })

  test('should handle memory management during extended play', async () => {
    // Monitor memory usage
    const initialMemory = await page.evaluate(() => {
      return (performance as any).memory?.usedJSHeapSize || 0
    })
    
    // Simulate extended gameplay
    for (let session = 0; session < 10; session++) {
      // Perform various actions
      for (let action = 0; action < 20; action++) {
        await page.keyboard.press(['KeyW', 'KeyA', 'KeyS', 'KeyD'][action % 4])
        await page.waitForTimeout(25)
      }
      
      // Simulate scene changes or heavy operations
      await page.evaluate(() => {
        // Force garbage collection if available
        if ((window as any).gc) {
          (window as any).gc()
        }
      })
      
      await page.waitForTimeout(100)
    }
    
    const finalMemory = await page.evaluate(() => {
      return (performance as any).memory?.usedJSHeapSize || 0
    })
    
    // Memory increase should be reasonable (less than 100MB)
    const memoryIncrease = finalMemory - initialMemory
    expect(memoryIncrease).toBeLessThan(100 * 1024 * 1024)
    
    // Game should still be responsive
    await expect(page.locator('[data-testid="knirvana-game"]')).toBeVisible()
  })
})
