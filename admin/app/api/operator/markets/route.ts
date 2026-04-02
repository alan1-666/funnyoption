import { NextResponse } from "next/server";

import {
  buildCreateMarketMessage,
  normalizeCreateMarketDraft,
  type CreateMarketDraft,
  type SignedOperatorAction
} from "@/lib/operator-auth";
import { authorizeOperatorAction, forwardJSON, operatorUserID, toCoreOperatorProof } from "@/lib/operator-server";

type CreateMarketRequestBody = {
  market?: CreateMarketDraft;
  operator?: SignedOperatorAction;
};

export async function POST(request: Request) {
  const payload = (await request.json().catch(() => null)) as CreateMarketRequestBody | null;
  if (!payload?.market || !payload?.operator) {
    return NextResponse.json({ error: "market and operator payloads are required" }, { status: 400 });
  }

  const draft = normalizeCreateMarketDraft(payload.market);
  if (!draft.title) {
    return NextResponse.json({ error: "title is required" }, { status: 400 });
  }

  const auth = await authorizeOperatorAction(
    buildCreateMarketMessage({
      walletAddress: payload.operator.walletAddress,
      market: draft,
      requestedAt: payload.operator.requestedAt
    }),
    payload.operator
  );
  if (!auth.ok) {
    return auth.response;
  }

  const response = await forwardJSON("/api/v1/markets", {
    method: "POST",
    body: {
      title: draft.title,
      description: draft.description,
      category_key: draft.categoryKey,
      collateral_asset: draft.collateralAsset,
      status: draft.status,
      open_at: draft.openAt,
      close_at: draft.closeAt,
      resolve_at: draft.resolveAt,
      created_by: operatorUserID(),
      cover_image_url: draft.coverImage,
      cover_source_url: draft.sourceUrl,
      cover_source_name: draft.sourceName,
      options: draft.options,
      metadata: {
        category: draft.categoryKey === "SPORTS" ? "体育" : "加密",
        categoryKey: draft.categoryKey,
        coverImage: draft.coverImage,
        sourceUrl: draft.sourceUrl,
        sourceSlug: draft.sourceSlug,
        sourceName: draft.sourceName,
        sourceKind: draft.sourceKind,
        yesOdds: 0.5,
        noOdds: 0.5
      },
      operator: toCoreOperatorProof(payload.operator)
    }
  });

  if (!response.ok) {
    return response.response;
  }

  return NextResponse.json(
    {
      ...(response.payload as Record<string, unknown>),
      operator_wallet_address: auth.walletAddress
    },
    { status: response.status }
  );
}
