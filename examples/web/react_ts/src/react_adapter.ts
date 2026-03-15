declare const React: any;
declare const ReactDOM: any;

type UINode = {
  t: string;
  v?: string;
  tag?: string;
  props?: Record<string, unknown>;
  children?: UINode[];
};

if (typeof (globalThis as any).BAZIC_UI_RENDER !== "function") {
  (globalThis as any).BAZIC_UI_RENDER = () => {};
}

function toReact(node: UINode | null): any {
  if (!node) return null;
  if (node.t === "text") return node.v || "";
  if (node.t === "elem") {
    const props = node.props || {};
    const children = Array.isArray(node.children) ? node.children.map(toReact) : [];
    return React.createElement(node.tag || "div", props, ...children);
  }
  return null;
}

(globalThis as any).BAZIC_UI_RENDER = (jsonText: string) => {
  if (!React || !ReactDOM) return;
  const root = document.getElementById("ui");
  if (!root) return;
  const tree = JSON.parse(jsonText) as UINode;
  const element = toReact(tree);
  ReactDOM.render(element, root);
};
