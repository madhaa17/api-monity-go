// Cloudflare Worker that relays the backend's Yahoo Finance requests, for servers whose IP Yahoo
// answers with 429. Deploy it, set the RELAY_TOKEN secret, then on the backend set
// STOCK_API_URL=https://<worker>.workers.dev and STOCK_API_RELAY_TOKEN=<same token>.
//
// Only the two Yahoo endpoints the backend uses are forwarded, and only with the right token,
// so the worker is not an open proxy.

const ALLOWED = [/^\/v8\/finance\/chart\/[^/]+$/, /^\/v1\/finance\/search$/];
const UPSTREAMS = ["https://query1.finance.yahoo.com", "https://query2.finance.yahoo.com"];
const UA =
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36";

export default {
  async fetch(request, env) {
    if (request.method !== "GET") return new Response("method not allowed", { status: 405 });
    if (!env.RELAY_TOKEN || request.headers.get("X-Relay-Token") !== env.RELAY_TOKEN) {
      return new Response("unauthorized", { status: 401 });
    }
    const url = new URL(request.url);
    if (!ALLOWED.some((re) => re.test(url.pathname))) {
      return new Response("not found", { status: 404 });
    }

    let resp;
    for (const base of UPSTREAMS) {
      resp = await fetch(base + url.pathname + url.search, {
        headers: { Accept: "application/json", "User-Agent": UA },
      });
      if (resp.status !== 429 && resp.status < 500) break; // else try the other Yahoo host
    }
    const headers = new Headers({ "Content-Type": resp.headers.get("Content-Type") || "application/json" });
    const retryAfter = resp.headers.get("Retry-After");
    if (retryAfter) headers.set("Retry-After", retryAfter);
    return new Response(resp.body, { status: resp.status, headers });
  },
};
