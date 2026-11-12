/* eslint-disable no-undef */
const CACHE_NAME = "momshell-v2";
const APP_PATH = "/";

async function cacheAppShell(response) {
  if (!response.ok || !response.headers.get("Content-Type")?.includes("text/html")) return;

  const cache = await caches.open(CACHE_NAME);
  const html = await response.clone().text();
  const entries = Array.from(html.matchAll(/(?:src|href)="(\/assets\/[^" ]+\.(?:js|css))"/g), (match) => match[1]);
  await Promise.all(entries.map(async (url) => {
    if (!(await cache.match(url))) await cache.add(url);
  }));
  await cache.put(APP_PATH, response);
}

self.addEventListener("install", (event) => {
  event.waitUntil((async () => {
    const response = await fetch(APP_PATH, { cache: "reload" });
    await cacheAppShell(response);
    await self.skipWaiting();
  })());
});

self.addEventListener("activate", (event) => {
  event.waitUntil((async () => {
    const keys = await caches.keys();
    await Promise.all(keys.filter((key) => key.startsWith("momshell-") && key !== CACHE_NAME).map((key) => caches.delete(key)));
    await self.clients.claim();
  })());
});

self.addEventListener("fetch", (event) => {
  const { request } = event;
  const url = new URL(request.url);
  if (request.method !== "GET" || url.origin !== self.location.origin) return;

  if (request.mode === "navigate" && [APP_PATH, "/index.html"].includes(url.pathname)) {
    const network = fetch(request);
    event.waitUntil(network.then((response) => cacheAppShell(response.clone())).catch(() => {}));
    event.respondWith(network.catch(async () => (await caches.match(APP_PATH, { cacheName: CACHE_NAME })) || Response.error()));
    return;
  }

  if (url.pathname.startsWith("/assets/") && !request.headers.has("Range")) {
    const asset = caches.open(CACHE_NAME).then(async (cache) => {
      const cached = await cache.match(request);
      if (cached) return { response: cached };
      return { response: await fetch(request), cache };
    });
    event.waitUntil(asset.then(({ response, cache }) => {
      if (cache && response.status === 200 && !response.headers.get("Content-Type")?.includes("text/html")) {
        return cache.put(request, response.clone());
      }
    }).catch(() => {}));
    event.respondWith(asset.then(({ response }) => response));
  }
});
