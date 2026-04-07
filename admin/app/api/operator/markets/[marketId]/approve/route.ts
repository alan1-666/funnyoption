import { NextResponse } from "next/server";

import {
  type SignedOperatorAction
} from "@/lib/operator-auth";
import { authorizeOperatorAction, forwardJSON, toCoreOperatorProof } from "@/lib/operator-server";

type ApproveRequestBody = {
  operator?: SignedOperatorAction;
};

export async function POST(request: Request, { params }: { params: Promise<{ marketId: string }> }) {
  const { marketId } = await params;
  const parsedMarketID = Number.parseInt(marketId, 10);
  if (!Number.isFinite(parsedMarketID) || parsedMarketID <= 0) {
    return NextResponse.json({ error: "invalid marketId" }, { status: 400 });
  }

  const payload = (await request.json().catch(() => null)) as ApproveRequestBody | null;
  if (!payload?.operator) {
    return NextResponse.json({ error: "operator payload is required" }, { status: 400 });
  }

  const message = `approve_market:${parsedMarketID}:${payload.operator.requestedAt}`;
  const auth = await authorizeOperatorAction(message, payload.operator);
  if (!auth.ok) {
    return auth.response;
  }

  const response = await forwardJSON(`/api/v1/admin/markets/${parsedMarketID}/approve`, {
    method: "POST",
    body: {
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
