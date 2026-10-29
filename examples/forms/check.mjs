// Checks the forms in a real browser, with and without JavaScript.
//
//   node check.mjs
//
// Run `make build` first. It starts the server and a headless Chromium. Needs
// Node 22+ and Chromium.
import { execFileSync, spawn } from "node:child_process";
import { createHash } from "node:crypto";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const SERVER_PORT = 8096;
const CHROME_PORT = 9335;
const BASE = `http://localhost:${SERVER_PORT}`;
const CHROMIUM = process.env.CHROMIUM || "chromium";
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const work = mkdtempSync(join(tmpdir(), "hx-forms-"));
execFileSync("go", ["build", "-o", join(work, "server"), "./server"]);
const server = spawn(join(work, "server"), ["-addr", `:${SERVER_PORT}`], { stdio: "ignore" });
const chrome = spawn(CHROMIUM, ["--headless=new", "--no-sandbox", "--disable-gpu",
  `--remote-debugging-port=${CHROME_PORT}`, `--user-data-dir=${join(work, "profile")}`, "about:blank"], { stdio: "ignore" });
const cleanup = () => { server.kill(); chrome.kill(); try { rmSync(work, { recursive: true, force: true }); } catch {} };
process.on("exit", cleanup);

async function waitFor(url, what) {
  for (let i = 0; i < 100; i++) {
    try { if ((await fetch(url)).ok) return; } catch {}
    await sleep(200);
  }
  throw new Error(`${what} did not start`);
}

async function until(fn, what, ms = 8000) {
  const end = Date.now() + ms;
  for (;;) {
    const value = await fn();
    if (value) return value;
    if (Date.now() > end) throw new Error(`timed out waiting for ${what}`);
    await sleep(60);
  }
}

// A browser tab driven with the Chrome DevTools Protocol.
async function openTab({ scripts = true } = {}) {
  const target = await (await fetch(`http://localhost:${CHROME_PORT}/json/new?about:blank`, { method: "PUT" })).json();
  const ws = new WebSocket(target.webSocketDebuggerUrl);
  await new Promise((r) => (ws.onopen = r));
  let id = 0; const pending = new Map(); const logs = []; const events = [];
  ws.onmessage = (e) => {
    const m = JSON.parse(e.data);
    if (m.id && pending.has(m.id)) { pending.get(m.id)(m.result ?? { error: m.error }); pending.delete(m.id); }
    else if (m.method === "Runtime.consoleAPICalled") logs.push(m.params.args.map((a) => a.value ?? a.description).join(" "));
    else if (m.method === "Runtime.exceptionThrown") logs.push("EXCEPTION " + (m.params.exceptionDetails.exception?.description ?? m.params.exceptionDetails.text));
    else events.push(m.method);
  };
  const send = (method, params = {}) => new Promise((r) => { const i = ++id; pending.set(i, r); ws.send(JSON.stringify({ id: i, method, params })); });
  await send("Runtime.enable"); await send("Page.enable"); await send("DOM.enable");
  if (!scripts) await send("Emulation.setScriptExecutionDisabled", { value: true });
  const evaluate = async (expression) => {
    const r = await send("Runtime.evaluate", { expression, returnByValue: true, awaitPromise: true });
    if (r.exceptionDetails) throw new Error("evaluate failed: " + (r.exceptionDetails.exception?.description ?? r.exceptionDetails.text));
    return r.result?.value;
  };
  const tab = {
    send, evaluate, logs,
    async goto(url) {
      events.length = 0;
      await send("Page.navigate", { url });
      await until(() => events.includes("Page.loadEventFired"), "the page to load");
    },
    async setFiles(selector, files) {
      const { root } = await send("DOM.getDocument", { depth: 0 });
      const { nodeId } = await send("DOM.querySelector", { nodeId: root.nodeId, selector });
      await send("DOM.setFileInputFiles", { files, nodeId });
    },
    async clickAndWaitForNavigation(selector) {
      events.length = 0;
      await evaluate(`document.querySelector(${JSON.stringify(selector)}).click()`);
      await until(() => events.includes("Page.loadEventFired"), "the navigation");
    },
    close() { ws.close(); },
  };
  return tab;
}

let failures = 0;
const check = (name, ok, detail = "") => {
  console.log(`${ok ? "PASS" : "FAIL"}  ${name}${ok ? "" : "   " + detail}`);
  if (!ok) failures++;
};
const stats = async () => (await fetch(`${BASE}/debug/stats`)).json();
const reset = () => fetch(`${BASE}/debug/reset`, { method: "POST" });
const sha = (text) => createHash("sha256").update(text).digest("hex").slice(0, 12);
const fill = (fields) => Object.entries(fields).map(([id, value]) =>
  `document.getElementById(${JSON.stringify(id)}).value = ${JSON.stringify(value)};`).join("\n");

await waitFor(`${BASE}/`, "the server");
await waitFor(`http://localhost:${CHROME_PORT}/json/version`, "Chromium");

const avatarText = "an avatar, as text";
const noteText = "first attachment";
const noteText2 = "second attachment, a bit longer";
const avatarPath = join(work, "me.txt"), noteA = join(work, "a.txt"), noteB = join(work, "b.txt");
writeFileSync(avatarPath, avatarText); writeFileSync(noteA, noteText); writeFileSync(noteB, noteText2);

// ---------------------------------------------------------------------------
console.log("--- without JavaScript: the forms are plain HTML forms");
{
  await reset();
  const tab = await openTab({ scripts: false });
  await tab.goto(`${BASE}/`);
  check("scripts are really off", (await tab.evaluate("typeof window.__mutations")) === "undefined");
  check("the form has what a browser needs to send it",
    await tab.evaluate(`(() => { const f = document.getElementById("contact-form");
      return f.method === "post" && f.getAttribute("action") === "/contacts" && f.enctype === "multipart/form-data"; })()`));

  // a wrong send: the page comes back with what was typed and what is wrong
  await tab.evaluate(fill({ name: "Bob", email: "not-an-email", message: "typed before the error" }) +
    `document.getElementById("country").value = "fr";`);
  await tab.clickAndWaitForNavigation("#submit");
  const bad = await tab.evaluate(`({
    status: performance.getEntriesByType("navigation")[0].responseStatus, path: location.pathname,
    email: document.getElementById("error-email").textContent,
    name: document.getElementById("name").value, message: document.getElementById("message").value,
    country: document.getElementById("country").value })`);
  check("an invalid form answers 422 with the same page", bad.status === 422 && bad.path === "/contacts", JSON.stringify(bad));
  check("the error is written next to the field", bad.email === "That is not an email", bad.email);
  check("what the user typed is still there", bad.name === "Bob" && bad.message === "typed before the error" && bad.country === "fr", JSON.stringify(bad));

  // a good send with three files
  await tab.evaluate(fill({ name: "Ada", email: "ada@example.com", message: "hello" }) +
    `document.getElementById("country").value = "de"; document.getElementById("newsletter").checked = true;`);
  await tab.setFiles("#avatar", [avatarPath]);
  await tab.setFiles("#attachments", [noteA, noteB]);
  await tab.clickAndWaitForNavigation("#submit");
  const saved = await tab.evaluate(`({
    url: location.pathname + location.search,
    banner: document.getElementById("banner").textContent,
    contact: document.querySelector("#contacts .contact")?.innerText ?? "" })`);
  check("a valid form redirects to the page (post, redirect, get)", saved.url === "/?saved=1", saved.url);
  check("the page says it was saved", saved.banner === "Saved contact #1", saved.banner);
  check("the avatar arrived with its size and hash", saved.contact.includes(`avatar me.txt (${avatarText.length} bytes, ${sha(avatarText)})`), saved.contact);
  check("both attachments arrived", saved.contact.includes(`a.txt (${noteText.length} bytes, ${sha(noteText)})`) &&
    saved.contact.includes(`b.txt (${noteText2.length} bytes, ${sha(noteText2)})`), saved.contact);
  check("the other fields arrived", saved.contact.includes("Ada <ada@example.com> from Germany"), saved.contact);

  await tab.clickAndWaitForNavigation("#contacts .delete-button");
  const afterDelete = await tab.evaluate(`({ contacts: document.querySelectorAll("#contacts .contact").length,
    empty: document.getElementById("no-contacts").textContent })`);
  check("the delete button is a form too", afterDelete.contacts === 0 && afterDelete.empty === "No contacts yet", JSON.stringify(afterDelete));

  const s = await stats();
  check("the server saw classic requests only", s["classic-saves"] === 1 && s["classic-rejected"] === 1 && s["classic-deletes"] === 1 && !s["json-saves"], JSON.stringify(s));
  check("no errors in the console", tab.logs.length === 0, tab.logs.join("\n"));
  tab.close();
}

// ---------------------------------------------------------------------------
console.log("--- the server protects the forms");
{
  await reset();
  const home = await fetch(`${BASE}/`);
  const cookie = home.headers.getSetCookie()[0].split(";")[0];
  const token = cookie.split("=")[1];
  const post = (fields, headers = {}) => {
    const form = new FormData();
    for (const [k, v] of Object.entries(fields)) form.append(k, v);
    return fetch(`${BASE}/contacts`, { method: "POST", body: form, headers: { Accept: "application/json", ...headers }, redirect: "manual" });
  };
  const good = { name: "Eve", email: "eve@example.com", country: "es" };
  check("a form without the token is refused", (await post(good, { Cookie: cookie })).status === 403);
  check("a wrong token is refused", (await post({ ...good, csrf: "nope" }, { Cookie: cookie })).status === 403);
  check("without the cookie it is refused", (await post({ ...good, csrf: token })).status === 403);
  const ok = await post({ ...good, csrf: token }, { Cookie: cookie });
  check("the right token works", ok.status === 200 && (await ok.json()).ok === true);
  const viaHeader = await fetch(`${BASE}/api/contacts/1`, { method: "DELETE", headers: { Cookie: cookie, "X-CSRF-Token": token, Accept: "application/json" } });
  check("a JSON request can send the token in a header", viaHeader.status === 200);
  check("deleting what does not exist is a 404", (await fetch(`${BASE}/api/contacts/1`, { method: "DELETE", headers: { Cookie: cookie, "X-CSRF-Token": token } })).status === 404);
}

// ---------------------------------------------------------------------------
console.log("--- with JavaScript: the WASM hydrates and sends the forms with hx.Do");
{
  await reset();
  const tab = await openTab();
  await tab.goto(`${BASE}/`);
  const ready = `window.__hxHydrationMismatches !== undefined && document.getElementById("status")?.textContent === "interactive"`;
  await until(() => tab.evaluate(ready), "the hydration");

  const first = await tab.evaluate(`({ m: window.__hxHydrationMismatches, mut: { ...window.__mutations } })`);
  check("hydrating the page changes nothing but the status text",
    first.m === 0 && first.mut.childList === 0 && first.mut.attributes === 0 && first.mut.characterData === 1, JSON.stringify(first));

  // the form is checked in the browser first: no request is made
  await tab.evaluate(`window.__navMarker = "same page"; document.getElementById("submit").click()`);
  await until(() => tab.evaluate(`document.getElementById("error-name").textContent !== ""`), "the browser-side errors");
  const empty = await tab.evaluate(`({ name: document.getElementById("error-name").textContent, email: document.getElementById("error-email").textContent })`);
  check("an empty form is refused in the browser, with the same rules", empty.name === "Write your name" && empty.email === "That is not an email", JSON.stringify(empty));
  check("without sending anything", Object.keys(await stats()).length === 0, JSON.stringify(await stats()));

  // a rule only the server knows
  await tab.evaluate(fill({ name: "Eve", email: "eve@site.invalid" }));
  await tab.evaluate(`document.getElementById("name").dispatchEvent(new Event("input")); document.getElementById("email").dispatchEvent(new Event("input"));
    document.getElementById("submit").click()`);
  await until(() => tab.evaluate(`document.getElementById("error-email").textContent === "That domain does not accept mail"`), "the error of the server");
  const refused = await tab.evaluate(`({ name: document.getElementById("name").value, contacts: document.querySelectorAll("#contacts .contact").length,
    banner: document.getElementById("banner").textContent, submit: document.getElementById("submit").textContent })`);
  check("the 422 of the server shows its errors without reloading", refused.contacts === 0 && refused.submit === "Save contact" && refused.banner === "");
  check("what was typed is kept", refused.name === "Eve", JSON.stringify(refused));

  // a good send, with files, and the progress of the upload
  await tab.evaluate(`(() => {
    document.getElementById("name").value = "Ada"; document.getElementById("name").dispatchEvent(new Event("input"));
    document.getElementById("email").value = "ada@example.com"; document.getElementById("email").dispatchEvent(new Event("input"));
    const select = document.getElementById("country"); select.value = "de"; select.dispatchEvent(new Event("change"));
    const news = document.getElementById("newsletter"); news.checked = true; news.dispatchEvent(new Event("change"));
    const pick = (id, files) => { const dt = new DataTransfer(); files.forEach(([content, name]) => dt.items.add(new File([content], name, { type: "text/plain" })));
      document.getElementById(id).files = dt.files; };
    pick("avatar", [[${JSON.stringify(avatarText)}, "me.txt"]]);
    pick("attachments", [[${JSON.stringify(noteText)}, "a.txt"], [${JSON.stringify(noteText2)}, "b.txt"]]);
    window.__progress = []; new MutationObserver(() => window.__progress.push(document.getElementById("progress").getAttribute("value")))
      .observe(document.getElementById("progress"), { attributes: true });
    document.getElementById("submit").click();
  })()`);
  check("while it is sent the button says so", await until(() => tab.evaluate(`document.getElementById("submit").textContent === "Saving..." || document.getElementById("banner").textContent !== ""`), "the sending"));
  await until(() => tab.evaluate(`document.getElementById("banner").textContent === "Saved contact #1"`), "the answer");
  const done = await tab.evaluate(`({
    navMarker: window.__navMarker, url: location.pathname + location.search,
    contact: document.querySelector("#contacts .contact")?.innerText ?? "",
    name: document.getElementById("name").value, country: document.getElementById("country").value,
    news: document.getElementById("newsletter").checked,
    avatarFiles: document.getElementById("avatar").files.length, attachmentFiles: document.getElementById("attachments").files.length,
    errors: [...document.querySelectorAll(".error")].map((e) => e.textContent).join(""),
    progress: document.getElementById("progress").getAttribute("value"),
    submit: document.getElementById("submit").textContent, disabled: document.getElementById("submit").disabled })`);
  check("the page did not reload", done.navMarker === "same page" && done.url === "/", JSON.stringify(done));
  check("the new contact is in the list", done.contact.includes("Ada <ada@example.com> from Germany"), done.contact);
  check("the files were uploaded (size and hash match)", done.contact.includes(`avatar me.txt (${avatarText.length} bytes, ${sha(avatarText)})`) &&
    done.contact.includes(`a.txt (${noteText.length} bytes, ${sha(noteText)})`) && done.contact.includes(`b.txt (${noteText2.length} bytes, ${sha(noteText2)})`), done.contact);
  check("the form was emptied, files included", done.name === "" && done.country === "es" && !done.news && done.avatarFiles === 0 && done.attachmentFiles === 0 && done.errors === "", JSON.stringify(done));
  check("the progress bar reached the end", done.progress === "1.00", done.progress);
  check("the button is ready again", done.submit === "Save contact" && !done.disabled, `${done.submit} ${done.disabled}`);

  // the list is keyed: a second contact does not recreate the first one
  await tab.evaluate(`document.querySelector("#contacts .contact").__first = true;` +
    fill({ name: "Grace", email: "grace@example.com" }) +
    `document.getElementById("name").dispatchEvent(new Event("input")); document.getElementById("email").dispatchEvent(new Event("input")); document.getElementById("submit").click()`);
  await until(() => tab.evaluate(`document.getElementById("banner").textContent === "Saved contact #2"`), "the second save");
  const two = await tab.evaluate(`({ names: [...document.querySelectorAll("#contacts .contact strong")].map((e) => e.textContent.split(" ")[0]),
    keptFirst: [...document.querySelectorAll("#contacts .contact")].some((e) => e.__first === true) })`);
  check("the newest contact goes first and the first one is the same element", two.names.join() === "Grace,Ada" && two.keptFirst, JSON.stringify(two));

  // delete, with JSON and the token in a header
  await tab.evaluate(`document.querySelector("#contact-1 .delete-button").click()`);
  await until(() => tab.evaluate(`!document.getElementById("contact-1")`), "the delete");
  const left = await tab.evaluate(`[...document.querySelectorAll("#contacts .contact strong")].map((e) => e.textContent.split(" ")[0]).join()`);
  check("deleting works without reloading", left === "Grace" && (await tab.evaluate("window.__navMarker")) === "same page", left);

  // the resource: load again what the server has
  const cookieHeader = await tab.evaluate("document.cookie"); // csrf is HttpOnly: not visible
  void cookieHeader;
  const home = await fetch(`${BASE}/`);
  const cookie = home.headers.getSetCookie()[0].split(";")[0];
  const form = new FormData(); form.append("csrf", cookie.split("=")[1]); form.append("name", "Linus"); form.append("email", "linus@example.com"); form.append("country", "fr");
  await fetch(`${BASE}/contacts`, { method: "POST", body: form, headers: { Cookie: cookie, Accept: "application/json" } });
  await tab.evaluate(`document.getElementById("refresh").click()`);
  await until(() => tab.evaluate(`document.querySelectorAll("#contacts .contact").length === 2`), "the refresh");
  const refreshed = await tab.evaluate(`[...document.querySelectorAll("#contacts .contact strong")].map((e) => e.textContent.split(" ")[0]).join()`);
  check("Refresh loads what the server has now", refreshed === "Linus,Grace", refreshed);

  const s = await stats();
  check("the server saw JSON requests, not classic ones", s["json-saves"] === 3 && s["json-deletes"] === 1 && !s["classic-saves"] && !s["classic-deletes"], JSON.stringify(s));
  check("no errors in the console", !tab.logs.some((l) => l.includes("EXCEPTION") || l.includes("hx:")), tab.logs.join("\n"));
  tab.close();
}

// ---------------------------------------------------------------------------
console.log("--- a page that came back from a failed classic send hydrates too");
{
  await reset();
  const tab = await openTab();
  await tab.goto(`${BASE}/`);
  await until(() => tab.evaluate(`window.__hxHydrationMismatches !== undefined`), "the first hydration");
  // submit() does not fire the submit event: this is what a browser does when the WASM is not ready yet
  await tab.evaluate(fill({ name: "Bob", email: "nope", message: "kept" }) + `
    document.getElementById("country").value = "us"; document.getElementById("newsletter").checked = true;`);
  await tab.evaluate(`HTMLFormElement.prototype.submit.call(document.getElementById("contact-form"))`).catch(() => {});
  await until(() => tab.evaluate(`location.pathname === "/contacts" && window.__hxHydrationMismatches !== undefined && document.getElementById("status")?.textContent === "interactive"`), "the page of the POST", 15000);
  const r = await tab.evaluate(`({ m: window.__hxHydrationMismatches, mut: { ...window.__mutations },
    name: document.getElementById("name").value, message: document.getElementById("message").value,
    country: document.getElementById("country").value, news: document.getElementById("newsletter").checked,
    error: document.getElementById("error-email").textContent })`);
  check("the typed values and the error are in the page", r.name === "Bob" && r.message === "kept" && r.country === "us" && r.news && r.error === "That is not an email", JSON.stringify(r));
  check("the hydration agrees with the HTML (text fields, select, checkbox)", r.m === 0 && r.mut.childList === 0 && r.mut.attributes === 0, JSON.stringify(r));
  tab.close();
}

// ---------------------------------------------------------------------------
console.log("--- the requests: cancel, timeout, errors and upload progress");
{
  await reset();
  const tab = await openTab();
  await tab.goto(`${BASE}/`);
  await until(() => tab.evaluate(`document.getElementById("status")?.textContent === "interactive"`), "the hydration");
  const result = () => tab.evaluate(`document.getElementById("lab-result").textContent`);
  const click = (id) => tab.evaluate(`document.getElementById(${JSON.stringify(id)}).click()`);

  await click("lab-slow");
  check("a slow request is running", await until(async () => (await result()) === "running", "running"));
  await sleep(150);
  await click("lab-cancel");
  check("cancel stops it", await until(async () => (await result()) === "idle" || (await result()) === "canceled", "canceled"), await result());
  check("the browser really dropped the connection", await until(async () => (await stats())["slow-abandoned"] === 1, "the server to notice", 4000), JSON.stringify(await stats()));

  await click("lab-timeout");
  check("a request that takes too long ends as a timeout", await until(async () => (await result()) === "timeout", "timeout"), await result());

  await click("lab-missing");
  const missing = await until(async () => { const t = await result(); return t.includes("404") && t; }, "the 404");
  check("a 404 is an HTTPError with its status", missing.includes("404"), missing);

  await click("lab-network");
  const network = await until(async () => { const t = await result(); return t.includes("network error") && t; }, "the network error");
  check("an unreachable server is a network error", network.includes("network error"), network);

  await click("lab-upload");
  const uploaded = await until(async () => { const t = await result(); return t.startsWith("uploaded") && t; }, "the upload", 15000);
  const bytes = Number(/"bytes":\s*(\d+)/.exec(uploaded)?.[1]);
  check("3 MB reached the server", bytes >= 3 * 1024 * 1024, uploaded);
  check("the progress reached 100%", (await tab.evaluate(`document.getElementById("lab-progress").textContent`)) === "upload: 100%");
  check("no exceptions", !tab.logs.some((l) => l.includes("EXCEPTION")), tab.logs.join("\n"));
  tab.close();
}

console.log(failures === 0 ? "\nall good" : `\n${failures} failed`);
process.exit(failures === 0 ? 0 : 1);
