import { NextResponse } from "next/server";

type PolymarketSource = {
  kind: "event" | "market";
  slug: string;
  title: string;
  description: string;
  category: string;
  coverImage: string;
  sourceUrl: string;
  sourceName: string;
  raw: Record<string, unknown>;
};

const POLYMARKET_API = "https://gamma-api.polymarket.com";
const POLYMARKET_SITE = "https://polymarket.com";

function extractSlug(input: string) {
  const trimmed = input.trim();
  if (!trimmed) return "";

  try {
    const parsed = new URL(trimmed);
    const parts = parsed.pathname.split("/").filter(Boolean);
    return parts.at(-1) ?? "";
  } catch {
    return trimmed.replace(/^\/+/, "").split(/[/?#]/).filter(Boolean).at(0) ?? "";
  }
}

function asObject(value: unknown) {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }
  return value as Record<string, unknown>;
}

function asString(value: unknown) {
  return typeof value === "string" ? value.trim() : "";
}

function pickImage(source: Record<string, unknown>) {
  const keys = [
    "image",
    "twitterCardImage",
    "featuredImage",
    "featured_image",
    "icon",
    "bannerImage",
    "banner_image",
    "sponsorImage",
    "sponsor_image"
  ];
  for (const key of keys) {
    const candidate = asString(source[key]);
    if (candidate) return candidate;
  }
  return "";
}

function pickTitle(source: Record<string, unknown>) {
  return (
    asString(source.title) ||
    asString(source.question) ||
    asString(source.name) ||
    asString(source.slug) ||
    "Polymarket market"
  );
}

function pickDescription(source: Record<string, unknown>) {
  return (
    asString(source.description) ||
    asString(source.subtitle) ||
    asString(source.question) ||
    asString(source.summary) ||
    "Imported from Polymarket."
  );
}

function pickCategory(source: Record<string, unknown>) {
  return (
    asString(source.category) ||
    asString(source.subcategory) ||
    asString(source.tagName) ||
    asString(source.groupItemTitle) ||
    asString(source.seriesTitle) ||
    "Polymarket"
  );
}

function normalizeSource(source: Record<string, unknown>, kind: "event" | "market"): PolymarketSource {
  const slug = asString(source.slug) || asString(source.marketSlug) || asString(source.questionSlug);
  const title = pickTitle(source);
  const description = pickDescription(source);
  const category = pickCategory(source);
  const coverImage = pickImage(source);
  const sourceUrl = `${POLYMARKET_SITE}/${kind}/${encodeURIComponent(slug || title.toLowerCase().replace(/\s+/g, "-"))}`;

  return {
    kind,
    slug: slug || title.toLowerCase().replace(/\s+/g, "-"),
    title,
    description,
    category,
    coverImage,
    sourceUrl,
    sourceName: "Polymarket",
    raw: source
  };
}

async function fetchJson(url: string) {
  const response = await fetch(url, {
    cache: "no-store",
    headers: {
      Accept: "application/json"
    }
  });

  if (response.status === 404) {
    return null;
  }

  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`);
  }

  return response.json();
}

async function fetchEvent(slug: string) {
  const payload = await fetchJson(`${POLYMARKET_API}/events/slug/${encodeURIComponent(slug)}`);
  if (!payload) {
    return null;
  }
  const source = asObject(payload);
  return source ? normalizeSource(source, "event") : null;
}

async function fetchMarket(slug: string) {
  const payload = await fetchJson(`${POLYMARKET_API}/markets/slug/${encodeURIComponent(slug)}`);
  if (!payload) {
    return null;
  }
  const source = asObject(payload);
  return source ? normalizeSource(source, "market") : null;
}

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const input = searchParams.get("input")?.trim() ?? "";

  if (!input) {
    return NextResponse.json({ error: "input is required" }, { status: 400 });
  }

  const slug = extractSlug(input);

  try {
    if (slug) {
      const event = await fetchEvent(slug);
      if (event) {
        return NextResponse.json(event);
      }

      const market = await fetchMarket(slug);
      if (market) {
        return NextResponse.json(market);
      }
    }

    return NextResponse.json({ error: "No Polymarket market found for the provided input" }, { status: 404 });
  } catch (error) {
    return NextResponse.json(
      {
        error: error instanceof Error ? error.message : "Failed to fetch Polymarket market"
      },
      { status: 502 }
    );
  }
}
