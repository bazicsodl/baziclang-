const path = require("path");
const vscode = require("vscode");
const { LanguageClient, TransportKind } = require("vscode-languageclient/node");

let client;

function activate(context) {
  const workspaceFolder = vscode.workspace.workspaceFolders && vscode.workspace.workspaceFolders[0];
  if (!workspaceFolder) {
    // Still allow activation for single-file mode.
  }
  const output = vscode.window.createOutputChannel("Bazic");
  output.appendLine("Bazic extension activating...");
  const setBazicLanguage = (doc) => {
    if (!doc || doc.languageId === "bazic") {
      return;
    }
    if (doc.uri && doc.uri.fsPath && doc.uri.fsPath.toLowerCase().endsWith(".bz")) {
      vscode.languages.setTextDocumentLanguage(doc, "bazic");
    }
  };
  vscode.workspace.textDocuments.forEach(setBazicLanguage);
  context.subscriptions.push(vscode.workspace.onDidOpenTextDocument(setBazicLanguage));
  const serverCwd = vscode.Uri.joinPath(context.extensionUri, "..", "..").fsPath;
  output.appendLine("LSP server cwd: " + serverCwd);
  const serverOptions = resolveServerOptions(serverCwd, output);
  const clientOptions = {
    documentSelector: [{ scheme: "file", language: "bazic" }],
    outputChannel: output
  };
  client = new LanguageClient(
    "baziclang",
    "Bazic Language Server",
    serverOptions,
    clientOptions
  );
  client.onReady().then(
    () => output.appendLine("Bazic LSP ready."),
    (err) => {
      output.appendLine("Bazic LSP failed to start: " + err);
      vscode.window.showErrorMessage("Bazic LSP failed to start. Check Output -> Bazic.");
    }
  );
  context.subscriptions.push(client.start());

  context.subscriptions.push(
    vscode.workspace.onWillSaveTextDocument(async (evt) => {
      if (evt.document.languageId !== "bazic") {
        return;
      }
      const cfg = vscode.workspace.getConfiguration("bazic", evt.document.uri);
      const autoFix = cfg.get("autoFixOnSave", true);
      const autoFormat = cfg.get("formatOnSave", true);
      if (!autoFix && !autoFormat) {
        return;
      }
      if (autoFormat) {
        await vscode.commands.executeCommand("editor.action.formatDocument");
      }
      if (autoFix) {
        await applyPreferredQuickFixes(evt.document, output);
      }
    })
  );
}

function deactivate() {
  if (!client) {
    return undefined;
  }
  return client.stop();
}

function resolveServerOptions(serverCwd, output) {
  const bazlspOnPath = "bazlsp";
  const bazlspLocal = path.join(serverCwd, "bin", "bazlsp.exe");
  const localLspSource = path.join(serverCwd, "cmd", "bazlsp");
  if (process.platform === "win32" && existsFile(bazlspLocal)) {
    output.appendLine("Using local bazlsp binary: " + bazlspLocal);
    return { command: bazlspLocal, args: [], options: { cwd: serverCwd } };
  }
  if (existsDir(localLspSource)) {
    output.appendLine("Using go run ./cmd/bazlsp");
    return { command: "go", args: ["run", "./cmd/bazlsp"], options: { cwd: serverCwd } };
  }
  output.appendLine("Using bazlsp from PATH if available.");
  return { command: bazlspOnPath, args: [], options: { cwd: serverCwd } };
}

function existsFile(p) {
  try {
    return require("fs").statSync(p).isFile();
  } catch (e) {
    return false;
  }
}

function existsDir(p) {
  try {
    return require("fs").statSync(p).isDirectory();
  } catch (e) {
    return false;
  }
}

async function applyPreferredQuickFixes(document, output) {
  try {
    const fullRange = new vscode.Range(
      document.positionAt(0),
      document.positionAt(document.getText().length)
    );
    const actions = await vscode.commands.executeCommand(
      "vscode.executeCodeActionProvider",
      document.uri,
      fullRange,
      { kind: "quickfix" }
    );
    if (!Array.isArray(actions)) {
      return;
    }
    for (const action of actions) {
      if (!action || !action.isPreferred || !action.title || !action.title.startsWith("Bazic:")) {
        continue;
      }
      if (action.edit && action.edit.changes) {
        const wsEdit = new vscode.WorkspaceEdit();
        for (const [uri, edits] of Object.entries(action.edit.changes)) {
          const docUri = vscode.Uri.parse(uri);
          edits.forEach((e) => {
            wsEdit.replace(
              docUri,
              new vscode.Range(
                new vscode.Position(e.range.start.line, e.range.start.character),
                new vscode.Position(e.range.end.line, e.range.end.character)
              ),
              e.newText
            );
          });
        }
        await vscode.workspace.applyEdit(wsEdit);
      }
    }
  } catch (e) {
    output.appendLine("Auto-fix failed: " + e);
  }
}

module.exports = {
  activate,
  deactivate
};
