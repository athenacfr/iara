import { test, expect } from "@microsoft/tui-test";
import { createTestEnv, createFakeProject, IARA_BIN, waitForReady } from "./helpers.js";

const env = createTestEnv();
createFakeProject(env.projectsDir, "snap-project", {
  repos: ["web", "api"],
  metadata: { title: "Snap Project", description: "For snapshot tests" },
});

test.use({
  program: { file: IARA_BIN },
  rows: 24,
  columns: 80,
  env: env.env,
});

test.describe("Snapshots", () => {
  test("project list initial render", async ({ terminal }) => {
    await waitForReady(terminal);
    await expect(terminal.getByText("snap-project", { strict: false })).toBeVisible();
    await expect(terminal).toMatchSnapshot();
  });

  test("project list with expanded repos", async ({ terminal }) => {
    await waitForReady(terminal);
    terminal.write("t");
    await expect(terminal.getByText("web", { strict: false })).toBeVisible();
    await expect(terminal).toMatchSnapshot();
  });

  test("project list search active", async ({ terminal }) => {
    await waitForReady(terminal);
    terminal.write("s");
    terminal.write("snap");
    await expect(terminal.getByText("snap-project", { strict: false })).toBeVisible();
  });

  test("mode select initial render", async ({ terminal }) => {
    await waitForReady(terminal);
    terminal.submit();
    // Navigate through task select screen - select default branch
    await expect(terminal.getByText("TASKS")).toBeVisible();
    terminal.keyDown();
    terminal.submit();
    await expect(terminal.getByText("MODE")).toBeVisible();
    await expect(terminal).toMatchSnapshot();
  });

  test("create project name step", async ({ terminal }) => {
    await waitForReady(terminal);
    terminal.write("n");
    await expect(terminal.getByText("NEW PROJECT")).toBeVisible();
    await expect(terminal).toMatchSnapshot();
  });

  test("delete confirmation dialog", async ({ terminal }) => {
    await waitForReady(terminal);
    terminal.write("d");
    await expect(terminal.getByText(/Delete project/g)).toBeVisible();
    await expect(terminal).toMatchSnapshot();
  });
});
