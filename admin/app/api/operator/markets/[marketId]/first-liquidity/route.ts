import { NextResponse } from "next/server";

import {
  buildBootstrapMarketMessage,
  normalizeBootstrapMarketDraft,
  type BootstrapMarketDraft,
  type SignedOperatorAction
} from "@/lib/operator-auth";
import { authorizeOperatorAction, forwardJSON, toCoreOperatorProof } from "@/lib/operator-server";

type BootstrapMarketRequestBody = {
  bootstrap?: BootstrapMarketDraft;
  operator?: SignedOperatorAction;
};

export async function POST(request: Request, { params }: { params: Promise<{ marketId: string }> }) {
  const { marketId } = await params;
  const parsedMarketID = Number.parseInt(marketId, 10);
  if (!Number.isFinite(parsedMarketID) || parsedMarketID <= 0) {
    return NextResponse.json({ error: "invalid marketId" }, { status: 400 });
  }

  const payload = (await request.json().catch(() => null)) as BootstrapMarketRequestBody | null;
  if (!payload?.bootstrap || !payload?.operator) {
    return NextResponse.json({ error: "bootstrap and operator payloads are required" }, { status: 400 });
  }

  const draft = normalizeBootstrapMarketDraft({
    ...payload.bootstrap,
    marketId: parsedMarketID
  });
  if (draft.userId <= 0 || draft.quantity <= 0 || draft.price <= 0) {
    return NextResponse.json({ error: "user_id, quantity, and price must be positive" }, { status: 400 });
  }

  const auth = await authorizeOperatorAction(
    buildBootstrapMarketMessage({
      walletAddress: payload.operator.walletAddress,
      bootstrap: draft,
      requestedAt: payload.operator.requestedAt
    }),
    payload.operator
  );
  if (!auth.ok) {
    return auth.response;
  }

  const firstLiquidity = await forwardJSON(`/api/v1/admin/markets/${draft.marketId}/first-liquidity`, {
    method: "POST",
    body: {
      user_id: draft.userId,
      quantity: draft.quantity,
      outcome: draft.outcome,
      price: draft.price,
      operator: toCoreOperatorProof(payload.operator)
    }
  });
  if (!firstLiquidity.ok) {
    const firstLiquidityPayload = firstLiquidity.payload as Record<string, unknown>;
    return NextResponse.json(
      {
        ...firstLiquidityPayload,
        operator_wallet_address: auth.walletAddress
      },
      { status: firstLiquidity.status }
    );
  }

  const firstLiquidityPayload = firstLiquidity.payload as Record<string, unknown>;

  return NextResponse.json(
    {
      ...firstLiquidityPayload,
      market_id: firstLiquidityPayload.market_id ?? draft.marketId,
      user_id: firstLiquidityPayload.user_id ?? draft.userId,
      operator_wallet_address: auth.walletAddress
    },
    { status: firstLiquidity.status }
  );
}
