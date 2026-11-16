import { retireRootServiceWorkers } from "../pwa-provider";

function registration(scope: string) {
  return { scope, unregister: jest.fn().mockResolvedValue(true) };
}

describe("retireRootServiceWorkers", () => {
  it("unregisters only the root-scope registration and keeps /app/ scope", async () => {
    const root = registration("https://knirv.example/");
    const app = registration("https://knirv.example/app/");
    const container = { getRegistrations: jest.fn().mockResolvedValue([root, app]) };

    await expect(retireRootServiceWorkers(container)).resolves.toBe(1);

    expect(root.unregister).toHaveBeenCalledTimes(1);
    expect(app.unregister).not.toHaveBeenCalled();
  });

  it("is a no-op when no root-scope registration exists", async () => {
    const app = registration("https://knirv.example/app/");
    const container = { getRegistrations: jest.fn().mockResolvedValue([app]) };

    await expect(retireRootServiceWorkers(container)).resolves.toBe(0);
    expect(app.unregister).not.toHaveBeenCalled();
  });
});
