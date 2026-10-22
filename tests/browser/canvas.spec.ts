import { test, expect, type Page, type WebSocketRoute } from "@playwright/test"
import AxeBuilder from "@axe-core/playwright"

const init = {
  type: "init",
  pixels: {},
  version: 0,
  cooldownMs: 0,
  protocol: 1,
  gridSize: 512
}
async function fakeServer(page: Page) {
  let socket: WebSocketRoute
  const writes: any[] = []
  const syncs: any[] = []
  let connections = 0
  await page.routeWebSocket("**/ws", (route) => {
    socket = route
    connections++
    route.onMessage((raw) => {
      const message = JSON.parse(String(raw))
      if (message.type === "init_client") route.send(JSON.stringify(init))
      if (message.type === "set_pixel") writes.push(message)
      if (message.type === "sync") syncs.push(message)
    })
  })
  return {
    writes,
    syncs,
    send: (message: object) => socket.send(JSON.stringify(message)),
    close: () => socket.close(),
    connections: () => connections
  }
}
async function select(page: Page, x = 200, y = 210) {
  await page.locator("#pixel-x").fill(String(x))
  await page.locator("#pixel-y").fill(String(y))
  await page.getByRole("button", { name: "Select", exact: true }).click()
}
async function connected(page: Page) {
  await expect(page.locator("#connection-overlay")).toBeHidden()
}

test("real WebSocket confirms, broadcasts, persists and enforces cooldown", async ({
  browser,
  page
}) => {
  const second = await browser.newPage()
  await page.goto("/")
  await connected(page)
  await second.goto("http://127.0.0.1:5410")
  await connected(second)
  await select(page, 100, 101)
  await page.getByRole("button", { name: "Blue", exact: true }).click()
  await page.locator("#cooldown").click()
  await expect(page.locator("#feedback")).toHaveText("Pixel saved.")
  await select(second, 100, 101)
  await expect(second.locator("#selection")).toContainText("Blue")
  await page.reload()
  await connected(page)
  await select(page, 100, 101)
  await expect(page.locator("#selection")).toContainText("Blue")
  await expect(page.locator("#cooldown")).toBeDisabled()
  await second.close()
})

test("rejections, delayed confirmation and reconnect preserve the right selection", async ({
  page
}) => {
  const server = await fakeServer(page)
  await page.goto("/")
  await connected(page)
  await select(page)
  await page.locator("#cooldown").click()
  await expect.poll(() => server.writes.length).toBe(1)
  await expect(page.locator("#cooldown")).toHaveText("Saving pixel…")
  server.send({
    type: "rejected",
    requestId: server.writes[0].requestId,
    code: "rate_limited",
    cooldownMs: 0
  })
  await expect(page.locator("#selection")).toContainText("200")
  await page.locator("#cooldown").click()
  await expect.poll(() => server.writes.length).toBe(2)
  await select(page, 205, 215)
  server.send({
    type: "accepted",
    requestId: server.writes[1].requestId,
    version: 1,
    cooldownMs: 0
  })
  await expect(page.locator("#feedback")).toHaveText("Pixel saved.")
  await expect(page.locator("#selection")).toContainText("205")
  await page.locator("#cooldown").click()
  await expect.poll(() => server.writes.length).toBe(3)
  server.close()
  await expect(page.locator("#feedback")).toContainText("Connection lost")
  await connected(page)
  expect(server.writes).toHaveLength(3) // uncertain writes must never auto-retry
  await expect(page.locator("#selection")).toContainText("205")
})

test("version gaps disable drawing and coalesce recovery into one contiguous delta", async ({
  page
}) => {
  const server = await fakeServer(page)
  await page.goto("/")
  await connected(page)
  await select(page)
  for (let version = 3; version <= 8; version++)
    server.send({ type: "pixel_update", version, x: 1, y: 2, color: "#ff0000" })
  await expect(page.locator("#cooldown")).toBeDisabled()
  await expect.poll(() => server.syncs.length).toBe(1)
  server.send({
    type: "sync",
    version: 2,
    cooldownMs: 0,
    events: [
      { version: 1, x: 1, y: 2, color: "#ff0000" },
      { version: 2, x: 1, y: 3, color: "#0000ff" }
    ]
  })
  await connected(page)
  await expect(page.locator("#cooldown")).toBeEnabled()
  await select(page, 1, 3)
  await expect(page.locator("#selection")).toContainText("Blue")
  server.send({
    type: "pixel_update",
    version: 4,
    x: 2,
    y: 3,
    color: "#ff0000"
  })
  await expect.poll(() => server.syncs.length).toBe(2)
  server.send({
    type: "sync",
    version: 4,
    cooldownMs: 0,
    events: [{ version: 4, x: 2, y: 3, color: "#ff0000" }]
  })
  await expect.poll(server.connections).toBe(2) // a corrupt delta requires authoritative re-init
  await connected(page)
})

for (const viewport of [
  { width: 1440, height: 900 },
  { width: 390, height: 844 },
  { width: 320, height: 568 },
  { width: 844, height: 390 }
]) {
  test(`keyboard, DPR and accessible layout at ${viewport.width}×${viewport.height}`, async ({
    browser
  }) => {
    const context = await browser.newContext({
      viewport,
      deviceScaleFactor: 2,
      reducedMotion: "reduce"
    })
    const page = await context.newPage()
    const errors: string[] = []
    page.on("pageerror", (error) => errors.push(error.message))
    await fakeServer(page)
    await page.goto("http://127.0.0.1:5410")
    await connected(page)
    await select(page)
    await page.getByRole("button", { name: "Blue", exact: true }).focus()
    await page.keyboard.press("Space")
    await expect(
      page.getByRole("button", { name: "Blue", exact: true })
    ).toHaveAttribute("aria-pressed", "true")
    await page.locator("#canvas").focus()
    await page.keyboard.press("ArrowRight")
    await expect(page.locator("#pixel-x")).toHaveValue("201")
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth
      )
    ).toBe(true)
    const size = await page
      .locator("#canvas")
      .evaluate((canvas: HTMLCanvasElement) => ({
        css: canvas.clientWidth,
        backing: canvas.width
      }))
    expect(size.backing).toBe(size.css * 2)
    for (const button of await page.locator("#palette button").all()) {
      await button.scrollIntoViewIfNeeded()
      await expect(button).toBeInViewport()
      const rect = await button.boundingBox()
      expect(rect!.width).toBeGreaterThanOrEqual(44)
      expect(rect!.height).toBeGreaterThanOrEqual(44)
    }
    const results = await new AxeBuilder({ page }).analyze()
    expect(results.violations).toEqual([])
    expect(errors).toEqual([])
    await context.close()
  })
}

test("blocked storage remains usable", async ({ page }) => {
  await page.addInitScript(() =>
    Object.defineProperty(window, "localStorage", {
      get() {
        throw new Error("blocked")
      }
    })
  )
  await fakeServer(page)
  await page.goto("/")
  await connected(page)
  await expect(page.locator("#feedback")).toContainText(
    "storage is unavailable"
  )
  await select(page)
  await expect(page.locator("#cooldown")).toBeEnabled()
  await page.getByRole("button", { name: "Privacy", exact: true }).click()
  await page.getByRole("button", { name: "Reset browser ID" }).click()
  await expect(page.locator("#privacy-feedback")).toContainText(
    "storage is blocked"
  )
})

test("offline recovery keeps the selection and a privacy reset replaces local identity", async ({
  page,
  context
}) => {
  await fakeServer(page)
  await page.goto("/")
  await connected(page)
  await select(page)
  const identity = await page.evaluate(() => localStorage.getItem("clientId"))
  await context.setOffline(true)
  await expect(page.locator("#cooldown")).toBeDisabled()
  await context.setOffline(false)
  await connected(page)
  await expect(page.locator("#selection")).toContainText("200")
  await page.getByRole("button", { name: "Privacy", exact: true }).click()
  await page.getByRole("button", { name: "Reset browser ID" }).click()
  await connected(page)
  await expect
    .poll(() => page.evaluate(() => localStorage.getItem("clientId")))
    .not.toBe(identity)
})

test("an idle canvas stops requesting animation frames", async ({ page }) => {
  await page.addInitScript(() => {
    const original = window.requestAnimationFrame.bind(window)
    ;(window as any).frameRequests = 0
    window.requestAnimationFrame = (callback) => {
      ;(window as any).frameRequests++
      return original(callback)
    }
  })
  await fakeServer(page)
  await page.goto("/")
  await connected(page)
  await page.waitForTimeout(300)
  const before = await page.evaluate(() => (window as any).frameRequests)
  await page.waitForTimeout(300)
  expect(await page.evaluate(() => (window as any).frameRequests)).toBe(before)
})

test("a slow drag does not place a click selection; touch pan and pinch work", async ({
  browser
}) => {
  const context = await browser.newContext({
    viewport: { width: 390, height: 844 },
    hasTouch: true,
    isMobile: true,
    reducedMotion: "reduce"
  })
  const page = await context.newPage()
  await fakeServer(page)
  await page.goto("http://127.0.0.1:5410")
  await connected(page)
  const box = (await page.locator("#canvas").boundingBox())!
  const x = box.x + box.width / 2,
    y = box.y + box.height / 2
  await page.mouse.move(x, y)
  await page.mouse.down()
  for (let step = 1; step <= 12; step++) {
    await page.mouse.move(x + step, y)
    await page.waitForTimeout(20)
  }
  await page.mouse.up()
  await expect(page.locator("#selection")).toHaveText(
    "Select a pixel to inspect its color."
  )
  const cdp = await context.newCDPSession(page)
  const point = (id: number, x: number, y: number) => ({ id, x, y })
  await cdp.send("Input.dispatchTouchEvent", {
    type: "touchStart",
    touchPoints: [point(0, x, y)]
  })
  await cdp.send("Input.dispatchTouchEvent", {
    type: "touchMove",
    touchPoints: [point(0, x - 70, y - 70)]
  })
  await cdp.send("Input.dispatchTouchEvent", {
    type: "touchEnd",
    touchPoints: []
  })
  await expect(page.locator("#selection")).toHaveText(
    "Select a pixel to inspect its color."
  )
  const before = await page
    .locator("#minimap")
    .evaluate((c: HTMLCanvasElement) => c.toDataURL())
  await cdp.send("Input.dispatchTouchEvent", {
    type: "touchStart",
    touchPoints: [point(0, x - 30, y), point(1, x + 30, y)]
  })
  await cdp.send("Input.dispatchTouchEvent", {
    type: "touchMove",
    touchPoints: [point(0, x - 65, y), point(1, x + 65, y)]
  })
  await cdp.send("Input.dispatchTouchEvent", {
    type: "touchEnd",
    touchPoints: []
  })
  await expect
    .poll(() =>
      page.locator("#minimap").evaluate((c: HTMLCanvasElement) => c.toDataURL())
    )
    .not.toBe(before)
  await context.close()
})

test("production HTTP applies CSP, revalidates HTML and caches hashed assets", async ({
  request
}) => {
  const response = await request.get("/")
  expect(response.headers()["content-security-policy"]).toContain(
    "frame-ancestors 'none'"
  )
  expect(response.headers()["cache-control"]).toBe("no-cache")
  const script = (await response.text()).match(/src="([^"]+\.js)"/)![1]!
  expect((await request.get(script)).headers()["cache-control"]).toContain(
    "immutable"
  )
  expect((await request.get("/assets/")).status()).toBe(404)
  expect((await request.get("/metrics")).status()).toBe(404)
  expect((await request.get("/backup")).status()).toBe(404)
})
