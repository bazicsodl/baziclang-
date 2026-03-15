const output = document.getElementById("output");
const statusEl = document.getElementById("status");
const runBtn = document.getElementById("run");
const clearBtn = document.getElementById("clear");
const uiRoot = document.getElementById("ui");

function setStatus(msg) {
  statusEl.textContent = msg;
}

function append(line) {
  output.textContent += line + "\n";
  output.scrollTop = output.scrollHeight;
}

function applyProps(el, props) {
  if (!props) return;
  if (props.id) el.id = String(props.id);
  if (props.class) el.className = String(props.class);
  if (props.key) el.dataset.key = String(props.key);
  for (const [key, value] of Object.entries(props)) {
    if (key === "id" || key === "class" || key === "key" || key === "value" || key === "checked") continue;
    if (value === undefined || value === null || value === false) {
      el.removeAttribute(key);
    } else if (value === true) {
      el.setAttribute(key, "");
    } else {
      el.setAttribute(key, String(value));
    }
  }
}

function renderNode(node) {
  if (!node) return document.createTextNode("");
  if (node.t === "text") {
    return document.createTextNode(String(node.v || ""));
  }
  if (node.t === "elem") {
    const el = document.createElement(node.tag || "div");
    const props = node.props || {};
    applyProps(el, props);
    if (node.tag === "a") {
      el.href = "#";
    }
    if (node.tag === "input") {
      if (props.type) el.type = String(props.type);
      if (props.type === "checkbox" && props.checked !== undefined) {
        el.checked = !!props.checked;
      }
      if (props.value !== undefined) {
        const next = String(props.value);
        if (el.value !== next) {
          el.value = next;
        }
      }
    }
    if (node.tag === "select" && props.value !== undefined) {
      const next = String(props.value);
      if (el.value !== next) {
        el.value = next;
      }
    }
    const children = Array.isArray(node.children) ? node.children : [];
    for (const child of children) {
      el.appendChild(renderNode(child));
    }
    return el;
  }
  return document.createTextNode("");
}

function patchNode(el, node) {
  if (!node) {
    return null;
  }
  if (node.t === "text") {
    const text = String(node.v || "");
    if (el && el.nodeType === Node.TEXT_NODE) {
      if (el.nodeValue !== text) el.nodeValue = text;
      return el;
    }
    return document.createTextNode(text);
  }
  if (node.t === "elem") {
    const tag = node.tag || "div";
    if (!el || el.nodeType !== Node.ELEMENT_NODE || el.tagName.toLowerCase() !== tag) {
      return renderNode(node);
    }
    const props = node.props || {};
    applyProps(el, props);
    if (tag === "a") {
      el.href = "#";
    }
    if (tag === "input") {
      if (props.type) el.type = String(props.type);
      if (props.type === "checkbox" && props.checked !== undefined) {
        el.checked = !!props.checked;
      }
      if (props.value !== undefined) {
        const next = String(props.value);
        if (el.value !== next) el.value = next;
      }
    }
    if (tag === "select" && props.value !== undefined) {
      const next = String(props.value);
      if (el.value !== next) el.value = next;
    }
    const nextChildren = Array.isArray(node.children) ? node.children : [];
    const keyed = new Map();
    for (const child of el.childNodes) {
      if (child.nodeType === Node.ELEMENT_NODE && child.dataset && child.dataset.key) {
        keyed.set(child.dataset.key, child);
      }
    }
    const max = Math.max(el.childNodes.length, nextChildren.length);
    for (let i = 0; i < max; i++) {
      const next = nextChildren[i];
      const key = next && next.props && next.props.key ? String(next.props.key) : "";
      const child = key && keyed.has(key) ? keyed.get(key) : el.childNodes[i];
      if (!next) {
        if (child) el.removeChild(child);
        continue;
      }
      const patched = patchNode(child, next);
      if (!child) {
        el.appendChild(patched);
      } else if (patched !== child) {
        el.replaceChild(patched, child);
      }
    }
    return el;
  }
  return el || document.createTextNode("");
}

globalThis.BAZIC_UI_RENDER = (jsonText) => {
  if (!uiRoot) return;
  if (globalThis.BAZIC_UI_LAST === jsonText) {
    return;
  }
  globalThis.BAZIC_UI_LAST = jsonText;
  try {
    const tree = JSON.parse(jsonText);
    const current = uiRoot.firstChild;
    const patched = patchNode(current, tree);
    if (!current && patched) {
      uiRoot.appendChild(patched);
    } else if (patched && patched !== current) {
      uiRoot.replaceChild(patched, current);
    }
  } catch (e) {
    uiRoot.textContent = "UI JSON parse error";
  }
};

const originalLog = console.log;
console.log = (...args) => {
  append(args.join(" "));
  originalLog(...args);
};

async function runBazic() {
  output.textContent = "";
  setStatus("loading wasm...");
  const go = new Go();
  const now = new Date().toISOString();
  go.argv = ["bazic", "web", now];
  go.env = Object.assign({}, go.env || {}, {
    BAZIC_MESSAGE: "hello from js",
    BAZIC_TS: now
  });
  const routeFromHash = () => {
    const hash = window.location.hash || "";
    let route = hash.replace(/^#/, "");
    if (route.startsWith("/")) {
      route = route.slice(1);
    }
    if (route === "" || route === "/") return "home";
    return route;
  };
  globalThis.BAZIC_WEB = {
    _data: new Map(),
    get(key) {
      const k = String(key);
      if (this._data.has(k)) return this._data.get(k);
      try {
        const v = localStorage.getItem(k);
        if (v !== null) return v;
      } catch (e) {}
      return undefined;
    },
    set(key, value) {
      const k = String(key);
      const v = String(value);
      this._data.set(k, v);
      try {
        localStorage.setItem(k, v);
      } catch (e) {}
      if (k === "focus") {
        const el = document.getElementById(v);
        if (el && typeof el.focus === "function") {
          el.focus();
        }
      }
      if (k === "route") {
        const next = v === "home" ? "#/" : "#/" + v;
        if (window.location.hash !== next) {
          window.location.hash = next;
        }
      }
      if (String(key) === "ui" && typeof globalThis.BAZIC_UI_RENDER === "function") {
        try {
          globalThis.BAZIC_UI_RENDER(String(value));
        } catch (e) {
          append("UI render error: " + String(e));
        }
      }
      return true;
    }
  };
  globalThis.BAZIC_WEB.set("route", routeFromHash());
  window.addEventListener("hashchange", () => {
    const route = routeFromHash();
    globalThis.BAZIC_WEB.set("route", route);
    globalThis.BAZIC_WEB.set("event", "route");
    globalThis.BAZIC_WEB.set("event_type", "route");
    globalThis.BAZIC_WEB.set("event_target", "hash");
    globalThis.BAZIC_WEB.set("event_value", route);
  });
  globalThis.BAZIC_WEB.set("payload", JSON.stringify({ ok: true, ts: now, name: "bazic" }));
  if (uiRoot) {
    const emitEvent = (type, targetId, value, action) => {
      globalThis.BAZIC_WEB.set("event", type);
      globalThis.BAZIC_WEB.set("event_type", type);
      globalThis.BAZIC_WEB.set("event_target", String(targetId || ""));
      globalThis.BAZIC_WEB.set("event_value", String(value || ""));
      if (action !== undefined) {
        globalThis.BAZIC_WEB.set("event_action", String(action || ""));
      }
    };
    uiRoot.addEventListener("click", (e) => {
      const target = e.target;
      if (!target) return;
      const actionEl = target.closest ? target.closest("[data-action]") : null;
      if (actionEl) {
        emitEvent("action", actionEl.getAttribute("data-action"), actionEl.id || "", actionEl.getAttribute("data-action"));
        if (actionEl.getAttribute("data-stop") === "1") {
          e.stopPropagation();
        }
        return;
      }
      if (!target.id) return;
      if (target.id === "btn") {
        emitEvent("click", "btn", "");
      }
      if (target.id === "focus-note") {
        emitEvent("click", "focus-note", "");
      }
      if (target.id === "form-submit") {
        emitEvent("click", "form-submit", "");
      }
      if (target.id === "toast-btn") {
        emitEvent("click", "toast-btn", "");
      }
      if (target.id === "toast-clear") {
        emitEvent("click", "toast-clear", "");
      }
      if (target.id === "tab-overview") {
        emitEvent("click", "tab-overview", "");
      }
      if (target.id === "tab-stats") {
        emitEvent("click", "tab-stats", "");
      }
      if (target.id === "tab-alerts") {
        emitEvent("click", "tab-alerts", "");
      }
      if (target.id === "nav-home" || target.id === "nav-about" || target.id === "nav-components" || target.id === "nav-dashboard" || target.id === "nav-form") {
        emitEvent("nav", target.id, target.id);
        return;
      }
      if (target.id === "table-empty" || target.id === "table-sort-reset" || target.id === "sort-latency" || target.id === "sort-status") {
        emitEvent("click", target.id, "");
        return;
      }
      if (target.id === "menu-toggle" || target.id === "menu-starter" || target.id === "menu-team" || target.id === "menu-enterprise") {
        emitEvent("click", target.id, "");
        return;
      }
      emitEvent("click", target.id, "");
    });
    const emitInput = (target) => {
      if (!target || !target.id) return;
      const type = target.type || "";
      const value = type === "checkbox" ? String(!!target.checked) : String(target.value || "");
      let handled = false;
      if (target.id === "name") {
        emitEvent("input", "name", value);
        handled = true;
      }
      if (target.id === "note") {
        emitEvent("input", "note", value);
        handled = true;
      }
      if (target.id === "form-email") {
        emitEvent("input", "form-email", value);
        handled = true;
      }
      if (target.id === "form-company") {
        emitEvent("input", "form-company", value);
        handled = true;
      }
      if (target.id === "plan") {
        emitEvent("input", "plan", value);
        handled = true;
      }
      if (target.id === "opt-in") {
        emitEvent("input", "opt-in", value);
        handled = true;
      }
      if (target.id === "dark-mode") {
        emitEvent("input", "dark-mode", value);
        handled = true;
      }
      if (target.id === "volume") {
        emitEvent("input", "volume", value);
        handled = true;
      }
      if (!handled) {
        emitEvent("input", target.id, value);
      }
    };
    uiRoot.addEventListener("input", (e) => {
      emitInput(e.target);
    });
    uiRoot.addEventListener("change", (e) => {
      emitInput(e.target);
    });
  }
  let resp = await fetch("app.wasm");
  if (!resp.ok) {
    resp = await fetch("../../app.wasm");
  }
  const buffer = await resp.arrayBuffer();
  const { instance } = await WebAssembly.instantiate(buffer, go.importObject);
  setStatus("running");
  try {
    await go.run(instance);
  } finally {
    setStatus("done");
  }
}

runBtn.addEventListener("click", () => {
  runBazic().catch(err => {
    setStatus("error");
    append(String(err));
  });
});

clearBtn.addEventListener("click", () => {
  output.textContent = "";
  setStatus("idle");
});
