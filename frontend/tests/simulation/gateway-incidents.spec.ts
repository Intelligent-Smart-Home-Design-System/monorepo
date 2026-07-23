import { createHmac } from "node:crypto";
import { expect, test } from "@playwright/test";

const gatewayURL = process.env.SIM_GATEWAY_E2E_URL ?? "http://127.0.0.1:8090";
const jwtSecret = process.env.JWT_SECRET ?? "dev-jwt-secret";

function base64URL(value: string | Buffer) {
  return Buffer.from(value).toString("base64url");
}

function issueTestJWT(subject = "playwright-simulation") {
  const now = Math.floor(Date.now() / 1000);
  const header = base64URL(JSON.stringify({ alg: "HS256", typ: "JWT" }));
  const payload = base64URL(JSON.stringify({ sub: subject, iat: now, exp: now + 300 }));
  const unsigned = `${header}.${payload}`;
  const signature = createHmac("sha256", jwtSecret).update(unsigned).digest("base64url");
  return `${unsigned}.${signature}`;
}

test("gateway rejects simulation websocket without JWT", async ({ request }) => {
  const response = await request.get(`${gatewayURL}/api/v1/simulation/ws`);
  expect(response.status()).toBe(401);
  await expect(response.json()).resolves.toMatchObject({ message: "token is required" });
});

test("gateway rejects simulation websocket with invalid JWT", async ({ request }) => {
  const response = await request.get(`${gatewayURL}/api/v1/simulation/ws?token=invalid.jwt.signature`);
  expect(response.status()).toBe(401);
  await expect(response.json()).resolves.toMatchObject({ message: "invalid token" });
});

test("frontend renders, resets and restarts backend incidents through gateway", async ({ page }) => {
  const token = issueTestJWT();
  await page.addInitScript(({ accessToken }) => {
    localStorage.clear();
    localStorage.setItem("smart-home-auth", JSON.stringify({ tokens: { access_token: accessToken } }));
    localStorage.setItem(
      "simulation-devices",
      JSON.stringify([{ id: "smoke_sensor", type: "smoke_sensor", x: 0.79, y: 0.25 }])
    );
  }, { accessToken: token });

  await page.goto("/sim-ui/simulation");
  await expect(page.getByTestId("websocket-status")).toHaveText("Бэк подключен");
  await expect(page.getByTestId("simulation-error")).toHaveCount(0);
  await page.getByTestId("fire-start").click();
  const plan = page.getByTestId("plan-surface");
  const box = await plan.boundingBox();
  if (!box) throw new Error("plan surface has no bounding box");
  await plan.click({ position: { x: box.width * 0.79, y: box.height * 0.31 } });
  await expect(page.locator(".event-log-message", { hasText: "Бэкенд запустил симуляцию" })).toBeVisible();

  await expect(page.getByTestId("incident-layer")).toBeVisible();
  await expect(page.getByTestId("incident-layer").locator("polygon")).not.toHaveCount(0);
  await expect(page.getByTestId("device-smoke_sensor")).toHaveAttribute("data-device-state", "idle");

  await page.getByTestId("fire-reset").click();
  await expect(page.getByTestId("incident-layer")).toHaveCount(0);
  await expect(page.getByTestId("device-smoke_sensor")).toHaveAttribute("data-device-state", "idle");

  await page.getByTestId("fire-start").click();
  await plan.click({ position: { x: box.width * 0.79, y: box.height * 0.315 } });
  await expect(page.getByTestId("incident-layer")).toBeVisible();
});
