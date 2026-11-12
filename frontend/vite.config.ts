import { defineConfig, loadEnv } from "vite";
import vue from "@vitejs/plugin-vue";
import react from "@vitejs/plugin-react";
import { randomBytes } from "node:crypto";
import { fileURLToPath, URL } from "node:url";

export default defineConfig(({ command, isPreview, mode }) => {
  const projectRoot = fileURLToPath(new URL("..", import.meta.url));
  const { PORT } = loadEnv(mode, projectRoot, "PORT");
  const backend = `http://127.0.0.1:${PORT || "8000"}`;
  const devNonce = command === "serve" && !isPreview ? randomBytes(16).toString("base64") : undefined;

  return {
    plugins: [
      vue(),
      react({ include: /\.(tsx|jsx)$/ }),
      devNonce ? {
        name: "momshell-dev-csp",
        transformIndexHtml(html: string) {
          return html.replace("script-src 'self'", `script-src 'self' 'nonce-${devNonce}'`);
        },
      } : false,
    ],
    html: { cspNonce: devNonce },
    resolve: {
      alias: {
        "@": fileURLToPath(new URL("./src", import.meta.url)),
      },
    },
    server: {
      host: "127.0.0.1",
      proxy: {
        "/api": backend,
        "/uploads": backend,
        "/admin": backend,
        "/health": backend,
      },
    },
  };
});
