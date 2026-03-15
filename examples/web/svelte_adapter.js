if (typeof globalThis.BAZIC_UI_RENDER !== "function") {
  globalThis.BAZIC_UI_RENDER = () => {};
}

function renderNode(node) {
  if (!node) return document.createTextNode("");
  if (node.t === "text") return document.createTextNode(String(node.v || ""));
  const el = document.createElement(node.tag || "div");
  const props = node.props || {};
  for (const [key, value] of Object.entries(props)) {
    if (value === undefined || value === null || value === false) continue;
    if (key === "class") {
      el.className = String(value);
      continue;
    }
    if (key === "id") {
      el.id = String(value);
      continue;
    }
    if (key === "value") {
      el.value = String(value);
      continue;
    }
    if (key === "checked") {
      el.checked = !!value;
      continue;
    }
    if (value === true) {
      el.setAttribute(key, "");
    } else {
      el.setAttribute(key, String(value));
    }
  }
  const children = Array.isArray(node.children) ? node.children : [];
  for (const child of children) {
    el.appendChild(renderNode(child));
  }
  return el;
}

globalThis.BAZIC_UI_RENDER = (jsonText) => {
  const root = document.getElementById("ui");
  if (!root) return;
  const tree = JSON.parse(jsonText);
  root.innerHTML = "";
  root.appendChild(renderNode(tree));
};
