// Optional React adapter stub (if you load React in the page).
// It expects a BAZIC UI JSON tree and renders it using React.createElement.
// Usage: include React/ReactDOM scripts and then this file, set globalThis.BAZIC_UI_ADAPTER="react".
if (typeof globalThis.BAZIC_UI_RENDER !== "function") {
  globalThis.BAZIC_UI_RENDER = () => {};
}

function toReact(node) {
  if (!node) return null;
  if (node.t === "text") return node.v || "";
  if (node.t === "elem") {
    const props = node.props || {};
    const children = Array.isArray(node.children) ? node.children.map(toReact) : [];
    return React.createElement(node.tag || "div", props, ...children);
  }
  return null;
}

globalThis.BAZIC_UI_RENDER = (jsonText) => {
  if (!globalThis.React || !globalThis.ReactDOM) return;
  const root = document.getElementById("ui");
  if (!root) return;
  const tree = JSON.parse(jsonText);
  const element = toReact(tree);
  ReactDOM.render(element, root);
};
