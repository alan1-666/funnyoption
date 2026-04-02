import { NextResponse } from "next/server";

import {
  buildResolveMarketMessage,
  normalizeResolveMarketDraft,
  type ResolveMarketDraft,
  type SignedOperatorAction
} from "@/lib/operator-auth";
import { authorizeOperatorAction, forwardJSON, toCoreOperatorProof } from "@/lib/operator-server";

type ResolveMarketRequestBody = {
  market?: ResolveMarketDraft;
  operator?: SignedOperatorAction;
};

export async function POST(request: Request, { params }: { params: Promise<{ marketId: string }> }) {
  const { marketId } = await params;
  const parsedMarketID = Number.parseInt(marketId, 10);
  if (!Number.isFinite(parsedMarketID) || parsedMarketID <= 0) {
    return NextResponse.json({ error: "invalid marketId" }, { status: 400 });
  }

  const payload = (await request.json().catch(() => null)) as ResolveMarketRequestBody | null;
  if (!payload?.market || !payload?.operator) {
    return NextResponse.json({ error: "market and operator payloads are required" }, { status: 400 });
  }

  const draft = normalizeResolveMarketDraft({
    ...payload.market,
    marketId: parsedMarketID
  });

  const auth = await authorizeOperatorAction(
    buildResolveMarketMessage({
      walletAddress: payload.operator.walletAddress,
      market: draft,
      requestedAt: payload.operator.requestedAt
    }),
    payload.operator
  );
  if (!auth.ok) {
    return auth.response;
  }

  const response = await forwardJSON(`/api/v1/markets/${draft.marketId}/resolve`, {
    method: "POST",
    body: {
      outcome: draft.outcome,
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
