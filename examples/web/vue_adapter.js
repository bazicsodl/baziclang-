if (typeof globalThis.BAZIC_UI_RENDER !== "function") {
  globalThis.BAZIC_UI_RENDER = () => {};
}

function toVNode(node, h) {
  if (!node) return null;
  if (node.t === "text") return node.v || "";
  const props = node.props || {};
  const children = Array.isArray(node.children) ? node.children.map((child) => toVNode(child, h)) : [];
  return h(node.tag || "div", props, children);
}

globalThis.BAZIC_UI_RENDER = (jsonText) => {
  if (!globalThis.Vue) return;
  const tree = JSON.parse(jsonText);
  const root = document.getElementById("ui");
  if (!root) return;
  const app = Vue.createApp({
    render() {
      return toVNode(tree, Vue.h);
    }
  });
  root.innerHTML = "";
  app.mount(root);
};
