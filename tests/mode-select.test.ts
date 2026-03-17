import { test, expect } from "@microsoft/tui-test";
import { createTestEnv, createFakeProject, IARA_BIN, waitForReady } from "./helpers.js";

const env = createTestEnv();
createFakeProject(env.projectsDir, "mode-test-project", {
  repos: ["app"],
  metadata: { title: "Mode Test", description: "For mode select tests" },
});

test.use({
  program: { file: IARA_BIN },
  rows: 24,
  columns: 80,
  env: env.env,
});

// Navigate to mode select screen (project list → task select → mode select)
async function goToModeSelect(terminal: any) {
  await waitForReady(terminal);
  await expect(terminal.getByText("mode-test-project", { strict: false })).toBeVisible();
  terminal.submit();
  // Navigate through task select screen - select default branch (item 2)
  await expect(terminal.getByText("TASKS")).toBeVisible();
  terminal.keyDown();
  terminal.submit();
  await expect(terminal.getByText("MODE")).toBeVisible();
}

test.describe("Mode Select", () => {
  test("shows MODE header after selecting project", async ({ terminal }) => {
    await goToModeSelect(terminal);
  });

  test("shows SESSIONS section", async ({ terminal }) => {
    await goToModeSelect(terminal);
    await expect(terminal.getByText("SESSIONS")).toBeVisible();
  });

  test("shows New Session option", async ({ terminal }) => {
    await goToModeSelect(terminal);
    await expect(terminal.getByText(/New Session/g, { strict: false })).toBeVisible();
  });

  test("shows available modes", async ({ terminal }) => {
    await goToModeSelect(terminal);
    await expect(terminal.getByText(/code/g, { strict: false })).toBeVisible();
  });

  test("switches mode with right arrow", async ({ terminal }) => {
    await goToModeSelect(terminal);
    terminal.keyRight();
    await expect(terminal).toMatchSnapshot();
  });

  test("shows permission toggle hint", async ({ terminal }) => {
    await goToModeSelect(terminal);
    await expect(terminal.getByText(/permissions/g, { strict: false })).toBeVisible();
  });

  test("toggles permissions with tab", async ({ terminal }) => {
    await goToModeSelect(terminal);
    // Default is "skip permissions" (bypass=true from global settings)
    await expect(terminal.getByText(/skip permissions/g)).toBeVisible();
    terminal.keyPress("Tab");
    await expect(terminal.getByText(/normal permissions/g)).toBeVisible();
  });

  test("returns to task select on Escape", async ({ terminal }) => {
    await goToModeSelect(terminal);
    terminal.keyEscape();
    await expect(terminal.getByText("TASKS")).toBeVisible();
  });

  test("launches on Enter (exits TUI)", async ({ terminal }) => {
    await goToModeSelect(terminal);
    await expect(terminal.getByText(/New Session/g, { strict: false })).toBeVisible();
    terminal.submit();
    terminal.onExit((exit) => {
      expect(exit.exitCode).toBe(0);
    });
  });
});
