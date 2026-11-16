import React from "react";
import { render, waitFor } from "@testing-library/react";
import { PWAProvider } from "../pwa-provider";

type Reg = { scope: string; unregister: jest.Mock };

function installServiceWorker(regs: Reg[], getRegistrations: jest.Mock) {
  const calls: string[] = [];
  const register = jest.fn(async (url: string, opts: { scope: string }) => {
    calls.push(`register ${url} ${opts.scope}`);
    return { scope: opts.scope };
  });
  for (const r of regs) {
    const original = r.unregister.getMockImplementation();
    r.unregister.mockImplementation(async () => {
      calls.push(`unregister ${r.scope}`);
      return original ? original() : true;
    });
  }
  const container = {
    register,
    getRegistrations,
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
  };
  Object.defineProperty(navigator, "serviceWorker", { value: container, configurable: true });
  return { container, register, calls };
}

function reg(scope: string): Reg {
  return { scope, unregister: jest.fn().mockResolvedValue(true) };
}

afterEach(() => {
  delete (navigator as unknown as { serviceWorker?: unknown }).serviceWorker;
  jest.restoreAllMocks();
});

describe("PWAProvider service worker wiring", () => {
  it("unregisters the root-scope worker before registering /app/service-worker.js", async () => {
    const root = reg("http://localhost/");
    const app = reg("http://localhost/app/");
    const getRegistrations = jest.fn().mockResolvedValue([root, app]);
    const { register, calls } = installServiceWorker([root, app], getRegistrations);

    render(<PWAProvider><span>child</span></PWAProvider>);

    await waitFor(() => expect(register).toHaveBeenCalledTimes(1));
    expect(register).toHaveBeenCalledWith("/app/service-worker.js", { scope: "/app/" });
    expect(root.unregister).toHaveBeenCalledTimes(1);
    expect(app.unregister).not.toHaveBeenCalled();
    expect(calls).toEqual([
      "unregister http://localhost/",
      "register /app/service-worker.js /app/",
    ]);
  });

  it("still registers /app/service-worker.js when root cleanup rejects", async () => {
    const getRegistrations = jest.fn().mockRejectedValue(new Error("boom"));
    const { register } = installServiceWorker([], getRegistrations);
    const warn = jest.spyOn(console, "warn").mockImplementation(() => {});
    jest.spyOn(console, "log").mockImplementation(() => {});

    render(<PWAProvider><span>child</span></PWAProvider>);

    await waitFor(() => expect(register).toHaveBeenCalledWith("/app/service-worker.js", { scope: "/app/" }));
    expect(warn).toHaveBeenCalled();
  });
});
