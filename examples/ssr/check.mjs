// Checks hydration in a real browser.
//
//   node check.mjs
//
// It builds nothing: run `make build` first. It starts the server and a headless
// Chromium, loads the page rendered by the server, waits for the hydration and
// checks that it did not touch the DOM, that the page works, and that a page
// whose HTML was broken on purpose recovers. Needs Node 22+ and Chromium.
import { execFileSync, spawn } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const SERVER_PORT = 8091;
const CHROME_PORT = 9334;
const BASE = `http://localhost:${SERVER_PORT}`;
const CHROMIUM = process.env.CHROMIUM || "chromium";
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function waitFor(url, what) {
  for (let i = 0; i < 100; i++) {
    try { if ((await fetch(url)).ok) return; } catch {}
    await sleep(200);
  }
  throw new Error(`${what} did not start`);
}

const profile = mkdtempSync(join(tmpdir(), "hx-check-"));
// The server is built first so it can be started, and stopped, as a single process.
execFileSync("go", ["build", "-o", join(profile, "server"), "./server"]);
const server = spawn(join(profile, "server"), ["-addr", `:${SERVER_PORT}`], { stdio: "ignore" });
const chrome = spawn(CHROMIUM, ["--headless=new", "--no-sandbox", "--disable-gpu",
  `--remote-debugging-port=${CHROME_PORT}`, `--user-data-dir=${profile}`, "about:blank"], { stdio: "ignore" });
const cleanup = () => { server.kill(); chrome.kill(); try { rmSync(profile, { recursive: true, force: true }); } catch {} };
process.on("exit", cleanup);

async function openPage(url) {
  const target = await (await fetch(`http://localhost:${CHROME_PORT}/json/new?${encodeURIComponent(url)}`, { method: "PUT" })).json();
  const ws = new WebSocket(target.webSocketDebuggerUrl);
  await new Promise((r) => (ws.onopen = r));
  let id = 0; const pending = new Map(); const logs = [];
  ws.onmessage = (e) => {
    const m = JSON.parse(e.data);
    if (m.id && pending.has(m.id)) { pending.get(m.id)(m.result); pending.delete(m.id); }
    else if (m.method === "Runtime.consoleAPICalled") logs.push(m.params.args.map((a) => a.value ?? a.description).join(" "));
    else if (m.method === "Runtime.exceptionThrown") logs.push("EXCEPTION " + (m.params.exceptionDetails.exception?.description ?? m.params.exceptionDetails.text));
  };
  const send = (method, params = {}) => new Promise((r) => { const i = ++id; pending.set(i, r); ws.send(JSON.stringify({ id: i, method, params })); });
  await send("Runtime.enable");
  const evaluate = async (expression) => (await send("Runtime.evaluate", { expression, returnByValue: true, awaitPromise: true })).result?.value;
  return { evaluate, logs, close: () => ws.close() };
}

// What a user does on the page. Returns what it saw.
const INTERACT = `(async () => {
  const w = (ms = 150) => new Promise((r) => setTimeout(r, ms));
  const $ = (s) => document.querySelector(s);
  const items = () => [...document.querySelectorAll("#list li span")].map((s) => s.textContent.trim());
  const out = {};
  out.initialMutations = { ...window.__mutations };
  out.mismatches = window.__hxHydrationMismatches;
  // The DOM written as text with the attributes sorted, so the order in which they were set does not matter.
  const canon = (n) => n.nodeType === 3 ? n.nodeValue : n.nodeType !== 1 ? "" :
    "<" + n.nodeName.toLowerCase() + [...n.attributes].map((a) => " " + a.name + "=" + JSON.stringify(a.value)).sort().join("") + ">" +
    [...n.childNodes].map(canon).join("") + "</" + n.nodeName.toLowerCase() + ">";
  out.html = canon(document.getElementById("app"));
  out.listAtStart = items();
  const adopted = [$("#inc"), $("#draft"), $("#count")];
  adopted.forEach((e, i) => (e.__mark = i));
  $("#inc").click(); $("#inc").click(); await w();
  out.count = $("#count").textContent;
  $("#draft").value = "learn hydration"; $("#draft").dispatchEvent(new Event("input")); await w();
  $("#add").click(); await w();
  out.afterAdd = items(); out.draftAfterAdd = $("#draft").value;
  $("#list li .remove").click(); await w();
  out.afterRemove = items();
  $("#color").value = "blue"; $("#color").dispatchEvent(new Event("change")); await w();
  out.picked = $("#picked").textContent;
  document.querySelectorAll("#list .remove").forEach((b) => b.click()); await w();
  out.emptyMessage = $("#empty").textContent;
  out.sameElements = adopted.every((e) => e.isConnected && e.__mark !== undefined);
  return out;
})()`;

const POLL = `window.__hxHydrationMismatches !== undefined && document.getElementById("status") && document.getElementById("status").textContent === "interactive"`;

async function run(variant) {
  const page = await openPage(`${BASE}/?break=${variant}&r=${Math.random()}`);
  for (let i = 0; i < 100 && !(await page.evaluate(POLL)); i++) await sleep(200);
  const result = await page.evaluate(INTERACT);
  const logs = page.logs.filter((l) => l.includes("hx:") || l.includes("EXCEPTION"));
  page.close();
  return { result, logs };
}

let failures = 0;
const check = (name, ok, detail = "") => {
  console.log(`${ok ? "PASS" : "FAIL"}  ${name}${ok ? "" : "   " + detail}`);
  if (!ok) failures++;
};
const same = (a, b) => JSON.stringify(a) === JSON.stringify(b);
const firstDifference = (got, want) => {
  let i = 0;
  while (i < got.length && got[i] === want[i]) i++;
  return `first difference at ${i}:\n      got:  ${got.slice(Math.max(0, i - 40), i + 60)}\n      want: ${want.slice(Math.max(0, i - 40), i + 60)}`;
};

await waitFor(`${BASE}/`, "the server");
await waitFor(`http://localhost:${CHROME_PORT}/json/version`, "Chromium");

const normal = await run("normal");
const r = normal.result;
console.log("--- normal page");
check("no mismatches", r.mismatches === 0, r.mismatches);
check("hydration creates and removes no nodes", r.initialMutations.childList === 0, JSON.stringify(r.initialMutations));
check("hydration sets no attributes", r.initialMutations.attributes === 0, JSON.stringify(r.initialMutations));
check("only OnMount changes the DOM (one text)", r.initialMutations.characterData === 1, JSON.stringify(r.initialMutations));
check("the server elements are the ones in use", r.sameElements);
check("clicking works", r.count === "count: 5", r.count);
check("adding works and clears the input", same(r.afterAdd, ["1. write docs", "2. ship it", "3. learn hydration"]) && r.draftAfterAdd === "", JSON.stringify(r.afterAdd));
check("removing works", same(r.afterRemove, ["1. ship it", "2. learn hydration"]), JSON.stringify(r.afterRemove));
check("select works", r.picked === "color: blue", r.picked);
check("a conditional that was empty on the server appears", r.emptyMessage === "Nothing to do", r.emptyMessage);
check("no errors in the console", normal.logs.length === 0, normal.logs.join("\n"));

const expectedHtml = r.html;
const broken = {
  class: "a missing class", attr: "a different attribute", id: "a different id",
  text: "different text", tag: "a different tag", count: "a missing item", empty: "no HTML at all",
};
for (const [variant, what] of Object.entries(broken)) {
  console.log(`--- ${what} (break=${variant})`);
  const { result, logs } = await run(variant);
  check("the mismatch is reported", result.mismatches >= 1 && logs.some((l) => l.includes("hydration mismatch")), `${result.mismatches} ${logs.join(" | ")}`);
  check("the page ends up the same as the normal one", result.html === expectedHtml, firstDifference(result.html, expectedHtml));
  check("the page works", result.count === "count: 5" && same(result.afterAdd, ["1. write docs", "2. ship it", "3. learn hydration"]) && result.emptyMessage === "Nothing to do");
  check("no exceptions", !logs.some((l) => l.includes("EXCEPTION")), logs.join("\n"));
}

console.log(failures === 0 ? "\nall good" : `\n${failures} failed`);
process.exit(failures === 0 ? 0 : 1);
