// Checks the form in a real browser, without JavaScript and with it.
//   node check.mjs        (run `make build` first; needs Node 22+ and Chromium)
import { execFileSync, spawn } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const SERVER_PORT = 8098, CHROME_PORT = 9336, BASE = `http://localhost:${SERVER_PORT}`;
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const work = mkdtempSync(join(tmpdir(), "hx-form-"));
execFileSync("go", ["build", "-o", join(work, "server"), "./server"]);
const server = spawn(join(work, "server"), ["-addr", `:${SERVER_PORT}`], { stdio: "ignore" });
const chrome = spawn(process.env.CHROMIUM || "chromium", ["--headless=new", "--no-sandbox", "--disable-gpu",
  `--remote-debugging-port=${CHROME_PORT}`, `--user-data-dir=${join(work, "profile")}`, "about:blank"], { stdio: "ignore" });
process.on("exit", () => { server.kill(); chrome.kill(); try { rmSync(work, { recursive: true, force: true }); } catch {} });

async function until(fn, what, ms = 8000) {
  const end = Date.now() + ms;
  for (;;) {
    let value; try { value = await fn(); } catch {}
    if (value) return value;
    if (Date.now() > end) throw new Error(`timed out waiting for ${what}`);
    await sleep(80);
  }
}

async function openTab({ scripts }) {
  const target = await (await fetch(`http://localhost:${CHROME_PORT}/json/new?about:blank`, { method: "PUT" })).json();
  const ws = new WebSocket(target.webSocketDebuggerUrl);
  await new Promise((r) => (ws.onopen = r));
  let id = 0; const pending = new Map(); const events = [];
  ws.onmessage = (e) => {
    const m = JSON.parse(e.data);
    if (m.id && pending.has(m.id)) { pending.get(m.id)(m.result); pending.delete(m.id); } else events.push(m.method);
  };
  const send = (method, params = {}) => new Promise((r) => { const i = ++id; pending.set(i, r); ws.send(JSON.stringify({ id: i, method, params })); });
  await send("Runtime.enable"); await send("Page.enable");
  if (!scripts) await send("Emulation.setScriptExecutionDisabled", { value: true });
  const evaluate = async (expression) => (await send("Runtime.evaluate", { expression, returnByValue: true, awaitPromise: true })).result?.value;
  return {
    evaluate,
    async goto(url) { events.length = 0; await send("Page.navigate", { url }); await until(() => events.includes("Page.loadEventFired"), "the page"); },
    async click(selector) { events.length = 0; await evaluate(`document.querySelector(${JSON.stringify(selector)}).click()`); },
    async clickAndWait(selector) { await this.click(selector); await until(() => events.includes("Page.loadEventFired"), "the navigation"); },
    close: () => ws.close(),
  };
}

let failures = 0;
const check = (name, ok, detail = "") => { console.log(`${ok ? "PASS" : "FAIL"}  ${name}${ok ? "" : "   " + detail}`); if (!ok) failures++; };
const subscribers = async () => (await fetch(`${BASE}/subscribers`)).json();
const type = (value) => `document.getElementById("email").value = ${JSON.stringify(value)}; document.getElementById("email").dispatchEvent(new Event("input"))`;

await until(async () => (await fetch(`${BASE}/`)).ok, "the server");
await until(async () => (await fetch(`http://localhost:${CHROME_PORT}/json/version`)).ok, "Chromium");

console.log("--- without JavaScript");
{
  const tab = await openTab({ scripts: false });
  await tab.goto(`${BASE}/`);
  await tab.evaluate(`document.getElementById("email").value = "nope"`);
  await tab.clickAndWait("#subscribe");
  const bad = await tab.evaluate(`({ status: performance.getEntriesByType("navigation")[0].responseStatus,
    problem: document.getElementById("problem").textContent, email: document.getElementById("email").value })`);
  check("a wrong email comes back as the same page, with the error and what was typed",
    bad.status === 422 && bad.problem === "That is not an email" && bad.email === "nope", JSON.stringify(bad));
  await tab.evaluate(`document.getElementById("email").value = "ada@example.com"`);
  await tab.clickAndWait("#subscribe");
  const ok = await tab.evaluate(`({ url: location.pathname + location.search, thanks: document.getElementById("thanks").textContent })`);
  check("a good one redirects and says thanks", ok.url === "/?done=1" && ok.thanks === "Thanks, you are subscribed!", JSON.stringify(ok));
  check("the server has it", (await subscribers()).join() === "ada@example.com");
  tab.close();
}

console.log("--- with JavaScript");
{
  const tab = await openTab({ scripts: true });
  await tab.goto(`${BASE}/`);
  await until(() => tab.evaluate(`window.__hxHydrationMismatches !== undefined`), "the hydration");
  check("the page hydrates without mismatches", (await tab.evaluate("window.__hxHydrationMismatches")) === 0);

  await tab.evaluate(`window.__same = true; ${type("nope")}`);
  await tab.click("#subscribe");
  await until(() => tab.evaluate(`document.getElementById("problem").textContent === "That is not an email"`), "the error");
  check("a wrong email is refused in the browser, with no request", (await subscribers()).length === 1);

  await tab.evaluate(type("grace@example.com"));
  await tab.click("#subscribe");
  await until(() => tab.evaluate(`document.getElementById("thanks").textContent !== ""`), "the answer");
  const r = await tab.evaluate(`({ same: window.__same, path: location.pathname + location.search,
    thanks: document.getElementById("thanks").textContent, email: document.getElementById("email").value,
    problem: document.getElementById("problem").textContent })`);
  check("it was sent without leaving the page", r.same === true && r.path === "/", JSON.stringify(r));
  check("it says thanks and the field is empty", r.thanks === "Thanks, you are subscribed!" && r.email === "" && r.problem === "", JSON.stringify(r));
  check("the server has both", (await subscribers()).join() === "ada@example.com,grace@example.com");
  tab.close();
}

console.log(failures === 0 ? "\nall good" : `\n${failures} failed`);
process.exit(failures === 0 ? 0 : 1);
